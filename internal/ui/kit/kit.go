package kit

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// 本文件是页面里反复用到的通用构件：卡片、区块、KPI 卡、状态药丸、键值行。
// 集中在一处是为了「所有页面长得像一个产品」，改一次全站统一。
//
// 这里的所有构件都只依赖 (c, t)，不引用任何业务状态——业务页面把它们当积木拼。

// Card 标准卡片容器（圆角 + 边框 + 内边距 + 表面色 + 内部统一行距）。
func Card(c *ui.Context, t *ui.Theme, fn func()) ui.Element {
	return ui.Column(c).Gap(CardGap).Padding(CardPad).Radius(CardRadius).
		Background(t.Surface).Border(1, t.Border).Children(fn)
}

// Section 带标题的设置区块卡片（标题字号略大于卡片内的正文标题）。
func Section(c *ui.Context, t *ui.Theme, title string, fn func()) ui.Element {
	return ui.Column(c).Gap(CardGap).Padding(CardPad).Radius(CardRadius).
		Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Text(c, title).FontSize(16).Bold()
		fn()
	})
}

// CardHeader 卡片标题行：标题 + 可选说明 + 右侧动作区。
func CardHeader(c *ui.Context, t *ui.Theme, title, desc string, action func()) {
	ui.Row(c).Gap(CardGap).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, title).FontSize(15).Bold().SingleLine()
			if desc != "" {
				ui.Text(c, desc).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			}
		})
		if action != nil {
			action()
		}
	})
}

// RowGrid 等宽分栏栅格：n 列严格等宽、同一行的卡片自动等高拉伸。
//
// 页面里凡是要「一行放几块」的地方都用它，而不是 Row + Grow：
// flex 的 Grow 只分配剩余空间，内容长度不同就会长出不同宽度（视觉上参差不齐），
// 栅格列宽则是严格相等的。
func RowGrid(c *ui.Context, cols int, fn func()) ui.Element {
	return ui.Grid(c).Columns(cols).Gap(RowGap).Children(fn)
}

// Pill 小号状态药丸：淡色底 + 同色文字，用于「已完成 / 运行中 / 失败」等状态。
func Pill(c *ui.Context, t *ui.Theme, text string, col ui.Color) ui.Element {
	return ui.Row(c).Shrink(0).AlignItems(ui.Center).
		Padding(2, 8).Radius(999).
		Background(Alpha(col, 0.14)).
		Children(func() {
			ui.Text(c, text).FontSize(t.FontSize - 2).TextColor(col).SingleLine()
		})
}

// StatusColor 把任务状态映射成颜色（成功绿、进行中强调色、排队灰、失败红）。
func StatusColor(t *ui.Theme, status string) ui.Color {
	switch status {
	case "已完成":
		return t.Success
	case "运行中":
		return t.Accent
	case "失败":
		return t.Danger
	default:
		return t.TextMuted
	}
}

// KV 一行「标签 → 值」，标签固定宽度保证多行对齐。
func KV(c *ui.Context, t *ui.Theme, key, value string) {
	ui.Row(c).Gap(CardGap).AlignItems(ui.Center).Children(func() {
		ui.Text(c, key).Width(LabelWidth).Shrink(0).
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		ui.Text(c, value).Grow(1).Selectable().SingleLine()
	})
}

// KPICard 指标卡：图标 + 名称 + 大数字 + 环比变化。
//
// 不设 Grow：宽度由所处的 RowGrid 列宽决定，同一行的指标卡因此严格等宽。
func KPICard(c *ui.Context, t *ui.Theme, icon *ui.SVG, label, value, delta string, up bool) {
	kpiCard(c, t, icon, label, delta, up, func() {
		ui.Text(c, value).FontSize(24).Bold().SingleLine()
	})
}

// CountingKPICard 数字滚动版指标卡：版式与 KPICard 完全一致，只是大数字进场时
// 从 0 滚到 target（见 CountUp 的说明）。start 是滚动开始的帧时刻。
func CountingKPICard(c *ui.Context, t *ui.Theme, icon *ui.SVG, label string,
	target float64, format func(float64) string, start time.Time, delta string, up bool) {
	kpiCard(c, t, icon, label, delta, up, func() {
		CountUp(c, target, start, CountUpDur, format)
	})
}

// kpiCard 两种指标卡共用的版式；number 负责画中间那个大数字。
func kpiCard(c *ui.Context, t *ui.Theme, icon *ui.SVG, label, delta string, up bool, number func()) {
	ui.Column(c).Gap(10).Padding(16).Radius(CardRadius).
		Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon).FontSize(15).TextColor(t.Accent)
			ui.Text(c, label).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1).SingleLine()
		})
		number()
		if delta != "" {
			col := t.Success
			arrow := "↑ "
			if !up {
				col, arrow = t.Danger, "↓ "
			}
			ui.Text(c, arrow+delta).FontSize(t.FontSize - 2).TextColor(col).SingleLine()
		}
	})
}

// StatRow 服务状态行：状态点 + 名称 + 右侧说明。
func StatRow(c *ui.Context, t *ui.Theme, name, desc string, ok bool) {
	col := t.Success
	if !ok {
		col = t.Danger
	}
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Width(8).Height(8).Radius(999).Shrink(0).Background(col)
		ui.Column(c).Gap(1).Grow(1).Children(func() {
			ui.Text(c, name).SingleLine()
			ui.Text(c, desc).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
		})
		Pill(c, t, map[bool]string{true: "正常", false: "异常"}[ok], col)
	})
}

// EmptyHint 空列表/未选择时的提示。
func EmptyHint(c *ui.Context, t *ui.Theme, text string) {
	ui.Text(c, text).TextColor(t.TextMuted).FontSize(t.FontSize-2).Padding(8, 0)
}

// Alpha 用「绝对」alpha（0..1）替换颜色的透明度，而不是与已有 alpha 相乘。
//
// 这一点很关键：mygo 的 Theme 是每帧共享的同一份实例，若用乘算累加，界面会
// 随帧数越来越透明。绝对赋值保证幂等——每帧结果相同，关掉背景图后也能立刻
// 恢复完全不透明。
func Alpha(c ui.Color, a float64) ui.Color {
	return ui.RGBA(c.R, c.G, c.B, float32(Clamp01(a)))
}

// Clamp01 把数值夹到 [0,1]。
func Clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
