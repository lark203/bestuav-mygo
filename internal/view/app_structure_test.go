package view

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestAppHasNoPageOwnedFields 守卫「App 不该随页面增长」这条架构约束。
//
// 背景：演示页的状态原本都挂在 App 结构体上（bh / pi / tb / todos …），而 App 是任何
// 项目都必须保留的文件。结果是「删掉一个页面文件」会连带编不过——生成器要靠砍页面来
// 瘦身，就被这层耦合卡死了。把状态迁进 pageState.data 之后，页面文件自包含，删整页
// 就等于彻底移除功能。
//
// 这条测试守住回归：**App 里不许出现「定义在页面文件里的类型」字段**。新增页面时正确
// 的写法是
//
//	st := pageStateOf(a, "my-page", "state", newMyState)
//
// 而不是往 App 上加字段。
func TestAppHasNoPageOwnedFields(t *testing.T) {
	// 少数几个「页面形状、但被持久化层共用」的状态：它们不是单个页面的私有数据，
	// 而是应用级数据（会落库），迁移反而会把持久化逻辑搅乱，故显式豁免并写明原因。
	allowed := map[string]string{
		"lic":   "授权表单与历史记录：persist.go 读写数据库，不属于单页私有状态",
		"tasks": "任务列表：page_home.go / page_task.go / persist.go 三方共用，且落库",
	}

	pageTypes, err := pageOwnedTypes()
	if err != nil {
		t.Fatal(err)
	}

	fields, err := appFieldTypes()
	if err != nil {
		t.Fatal(err)
	}

	var bad []string
	for _, f := range fields {
		owner, ok := pageTypes[f.Type]
		if !ok {
			continue
		}
		if reason, ok := allowed[f.Name]; ok {
			_ = reason
			continue
		}
		bad = append(bad, f.Name+" "+f.Type+"（定义于 "+owner+"）")
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Errorf("App 结构体里出现了页面私有类型的字段，会让「删除该页面文件」直接编不过：\n  %s\n"+
			"改法：把状态挪进 pageState.data，用 pageStateOf(a, 页面id, 键, 构造函数) 取；"+
			"若确实需要跨页面持久化，请在 allowed 里登记并写明原因。",
			strings.Join(bad, "\n  "))
	}
}

// pageOwnedTypes 收集 internal/view/page_*.go 里定义的类型：类型名 → 文件名。
func pageOwnedTypes() (map[string]string, error) {
	dir := "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "page_") ||
			!strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if perr != nil {
			return nil, perr
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				out[ts.Name.Name] = name
			}
		}
	}
	return out, nil
}

// typeField App 结构体里的一个字段。
type typeField struct {
	Name string
	Type string // 去掉指针/切片/映射后最内层的类型名
}

// appFieldTypes 解析 app.go 里 App 结构体各字段的类型名。
func appFieldTypes() ([]typeField, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "app.go", nil, 0)
	if err != nil {
		return nil, err
	}
	var out []typeField
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "App" {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range st.Fields.List {
				base := baseTypeName(field.Type)
				if base == "" {
					continue
				}
				for _, n := range field.Names {
					if n.Name == "_" {
						continue
					}
					out = append(out, typeField{Name: n.Name, Type: base})
				}
			}
		}
	}
	return out, nil
}

// baseTypeName 剥掉 *、[]、map[...]、chan 等外壳，返回最内层的类型名。
//
// 只关心「是不是页面里定义的类型」，所以像 map[string]*noteItem 这种也要能取到
// noteItem。
func baseTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return baseTypeName(e.X)
	case *ast.ArrayType:
		return baseTypeName(e.Elt)
	case *ast.MapType:
		// map[K]V：键通常是基础类型，值才是业务类型。
		if v := baseTypeName(e.Value); v != "" {
			return v
		}
		return baseTypeName(e.Key)
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.ParenExpr:
		return baseTypeName(e.X)
	default:
		return ""
	}
}
