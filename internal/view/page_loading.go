package view

import (
	"github.com/egoist/mygo/plugins/loading"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
)

// ---- 官方示例：加载指示器 ----
//
// 移植自 mygo examples/loading（loading.dev 的 29 种指示器）。动画由插件自己
// 通过 ui.Painter.AnimationFrame 推进，页面不需要额外请求重绘。
//
// 与示例的差别：示例占满整个窗口；这里作为菜单里的一个页面，套进统一的内容区，
// 选项行用卡片承载、指示器网格自适应换行。

var (
	loadingSizes   = [...]float32{16, 24, 32, 48}
	loadingEasings = [...]string{"线性", "缓入缓出", "叠层"}
	loadingCaps    = [...]string{"圆头", "平头"}
)

// loadingDemoPage 加载指示器一览。
func (a *App) loadingDemoPage(c *ui.Context, t *ui.Theme) {
	o := loading.Options{
		Size:   loadingSizes[a.loadingSize],
		Color:  [...]ui.Color{t.Text, t.Accent, t.TextMuted}[a.loadingColor],
		Easing: loading.Easing(a.loadingEasing),
		Cap:    loading.Cap(a.loadingCap),
		Paused: a.loadingPaused,
	}
	spinners := loading.All()

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "加载指示器",
			"官方 examples/loading 移植：29 种指示器，尺寸 / 颜色 / 缓动 / 端点可调", func() {
				ui.Checkbox(c, &a.loadingPaused, "暂停动画")
			})
		ui.Row(c).Wrap().Gap(kit.CardGap).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.loadingSize, "16", "24", "32", "48").Label("尺寸")
			ui.Segmented(c, &a.loadingColor, "正文", "强调", "次要").Label("颜色")
			ui.Segmented(c, &a.loadingEasing, loadingEasings[:]...).Label("缓动")
			ui.Segmented(c, &a.loadingCap, loadingCaps[:]...).Label("端点")
		})
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "全部指示器", itoa(len(spinners))+" 种，地址处为名称", nil)
		ui.Row(c).Wrap().Gap(kit.CardGap).Children(func() {
			for _, s := range spinners {
				ui.Column(c).Width(132).Height(116).Center().Gap(14).Radius(t.Radius).
					Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Column(c).Height(48).Center().Children(func() {
						s.New(c, o).Label(s.Name)
					})
					ui.Text(c, s.Name).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
				})
			}
		})
	})
}
