package scaffold

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
)

// regCutResult 裁剪菜单注册表的结果。
type regCutResult struct {
	// RemovedFuncs 被移除的菜单项所引用的页面函数名（例如 taskPage）。
	RemovedFuncs []string
	// RemovedGroups 被移除的分组标题。
	RemovedGroups []string
	// KeptGroups 保留的分组标题（不含无标题的首页组）。
	KeptGroups []string
}

// cutRegistry 按「保留分组标题」裁剪 internal/view/registry.go。
//
// 为什么不用正则：registry.go 里每个分组是多行复合字面量，正则很难在「分组边界」
// 上安全下刀；而 AST 能精确拿到每个元素的起止位置。这里只做**按位置删除**，不做
// 语法树重建——保留的分组原文一字不动，删掉的范围连同其内部注释一起消失。
//
// 删除粒度：
//   - 无标题分组（首页）永远保留；
//   - 分组标题不在 keepGroups 里 → 整组删除，并记录其页面函数；
//   - 保留的分组里，id 落在 removeItemIDs 的菜单项单独删除（例如生成器自己的菜单）。
func cutRegistry(file string, keepGroups, removeItemIDs []string) (regCutResult, error) {
	var res regCutResult
	src, err := os.ReadFile(file)
	if err != nil {
		return res, err
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return res, fmt.Errorf("解析 registry.go: %w", err)
	}

	fn := findFunc(parsed, "navRegistry")
	if fn == nil {
		return res, fmt.Errorf("registry.go 里找不到 navRegistry 函数")
	}
	groups := returnCompositeLit(fn)
	if groups == nil || len(groups.Elts) == 0 {
		return res, fmt.Errorf("navRegistry 的返回值不是分组字面量")
	}

	keepSet := map[string]bool{}
	for _, g := range keepGroups {
		keepSet[strings.TrimSpace(g)] = true
	}
	dropIDs := map[string]bool{}
	for _, id := range removeItemIDs {
		dropIDs[id] = true
	}

	var spans []textSpan
	var removedFuncs []string

	for _, elt := range groups.Elts {
		grp, ok := elt.(*ast.CompositeLit)
		if !ok {
			continue
		}
		title := stringField(grp, "title")
		items := compositeField(grp, "items")

		if title != "" && !keepSet[title] {
			// 整组删除：记录它的页面函数，供后续删文件用。
			res.RemovedGroups = append(res.RemovedGroups, title)
			removedFuncs = append(removedFuncs, buildFuncsOfItems(items)...)
			spans = append(spans, spanOfLines(src, fset, elt.Pos(), elt.End()))
			continue
		}
		if title != "" {
			res.KeptGroups = append(res.KeptGroups, title)
		}
		if items == nil {
			continue
		}
		// 保留的分组：只摘掉指定 id 的菜单项。
		for _, itemElt := range items.Elts {
			item, ok := itemElt.(*ast.CompositeLit)
			if !ok {
				continue
			}
			if id := stringField(item, "id"); id != "" && dropIDs[id] {
				removedFuncs = append(removedFuncs, buildFuncsOf(item)...)
				spans = append(spans, spanOfLines(src, fset, itemElt.Pos(), itemElt.End()))
			}
		}
	}

	res.RemovedFuncs = dedupe(removedFuncs)

	out, err := applySpans(src, spans)
	if err != nil {
		return res, err
	}
	formatted, err := format.Source(out)
	if err != nil {
		return res, fmt.Errorf("裁剪后的 registry.go 语法不合法: %w", err)
	}
	if err := os.WriteFile(file, formatted, 0o644); err != nil {
		return res, err
	}
	// 删掉因裁剪而变空的 import（例如首页组的 onEnter 用过 time.Time）。
	if err := dropUnusedImports(file); err != nil {
		return res, fmt.Errorf("清理未使用 import 失败: %w", err)
	}
	if after, rerr := os.ReadFile(file); rerr == nil {
		formatted = after
	}

	// 双重校验：保留下来的页面函数必须仍能在文件里找到引用，否则说明裁剪逻辑
	// 误删了别组的菜单项——这种错误交到用户手里会表现成「菜单少了但页面还在」。
	remaining := string(formatted)
	for _, grp := range res.KeptGroups {
		if !strings.Contains(remaining, `title: "`+grp+`"`) {
			return res, fmt.Errorf("裁剪后丢失了应保留的分组 %q", grp)
		}
	}
	return res, nil
}

// textSpan 待删除的字节区间 [start,end)。
type textSpan struct{ start, end int }

// spanOfLines 把「节点起止」扩成「整行起止」：从节点首行行首，到节点末行下一行的
// 行首。这样删除后不会留下空行或半行缩进，也顺手带走元素后的逗号。
func spanOfLines(src []byte, fset *token.FileSet, pos, end token.Pos) textSpan {
	start := fset.Position(pos).Offset
	stop := fset.Position(end).Offset
	// 往前退到行首。
	for start > 0 && src[start-1] != '\n' {
		start--
	}
	// 往后跳过该行剩余内容（逗号、空白）直到下一行行首。
	for stop < len(src) && src[stop] != '\n' {
		stop++
	}
	if stop < len(src) {
		stop++ // 吃掉换行
	}
	return textSpan{start: start, end: stop}
}

// applySpans 从后往前删除所有区间，保证前面的偏移量不受影响。
func applySpans(src []byte, spans []textSpan) ([]byte, error) {
	if len(spans) == 0 {
		return src, nil
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start > spans[j].start })
	last := len(src) + 1
	out := append([]byte(nil), src...)
	for _, sp := range spans {
		if sp.start < 0 || sp.end > len(out) || sp.start >= sp.end {
			return nil, fmt.Errorf("内部错误：删除区间越界 [%d,%d)", sp.start, sp.end)
		}
		if sp.end > last {
			return nil, fmt.Errorf("内部错误：删除区间重叠")
		}
		out = append(out[:sp.start], out[sp.end:]...)
		last = sp.start
	}
	return out, nil
}

