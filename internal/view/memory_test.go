package view

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
)

// TestClearPrefsDropsPageCaches 「清理页面状态」不只清 page_pref，还要丢其它页面的
// 运行时缓存（这正是之前「看不出效果」的原因），当前页保持不动。
func TestClearPrefsDropsPageCaches(t *testing.T) {
	s := openTempStore(t)
	a := newApp(settings.Default())
	a.AttachStore(s)
	a.page = "settings"

	if err := a.PrefsFor("orders").Set("filter", orderFilter{Kind: "批发"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// 造两个页面的运行时缓存。
	a.pages["orders"] = &pageState{born: time.Now(), lastSeen: time.Now(), data: map[string]any{"cache": 1}}
	a.pages["task"] = &pageState{born: time.Now(), lastSeen: time.Now(), data: map[string]any{"cache": 2}}

	if n := a.pageStateCount(); n != 2 {
		t.Fatalf("清理前应有 2 个页面带运行时缓存，实际 %d", n)
	}
	a.clearPrefs()

	if n := a.prefCount(); n != 0 {
		t.Errorf("page_pref 应被清空，实际还剩 %d 条", n)
	}
	if n := a.pageStateCount(); n != 0 {
		t.Errorf("其它页面的运行时缓存应被丢弃，实际还剩 %d", n)
	}
	if a.prefMem != nil {
		t.Error("内存里的页面偏好缓存也应清空")
	}
	if _, ok := a.pages["orders"]; ok {
		t.Error("被清理页面的状态条目应已删除")
	}
}

// TestReleaseMemory 手动/自动释放内存入口不 panic，并返回一个有限的归还量。
func TestReleaseMemory(t *testing.T) {
	a := newApp(settings.Default())
	got := a.releaseMemory("test")
	if math.IsNaN(got) || math.IsInf(got, 0) {
		t.Fatalf("归还量应是有限数值，实际 %v", got)
	}
}

// TestViewportSettleReleasesMemory 整体缩放停止后应自动触发一次内存回收。
func TestViewportSettleReleasesMemory(t *testing.T) {
	a := newApp(settings.Default())
	tt := ui.NewTester(a.View, 1100, 700)
	tt.Frame()
	tt.SetZoom(1.4) // 模拟窗口整体缩放
	tt.Frame()

	deadline := time.Now().Add(3 * time.Second)
	for a.viewportReleases.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if a.viewportReleases.Load() == 0 {
		t.Fatal("缩放停止后应自动回收一次内存")
	}
}

// TestRebuildRenderCacheTwoStep 「重建渲染缓存」要两次点击才执行，且会清页面缓存后重启。
func TestRebuildRenderCacheTwoStep(t *testing.T) {
	a := newApp(settings.Default())
	a.pages["orders"] = &pageState{born: time.Now(), lastSeen: time.Now(), data: map[string]any{"cache": 1}}

	called := 0
	oldRelaunch := relaunchApp
	relaunchApp = func() { called++ }
	defer func() { relaunchApp = oldRelaunch }()

	a.onRebuildClicked()
	if called != 0 {
		t.Fatal("第一次点击不应直接重启")
	}
	if a.confirmRebuildAt.IsZero() {
		t.Fatal("第一次点击应进入待确认状态")
	}

	a.onRebuildClicked()
	if called != 1 {
		t.Fatalf("第二次点击应重启一次，实际 %d", called)
	}
	if len(a.pages) != 0 {
		t.Errorf("页面运行时缓存应被清空，实际还剩 %d 个", len(a.pages))
	}
	if !a.confirmRebuildAt.IsZero() {
		t.Error("确认状态应复位")
	}
}
