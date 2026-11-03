package view

import (
	"testing"

	"bestuav-mygo/internal/settings"
)

// TestEffortSliderIsNotBuiltAtStartup 投入滑块的离屏栅格化不能放在启动路径上。
//
// effort.New() 要调 ui.Render 跑一整帧来栅格化字形，实测占 newApp 的 61%
// （约 1.4ms、1MB），而这个演示页大多数用户根本不会打开。改成「第一次进入该页时
// 在帧外构建」之后，启动不再为它付钱。
func TestEffortSliderIsNotBuiltAtStartup(t *testing.T) {
	// 刻意不用 newTestApp：那个辅助函数会显式预建（为了让页面断言拿到就绪态的界面）。
	a := newApp(settings.Default())
	if _, built := a.pageStat(effortPageID).data[effortSliderKey]; built {
		t.Error("启动时不应构建投入滑块")
	}

	// 第一次进入：本帧只画占位，并把构建排到帧外。
	a.page = effortPageID
	tt := render(a)
	mustHave(t, tt, "正在准备画布…")

	// 构建完成后（真实运行由主循环任务触发，单测里手动调）应当直接可用。
	a.buildEffortSlider()
	tt.Frame()
	mustHave(t, tt, "当前档位 Low", "拉满（Galaxy）")
}
