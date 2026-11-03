package view

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
)

// TestShortcutRebind 改绑后按新键生效、旧键失效。
func TestShortcutRebind(t *testing.T) {
	a := newTestApp()
	a.settings.Shortcuts = map[string]string{"nav.task": "cmd+9"}
	a.rebuildShortcuts()

	tt := render(a)
	tt.Image()

	tt.Key(ui.Cmd, ui.Key2) // 旧键：已改绑，不应生效
	tt.Image()
	if a.page == "task" {
		t.Fatal("旧键位不应再触发（nav.task 已改到 cmd+9）")
	}

	tt.Key(ui.Cmd, ui.Key9) // 新键
	tt.Image()
	if a.page != "task" {
		t.Fatalf("新键位应打开任务页，实际 %q", a.page)
	}
}

// TestShortcutConflictRejected 冲突的组合不能被接受。
func TestShortcutConflictRejected(t *testing.T) {
	a := newTestApp()
	// nav.home 默认 cmd+1，nav.task 默认 cmd+2：把 home 改到 cmd+2 应冲突。
	if err := a.setShortcut("nav.home", ui.Cmd, ui.Key2); err == nil {
		t.Fatal("与 nav.task 冲突的组合应被拒绝")
	}
	if err := a.setShortcut("nav.home", ui.Cmd, ui.Key6); err != nil {
		t.Fatalf("无冲突的组合应被接受: %v", err)
	}
	if got := a.settings.Shortcuts["nav.home"]; got != "cmd+6" {
		t.Fatalf("应写入 cmd+6，实际 %q", got)
	}
}

// TestShortcutRejectsEscape Esc 保留给关闭弹窗。
func TestShortcutRejectsEscape(t *testing.T) {
	a := newTestApp()
	if err := a.setShortcut("nav.home", 0, ui.KeyEscape); err == nil {
		t.Fatal("Esc 不应被允许绑定")
	}
}

// TestShortcutReset 恢复默认清掉覆盖。
func TestShortcutReset(t *testing.T) {
	a := newTestApp()
	if err := a.setShortcut("nav.task", ui.Cmd, ui.Key9); err != nil {
		t.Fatal(err)
	}
	a.resetShortcuts()
	if len(a.settings.Shortcuts) != 0 {
		t.Fatalf("恢复默认应清空覆盖，实际 %v", a.settings.Shortcuts)
	}
	if got := a.shortcutLabel("nav.task"); got == "未设置" {
		t.Fatal("恢复默认后 nav.task 应有默认键位")
	}
}

// TestShortcutDisableByEmpty 配置成空串 = 禁用该动作。
func TestShortcutDisableByEmpty(t *testing.T) {
	a := newTestApp()
	a.settings.Shortcuts = map[string]string{"nav.task": ""}
	a.rebuildShortcuts()
	if _, ok := a.shortcuts["nav.task"]; ok {
		t.Fatal("空串应禁用动作")
	}
	if got := a.shortcutLabel("nav.task"); got != "未设置" {
		t.Fatalf("禁用后应显示「未设置」，实际 %q", got)
	}
}

// TestShortcutSectionRenders 设置页要有快捷键区块与各动作名。
func TestShortcutSectionRenders(t *testing.T) {
	a := newTestApp()
	a.page = "settings"
	tt := render(a)
	mustHave(t, tt, "快捷键", "打开首页", "打开异步任务", "刷新当前页", "恢复默认")
}

// TestShortcutCaptureRebinds 录制态：按下组合键即改绑，且这一帧不触发全局快捷键。
func TestShortcutCaptureRebinds(t *testing.T) {
	a := newTestApp()
	a.page = "settings"
	a.shortcutCapture = "view.refresh" // 直接进入录制态（等价于点了「修改」）
	tt := render(a)
	tt.Image() // 让捕获按钮 AutoFocus

	tt.Key(ui.Cmd, ui.Key9)
	tt.Image()

	if got := a.settings.Shortcuts["view.refresh"]; got != "cmd+9" {
		t.Fatalf("录制后应写入 cmd+9，实际 %q", got)
	}
	if a.shortcutCapture != "" {
		t.Fatal("录制成功后应退出录制态")
	}
	// 录制期间按的 cmd+9 不应顺带触发页面切换（nav.* 里没有 cmd+9，这里主要防止 panic/串扰）。
	if a.page != "settings" {
		t.Fatalf("录制期间不应切页，实际 %q", a.page)
	}
}

// TestShortcutCaptureEscapeCancels 录制态按 Esc 取消。
func TestShortcutCaptureEscapeCancels(t *testing.T) {
	a := newTestApp()
	a.page = "settings"
	a.shortcutCapture = "nav.home"
	tt := render(a)
	tt.Image()

	tt.Key(0, ui.KeyEscape)
	tt.Image()
	if a.shortcutCapture != "" {
		t.Fatal("Esc 应退出录制态")
	}
	if _, ok := a.settings.Shortcuts["nav.home"]; ok {
		t.Fatal("取消不应写入任何绑定")
	}
}

// TestShortcutPersistedToSettings 改绑要写进 settings（重启后仍生效）。
func TestShortcutPersistedToSettings(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	a := newTestApp()
	if err := a.setShortcut("view.refresh", ui.Cmd|ui.Shift, ui.KeyR); err != nil {
		t.Fatal(err)
	}
	a.FlushSettings()

	s, err := settings.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Shortcuts["view.refresh"]; got != "cmd+shift+r" {
		t.Fatalf("落盘后应能读回 cmd+shift+r，实际 %q", got)
	}
}
