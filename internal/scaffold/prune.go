package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// pruneRemovedPages 删除「只服务于被移除菜单」的页面文件。
//
// 为什么需要这一步：只从注册表里摘掉菜单项，页面文件与其依赖仍然留在工程里，
// 新项目会背着一堆看不见的演示代码（以及它们的依赖与编译时间）。
//
// 为什么必须逐个删 + 每次编译：页面文件之间并不是彼此独立的。实测踩过的两个坑：
//   - `page_home.go` 里的 `greeting()` 被欢迎页调用，函数级判定「homePage 已移除」
//     成立，但整份文件不能删；
//   - `page_demo_features.go` 定义的一批类型被 `App` 结构体字段引用，删掉当场编不过。
//
// 所以这里不做「先全删再回滚」的批量操作，而是**一次只删一个、删完立刻编译该包**，
// 编不过就原样写回。代价是 N 次编译（每份文件约 1 秒），换来的是绝不可能交出一个
// 编不过的工程——这个取舍对一次性生成任务是划算的。
func pruneRemovedPages(root string, funcs []string) (removed, kept []string, err error) {
	if len(funcs) == 0 {
		return nil, nil, nil
	}
	viewDir := filepath.Join(root, "internal", "view")

	// 1) 找出定义这些函数的文件，并带上同名测试文件。
	candidates := map[string]bool{}
	for _, fn := range funcs {
		file, ferr := fileDefiningFunc(viewDir, fn)
		if ferr != nil {
			return nil, nil, ferr
		}
		if file == "" {
			continue
		}
		candidates[file] = true
		if _, statErr := os.Stat(strings.TrimSuffix(file, ".go") + "_test.go"); statErr == nil {
			candidates[strings.TrimSuffix(file, ".go")+"_test.go"] = true
		}
	}

	// 2) 注册表里还有引用的文件直接排除（两组共用同一页面的情况）。
	regData, rerr := os.ReadFile(filepath.Join(viewDir, "registry.go"))
	if rerr != nil {
		return nil, nil, rerr
	}
	prefix := filepath.Join(root) + string(filepath.Separator)
	names := make([]string, 0, len(candidates))
	for file := range candidates {
		if stillUsedByRegistry(regData, file) {
			kept = append(kept, relTo(root, file))
			continue
		}
		names = append(names, file)
	}
	sortStrings(names)

	// 3) 逐个尝试删除。
	for _, file := range names {
		data, berr := os.ReadFile(file)
		if berr != nil {
			return removed, kept, berr
		}
		if rmerr := os.Remove(file); rmerr != nil {
			return removed, kept, rmerr
		}
		if buildErr := goBuildPkg(root, "./internal/view/"); buildErr != nil {
			// 这个文件还被别处依赖：写回并记为保留。
			if werr := os.WriteFile(file, data, 0o644); werr != nil {
				return removed, kept, werr
			}
			kept = append(kept, strings.TrimPrefix(file, prefix))
			continue
		}
		removed = append(removed, strings.TrimPrefix(file, prefix))
	}
	return removed, kept, nil
}

// stillUsedByRegistry 判断裁剪后的注册表里是否仍然引用该文件定义的页面函数。
func stillUsedByRegistry(regData []byte, file string) bool {
	text := string(regData)
	for _, fn := range appMethodsInFile(file) {
		if strings.Contains(text, "App)."+fn) {
			return true
		}
	}
	return false
}

// appMethodsInFile 列出文件里所有 `func (a *App) xxx(` 的方法名。
func appMethodsInFile(file string) []string {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	re := regexp.MustCompile(`(?m)^func \(a \*App\) ([A-Za-z0-9_]+)\(`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(string(data), -1) {
		out = append(out, m[1])
	}
	return out
}

// fileDefiningFunc 找定义 `func (a *App) fn(` 的文件，返回绝对路径（找不到返回空串）。
func fileDefiningFunc(dir, fn string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`(?m)^func \(a \*App\) ` + regexp.QuoteMeta(fn) + `\(`)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return "", rerr
		}
		if re.Match(data) {
			return p, nil
		}
	}
	return "", nil
}

// goBuild 在 root 里跑 `go build ./...`（最终自检）。
//
// 用 go build 而不是 go vet：vet 会连带跑静态检查，在生成的工程里可能因为风格问题
// 报警，那不是我们要验的「能不能编译」。
func goBuild(root string) error { return goBuildPkg(root, "./...") }

// goBuildPkg 编译指定包模式，失败时返回带输出的错误。
func goBuildPkg(root, pattern string) error {
	cmd := exec.Command("go", "build", pattern)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build %s 失败: %v\n%s", pattern, err, truncate(string(out), 2000))
	}
	return nil
}

// relTo 返回相对 root 的斜杠路径，便于展示。
func relTo(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(rel)
}

// truncate 截断过长输出（错误提示要能塞进 UI）。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n…（输出已截断）"
}

// sortStrings 简单插入排序（候选文件数量很小，不引 sort 包更省心）。
func sortStrings(in []string) {
	for i := 1; i < len(in); i++ {
		for j := i; j > 0 && in[j] < in[j-1]; j-- {
			in[j], in[j-1] = in[j-1], in[j]
		}
	}
}

// demoPkgCandidates 只服务于「演示页」的子包。页面被裁掉后它们通常就没人引用了，
// 但删之前必须编译验证——例如保留「工作台」时 internal/license 仍被 license 页引用。
var demoPkgCandidates = []string{
	"internal/view/blackhole", // GPU 黑洞特效页
	"internal/view/effort",    // 自绘投入滑块页
	"internal/view/piano",     // 合成音钢琴页
	"internal/view/fx",        // 彩带特效
	"internal/imgtest",        // 只给测试用的像素取样工具
	"internal/license",        // 生成 license 页的签发逻辑
}

// pruneDemoPackages 整包删除「已无人引用」的演示子包，返回删掉的相对路径。
//
// 做法：把目录改名成 `.xxx.stash`（go 会忽略以点开头的目录）→ `go build ./...`
// → 编不过就改回来。这比逐个删文件省事得多，也保证不会删出编译错误。
func pruneDemoPackages(root string) (removed []string, err error) {
	for _, rel := range demoPkgCandidates {
		dir := filepath.Join(root, filepath.FromSlash(rel))
		info, statErr := os.Stat(dir)
		if statErr != nil || !info.IsDir() {
			continue
		}
		stash := filepath.Join(filepath.Dir(dir), "."+filepath.Base(dir)+".stash")
		_ = os.RemoveAll(stash)
		if renameErr := os.Rename(dir, stash); renameErr != nil {
			return removed, renameErr
		}
		if buildErr := goBuildPkg(root, "./..."); buildErr != nil {
			if backErr := os.Rename(stash, dir); backErr != nil {
				return removed, backErr
			}
			continue
		}
		if rmErr := os.RemoveAll(stash); rmErr != nil {
			return removed, rmErr
		}
		removed = append(removed, rel)
	}
	return removed, nil
}
