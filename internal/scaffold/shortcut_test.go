package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sampleActions 与 internal/shortcut/shortcut.go 的动作表保持同构（生成器裁剪的
// 就是这份表；字段含义见 shortcut.Action）。
const sampleActions = `package shortcut

var Actions = []Action{
	{ID: "nav.home", Label: "打开首页", Page: "home", Def: "cmd+1"},
	{ID: "nav.task", Label: "打开异步任务", Page: "task", Def: "cmd+2"},
	{ID: "nav.license", Label: "打开生成 license", Page: "gen-license", Def: "cmd+3"},
	{ID: "nav.new-project", Label: "打开新建项目", Page: "new-project", Def: "cmd+4"},
	{ID: "nav.settings", Label: "打开设置", Page: "settings", Def: "cmd+5"},
	{ID: "view.refresh", Label: "刷新当前页", Refresh: true, Def: "f5"},
	{ID: "view.toggle-sidebar", Label: "收起/展开侧栏", ToggleSidebar: true, Def: "cmd+b"},
}
`

func writeActions(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "internal", "shortcut", "shortcut.go")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(sampleActions), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestPruneShortcuts 删掉失效动作、重排导航键位；非导航动作与注释/空行原样保留。
func TestPruneShortcuts(t *testing.T) {
	dir := t.TempDir()
	p := writeActions(t, dir)

	// 「工作台」被裁掉（task、gen-license），生成器自身菜单永远被摘除（new-project）。
	removed, err := pruneShortcuts(dir, map[string]bool{"task": true, "gen-license": true, "new-project": true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(removed, ",") != "nav.task,nav.license,nav.new-project" {
		t.Errorf("被删动作 = %v，期望 [nav.task nav.license nav.new-project]", removed)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, gone := range []string{"nav.task", "nav.license", "nav.new-project"} {
		if strings.Contains(text, gone) {
			t.Errorf("动作 %s 应被删除", gone)
		}
	}
	// 保留的导航动作键位重排成连续的 cmd+1..N。
	for _, want := range []string{
		`{ID: "nav.home", Label: "打开首页", Page: "home", Def: "cmd+1"},`,
		`{ID: "nav.settings", Label: "打开设置", Page: "settings", Def: "cmd+2"},`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("缺少重排后的行 %q\n实际：\n%s", want, text)
		}
	}
	// 非导航动作不受键位重排影响。
	if !strings.Contains(text, `{ID: "view.refresh", Label: "刷新当前页", Refresh: true, Def: "f5"},`) {
		t.Errorf("view.refresh 应原样保留\n实际：\n%s", text)
	}
	// 侧栏开关是窗口级动作：不指向页面，任何分组被裁掉都不该删它（否则新项目里
	// 侧栏就只剩鼠标一条路）。
	if !strings.Contains(text, `{ID: "view.toggle-sidebar", Label: "收起/展开侧栏", ToggleSidebar: true, Def: "cmd+b"},`) {
		t.Errorf("view.toggle-sidebar 应原样保留\n实际：\n%s", text)
	}

	// 幂等：再跑一次（无新增删除）不应改动文件。
	before, _ := os.ReadFile(p)
	if removed, err = pruneShortcuts(dir, map[string]bool{"new-project": true}); err != nil {
		t.Fatal(err)
	}
	if len(removed) != 0 {
		t.Errorf("第二次运行不应再删动作，实际删了 %v", removed)
	}
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Error("没有删除与键位变化时不应重写文件")
	}
}

// TestPruneShortcutsNoop 没有页面被裁时不碰动作表；模板结构变了（找不到动作表）
// 也不报错、不写坏文件。
func TestPruneShortcutsNoop(t *testing.T) {
	dir := t.TempDir()
	p := writeActions(t, dir)
	before, _ := os.ReadFile(p)

	if removed, err := pruneShortcuts(dir, nil); err != nil || removed != nil {
		t.Errorf("空删除集应直接返回 nil，实际 removed=%v err=%v", removed, err)
	}
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Error("空删除集不应改动文件")
	}

	// 动作表不存在（模板结构变化）：静默跳过。
	empty := t.TempDir()
	if removed, err := pruneShortcuts(empty, map[string]bool{"task": true}); err != nil || removed != nil {
		t.Errorf("缺动作表时应返回 nil,nil，实际 removed=%v err=%v", removed, err)
	}
}
