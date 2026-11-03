package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// 模板白名单：只有这些内容会被带进新项目。
//
// 取舍依据是「新项目跑起来需要什么」：
//   - 代码与资源：internal/、resources/、main.go、go.mod/go.sum
//   - 构建与发布：mygo.json、scripts/、.github/workflows/
//   - 工程习惯：.gitignore、.gitattributes、.agents/（mygo 维护技能）
//
// 明确**不带**：docs/（本项目的历史笔记）、CHANGELOG.md / README.md / RELEASING.md
// （都是 bestuav-mygo 自己的内容，新项目会重新生成）、docs/screenshots（截图属于
// 本项目）、internal/scaffold（生成器自身不该出现在产物里）。
var (
	// keepExact 顶层必须原样带走的文件。
	//
	// go.mod / go.sum 在**快照**里以 .snapshot 后缀存放：go:embed 拒绝嵌入「含
	// go.mod 的目录」（会把它当成另一个 module）。复制时统一改回原名。
	keepExact = map[string]bool{
		"go.mod":          true,
		"go.sum":          true,
		"go.mod.snapshot": true,
		"go.sum.snapshot": true,
		"mygo.json":       true,
		"main.go":         true,
		".gitignore":      true,
		".gitattributes":  true,
	}
	// keepPrefix 需要整目录带走的相对路径前缀。
	keepPrefix = []string{
		"internal/",
		"scripts/",
		"resources/",
		".github/",
		".agents/",
	}
	// dropPrefix 即使落在上面某个前缀里也要丢掉的路径。
	dropPrefix = []string{
		"internal/scaffold/", // 生成器自身
		"assets/",            // 本项目素材（各项目自带）
	}
	// dropExact 单个要丢掉的路径。
	dropExact = map[string]bool{
		// 生成器自己的菜单页：它依赖 internal/scaffold。
		"internal/view/page_newproject.go":      true,
		"internal/view/page_newproject_test.go": true,
		// 快照刷新脚本只对「本仓库」有意义，新项目里没有 internal/scaffold。
		"scripts/snapshot-template.py": true,
	}
)

// templateFS 返回模板根文件系统。
//
// 优先级：Options.SourceDir（本地源码，永远最新）> 编译期快照（exe 自带兜底）。
// 返回值第二项是来源说明，会进 Result.Warnings，让用户知道用的是哪一份。
func templateFS(sourceDir string) (fs.FS, string, error) {
	if sourceDir != "" {
		if err := checkSourceDir(sourceDir); err != nil {
			return nil, "", err
		}
		return os.DirFS(sourceDir), "本地源码目录 " + sourceDir, nil
	}
	sub, err := fs.Sub(embeddedTemplate, "testdata/template")
	if err != nil {
		return nil, "", fmt.Errorf("内置模板不可用: %w", err)
	}
	return sub, "exe 内置模板快照", nil
}

// checkSourceDir 确认目录确实是一个 bestuav-mygo 工程。
func checkSourceDir(dir string) error {
	for _, name := range []string{"mygo.json", "go.mod", "internal/view/registry.go"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); err != nil {
			return fmt.Errorf("模板源目录 %s 缺少 %s，看起来不是 bestuav-mygo 工程", dir, name)
		}
	}
	return nil
}

// copyTemplate 按白名单把模板复制到 dst，返回复制的文件数。
func copyTemplate(tpl fs.FS, dst string) (int, error) {
	count := 0
	err := fs.WalkDir(tpl, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		if !wantPath(p) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, filepath.FromSlash(p)), 0o755)
		}
		data, rerr := fs.ReadFile(tpl, p)
		if rerr != nil {
			return fmt.Errorf("读取模板文件 %s: %w", p, rerr)
		}
		target := filepath.Join(dst, filepath.FromSlash(outputName(p)))
		if mkErr := os.MkdirAll(filepath.Dir(target), 0o755); mkErr != nil {
			return mkErr
		}
		if wErr := os.WriteFile(target, data, 0o644); wErr != nil {
			return fmt.Errorf("写入 %s: %w", p, wErr)
		}
		count++
		return nil
	})
	return count, err
}

// wantPath 判断模板内的相对路径是否属于白名单。
//
// 注意目录自身的写法没有结尾斜杠（"internal" 而不是 "internal/"）：只按前缀匹配会
// 让整个目录被 SkipDir 跳过，表现为「顶层文件都在、internal 一个都没有」。
func wantPath(p string) bool {
	if dropExact[p] {
		return false
	}
	for _, drop := range dropPrefix {
		if strings.HasPrefix(p, drop) || p == strings.TrimSuffix(drop, "/") {
			return false
		}
	}
	if keepExact[p] {
		return true
	}
	for _, pre := range keepPrefix {
		if strings.HasPrefix(p, pre) || p == strings.TrimSuffix(pre, "/") {
			// 模板快照自身不能再被复制进来（快照位于 internal/scaffold/testdata 下，
			// 已被 dropPrefix 拦掉，这里只是双保险）。
			if strings.HasPrefix(p, "internal/scaffold") {
				return false
			}
			return true
		}
	}
	return false
}

// embeddedTemplate 是编译期快照，由 scripts/snapshot-template.py 刷新。
//
// 放在 testdata 下是刻意的：go 工具链会忽略 testdata 目录，里面的 .go 文件不会
// 参与编译；而 go:embed 仍能读到它们。
//
//go:embed all:testdata/template
var embeddedTemplate embed.FS

// templateFileExts 会被当作「待重写源码」处理的扩展名（大小写不敏感）。
var templateFileExts = map[string]bool{
	".go":   true,
	".md":   true,
	".json": true,
	".yml":  true,
	".yaml": true,
	".ps1":  true,
	".sh":   true,
	".py":   true,
}

// outputName 把快照里的占位名还原成真实文件名（目前只有 go.mod / go.sum 需要）。
func outputName(p string) string {
	if strings.HasSuffix(p, ".snapshot") {
		return strings.TrimSuffix(p, ".snapshot")
	}
	return p
}

// isRewritableSource 判断某个相对路径是否需要在复制后做文本重写。
func isRewritableSource(p string) bool {
	return templateFileExts[strings.ToLower(path.Ext(p))]
}