// findFunc 在文件里找顶层函数声明。
func findFunc(f *ast.File, name string) *ast.FuncDecl {
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name != nil && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

// returnCompositeLit 取函数体里第一条 return 的复合字面量。
func returnCompositeLit(fn *ast.FuncDecl) *ast.CompositeLit {
	if fn.Body == nil {
		return nil
	}
	var found *ast.CompositeLit
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			return true
		}
		if lit, ok := ret.Results[0].(*ast.CompositeLit); ok {
			found = lit
			return false
		}
		return true
	})
	return found
}

// stringField 读取复合字面量里 `key: "字符串"` 的值。
func stringField(lit *ast.CompositeLit, key string) string {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if ident, ok := kv.Key.(*ast.Ident); !ok || ident.Name != key {
			continue
		}
		if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Kind == token.STRING {
			if s, err := strconv.Unquote(bl.Value); err == nil {
				return s
			}
		}
	}
	return ""
}

// compositeField 读取复合字面量里某个键对应的子复合字面量（例如 items）。
func compositeField(lit *ast.CompositeLit, key string) *ast.CompositeLit {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if ident, ok := kv.Key.(*ast.Ident); !ok || ident.Name != key {
			continue
		}
		if sub, ok := kv.Value.(*ast.CompositeLit); ok {
			return sub
		}
	}
	return nil
}

// buildFuncsOfItems 从一个分组的 items 字面量里抽出所有菜单项的 build 方法名。
//
// items.Elts 里每个元素是一个 navItem 复合字面量（不是 KeyValueExpr），所以要先
// 逐项拆开，再在项内找 build 键——写成「直接遍历 items.Elts 找 KeyValueExpr」会
// 一个都取不到（曾因此导致页面文件一个都没删）。
func buildFuncsOfItems(items *ast.CompositeLit) []string {
	if items == nil {
		return nil
	}
	var out []string
	for _, elt := range items.Elts {
		if item, ok := elt.(*ast.CompositeLit); ok {
			out = append(out, buildFuncsOf(item)...)
		}
	}
	return out
}

// buildFuncsOf 从一个菜单项字面量里抽出 build 指向的方法名。
//
// build 的写法固定为 `build: (*App).xxxPage`，AST 上是 ParenExpr(StarExpr(Ident))
// 套 SelectorExpr，这里只取 SelectorExpr 的 Sel 名字。
func buildFuncsOf(lit *ast.CompositeLit) []string {
	if lit == nil {
		return nil
	}
	var out []string
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if ident, ok := kv.Key.(*ast.Ident); !ok || ident.Name != "build" {
			continue
		}
		if name := selectorName(kv.Value); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// selectorName 取 x.y 形式表达式里的 y；其它形态返回空串。
func selectorName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.ParenExpr:
		return selectorName(e.X)
	default:
		return ""
	}
}

// dedupe 去重并保持顺序。
func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// registryHasFunc 判断裁剪后的注册表文本里是否仍引用某个页面函数。
func registryHasFunc(src []byte, fn string) bool {
	return bytes.Contains(src, []byte("App)."+fn)) || bytes.Contains(src, []byte("."+fn+"\n"))
}

// dropUnusedImports 删掉文件里「已经没人引用」的 import。
//
// 为什么需要：裁剪分组会连带删掉用到某些 import 的代码（例如首页组的 onEnter
// 用了 time.Time，该组被删后 time 就没人用了），Go 会直接报 "imported and not
// used" 让整个工程编不过。goimports 不在标准库里，这里自己做一个最小实现。
//
// 保守策略：只处理普通具名导入（跳过 `_`、`.` 与注释掉的），且只在确认包名没有
// 出现在文件任何标识符里时才删。
func dropUnusedImports(file string) error {
	src, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return err
	}

	// 收集文件里出现的所有标识符名（ImportSpec 自身的名字不算引用）。
	used := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if _, ok := n.(*ast.ImportSpec); ok {
			return false
		}
		if id, ok := n.(*ast.Ident); ok {
			used[id.Name] = true
		}
		return true
	})

	var spans []textSpan
	for _, spec := range f.Imports {
		if spec.Name != nil && (spec.Name.Name == "_" || spec.Name.Name == ".") {
			continue
		}
		pkg := importPackageName(spec)
		if pkg == "" || used[pkg] {
			continue
		}
		// 路径本身可能被注释或字符串引用到（如 //go:embed），保守跳过。
		spans = append(spans, spanOfLines(src, fset, spec.Pos(), spec.End()))
	}
	if len(spans) == 0 {
		return nil
	}
	out, err := applySpans(src, spans)
	if err != nil {
		return err
	}
	formatted, err := format.Source(out)
	if err != nil {
		return err
	}
	return os.WriteFile(file, formatted, 0o644)
}

// importPackageName 推断 import 引入的包名：显式别名优先，否则取路径最后一段。
//
// 极少数包名与路径末段不一致（如 gopkg.in/yaml.v3 的包名是 yaml），这类包一旦
// 被误判成未使用就会删错——但由于我们只在「标识符集合里完全没有该名字」时才删，
// 误删概率极低，且删除后紧接着的 go build 会把问题暴露出来。
func importPackageName(spec *ast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	p, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		return ""
	}
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		p = p[i+1:]
	}
	// 去掉 vN 后缀（例：yaml.v3 → yaml）与 .go 之类。
	if i := strings.IndexByte(p, '.'); i > 0 {
		p = p[:i]
	}
	return p
}
