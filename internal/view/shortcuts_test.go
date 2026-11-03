package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestShortcutPageJump 验证 Ctrl+2 跳到异步任务页。
//
// mygo 的 Shortcut 是「上帧登记、本帧匹配」：先渲染一帧把快捷键登记上，
// 再发按键，然后渲染下一帧让 View 读到匹配结果。
func TestShortcutPageJump(t *testing.T) {
	a := newTestApp()
	tt := render(a)
	tt.Image() // 登记快捷键

	if a.page != "home" {
		t.Fatalf("初始页应为 home，实际 %q", a.page)
	}
	tt.Key(ui.Cmd, ui.Key2)
	tt.Image()
	if a.page != "task" {
		t.Fatalf("Ctrl+2 应跳到 task，实际 %q", a.page)
	}

	// Ctrl+5 = 设置。
	tt.Key(ui.Cmd, ui.Key5)
	tt.Image()
	if a.page != "settings" {
		t.Fatalf("Ctrl+5 应跳到 settings，实际 %q", a.page)
	}
}

// TestShortcutEscapeClosesOverlay 验证 Esc 关闭最上层弹窗。
func TestShortcutEscapeClosesOverlay(t *testing.T) {
	a := newTestApp()
	a.showBasicInfo = true
	tt := render(a)
	tt.Image()

	tt.Key(0, ui.KeyEscape)
	tt.Image()
	if a.showBasicInfo {
		t.Fatal("Esc 应关闭基本信息弹窗")
	}

	// 没有弹窗时按 Esc 不应 panic，也不应改变页面。
	page := a.page
	tt.Key(0, ui.KeyEscape)
	tt.Image()
	if a.page != page {
		t.Fatalf("无弹窗时 Esc 不应切页（%q → %q）", page, a.page)
	}
}

// TestShortcutRefreshPage 验证 F5 重新进入当前页（进入次数 +1）。
func TestShortcutRefreshPage(t *testing.T) {
	a := newTestApp()
	tt := render(a)
	tt.Image()

	a.page = "task"
	a.syncPages() // 模拟切到任务页那一帧（真实运行时由 View 每帧调用）
	before := a.pages["task"].enters

	tt.Key(0, ui.KeyF5) // 按键会触发一帧 View，F5 在 handleShortcuts 里生效
	if got := a.pages["task"].enters; got != before+1 {
		t.Fatalf("F5 后 enters 应为 %d，实际 %d", before+1, got)
	}
}
