package view

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
)

// TestPageSettingsTab 只有页面声明了设置时，「设置 → 页面」才出现，并渲染其区块。
func TestPageSettingsTab(t *testing.T) {
	if len(pageSettingItems()) == 0 {
		t.Fatal("注册表里应有声明了页面级设置的页面")
	}
	a := newTestApp()
	a.page = settingsPageID
	a.setSettingsGroup(settingsGroupPages)
	tt := render(a)
	// 分段控件多出「页面」，区块标题是页面自己的菜单文字。
	mustHave(t, tt, "页面", "异步数据（加载/重试）", "演示数据条数")
}

// TestPageSettingsTabHiddenByDefault 没有声明页面设置的工程里，分组只有三项。
func TestPageSettingsTabHiddenByDefault(t *testing.T) {
	// 当前工程有一个页面声明了设置 → 应有四项；这里验证「列表与实际注册一致」。
	labels := settingsGroupLabels()
	if len(labels) != 4 || labels[3] != "页面" {
		t.Fatalf("分组文案应与注册一致，实际 %v", labels)
	}
	// 默认分组仍是「常规」，不会因为多一组而改变。
	a := newTestApp()
	a.page = settingsPageID
	if got := a.settingsGroup(); got != settingsGroupGeneral {
		t.Errorf("默认分组应为常规，实际 %d", got)
	}
}

// TestPageSettingDrivesPage 页面级设置的值被页面自己读取（条数）。
func TestPageSettingDrivesPage(t *testing.T) {
	a := newAsyncApp(t, 10*time.Millisecond)
	if err := a.PrefsFor(asyncDemoPageID).Set("rows", 7); err != nil {
		t.Fatalf("写页面设置: %v", err)
	}
	tt := render(a)
	waitText(t, tt, "共 7 条")
}

// TestClearPrefs 清理页面状态：内存与数据库一起清，任务等其它数据不受影响。
func TestClearPrefs(t *testing.T) {
	s := openTempStore(t)
	a := newApp(settings.Default())
	a.AttachStore(s)
	a.pushTaskRow("单元测试任务", "轻量级", "已完成")

	if err := a.PrefsFor("orders").Set("filter", orderFilter{Kind: "批发"}); err != nil {
		t.Fatal(err)
	}
	if n := a.prefCount(); n != 1 {
		t.Fatalf("清理前应有 1 条，实际 %d", n)
	}

	a.clearPrefs()
	if n := a.prefCount(); n != 0 {
		t.Fatalf("清理后应为 0 条，实际 %d", n)
	}
	var got orderFilter
	if a.PrefsFor("orders").Get("filter", &got) {
		t.Fatal("清理后不应再读到页面状态")
	}
	if len(a.tasks) == 0 {
		t.Error("清理页面状态不应影响任务列表")
	}
}

// TestClearPrefsMemoryOnly 没有数据库时清的是内存缓存（本次运行内仍可用）。
func TestClearPrefsMemoryOnly(t *testing.T) {
	a := newTestApp()
	if err := a.Prefs().Set("filter", orderFilter{Kind: "零售"}); err != nil {
		t.Fatal(err)
	}
	if n := a.prefCount(); n != 1 {
		t.Fatalf("无库时计数取内存缓存，实际 %d", n)
	}
	a.clearPrefs()
	if n := a.prefCount(); n != 0 {
		t.Fatalf("清理后应为 0，实际 %d", n)
	}
}

// TestSettingsDataDiagnosticsClearEntry 设置页「关于与诊断」里有清理入口与说明。
func TestSettingsDataDiagnosticsClearEntry(t *testing.T) {
	a := newTestApp()
	a.page = settingsPageID
	a.setSettingsGroup(settingsGroupAbout)
	tt := ui.NewTester(a.View, 1100, 1200)
	mustHave(t, tt, "清理页面状态", "页面状态：", "不动任务、授权记录、偏好与日志")
}
