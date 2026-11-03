package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 本文件负责「菜单裁剪之后，把快捷键动作表也对齐」。
//
// 背景：internal/shortcut 的动作表把页面 id 写死了（nav.task → "task"、
// nav.new-project → "new-project"…）。新项目勾掉「工作台」分组后这些页面没了，
// 动作却还在，表现是：
//   - 设置页的快捷键列表里仍列着「打开异步任务」，按 Ctrl+2 什么都没发生；
//   - 默认键位出现空洞（Ctrl+1、Ctrl+4、Ctrl+5），看起来像坏了；
//   - 用户改绑这些动作也毫无意义。
//
// 所以生成时必须按「真正被裁掉的页面」删掉对应动作，并把剩下的导航类动作默认键位
// 重排成 cmd+1..cmd+N。页面 id 的真源是 cutRegistry 的 AST 结果（RemovedIDs），
// 这里不再二次解析 registry.go——那会把 settingsNav（不在分组字面量里）这类页面
// 误判成不存在。

var (
	// 动作表一行里的各字段（动作表每行一个字面量，gofmt 保证不换行）。
	actionIDRe   = regexp.MustCompile(`\{ID:\s*"([^"]+)"`)
	actionPageRe = regexp.MustCompile(`Page:\s*"([^"]+)"`)
	actionDefRe  = regexp.MustCompile(`Def:\s*"([^"]*)"`)
)

// pruneShortcuts 删掉指向「已不存在页面」的导航动作，并重排导航键位。
//
// removedPages 是被裁掉的页面 id 集合（见 cutRegistry 的 RemovedIDs）。返回被删掉
// 的动作 id（用于在生成结果里说明）。找不到动作表时不改文件：那说明模板结构变了，
// 宁可不改也不能把文件写坏。
func pruneShortcuts(root string, removedPages map[string]bool) ([]string, error) {
	if len(removedPages) == 0 {
		return nil, nil
	}
	path := filepath.Join(root, "internal", "shortcut", "shortcut.go")
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil // 文件都不在 = 模板结构变了，按「找不到动作表」处理
	}
	text := string(src)

	start := strings.Index(text, "var Actions = []Action{")
	if start < 0 {
		return nil, nil
	}
	bodyStart := start + len("var Actions = []Action{")
	end := strings.Index(text[bodyStart:], "\n}")
	if end < 0 {
		return nil, nil
	}
	end += bodyStart

	var (
		kept    []string
		removed []string
		navSeq  int
		changed bool
	)
	for _, line := range strings.Split(text[bodyStart:end], "\n") {
		trimmed := strings.TrimSpace(line)
		idM := actionIDRe.FindStringSubmatch(trimmed)
		if idM == nil {
			kept = append(kept, line) // 空行、注释原样保留
			continue
		}
		pageM := actionPageRe.FindStringSubmatch(trimmed)
		if pageM != nil && removedPages[pageM[1]] {
			removed = append(removed, idM[1])
			changed = true
			continue
		}
		if pageM != nil {
			// 导航类：默认键位重排成 cmd+1..N，避免 cmd+1、cmd+5 这种空洞。
			// 只重排「原本有默认键」的动作；默认不绑定的保持不绑定。
			if defM := actionDefRe.FindStringSubmatch(trimmed); defM != nil && defM[1] != "" {
				navSeq++
				if def := fmt.Sprintf("cmd+%d", navSeq); def != defM[1] {
					line = actionDefRe.ReplaceAllLiteralString(line, `Def: "`+def+`"`)
					changed = true
				}
			}
		}
		kept = append(kept, line)
	}
	if !changed || len(kept) == 0 {
		// 没有动作被删、键位也没变化（比如模板已经裁好），保持文件原样。
		return removed, nil
	}

	newText := text[:bodyStart] + "\n" + strings.Join(kept, "\n") + text[end:]
	if err := os.WriteFile(path, []byte(newText), 0o644); err != nil {
		return removed, err
	}
	return removed, nil
}
