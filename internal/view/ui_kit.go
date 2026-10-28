package view

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// 本文件是页面里反复用到的小构件：卡片、KPI 卡、状态药丸、键值行、区块标题。
// 集中在一处是为了「所有页面长得像一个产品」，改一次全站统一。

// card 一个标准卡片容器（圆角 + 边框 + 内边距 + 表面色）。
func card(c *ui.Context, t *ui.Theme, fn func()) ui.Element {
	return ui.Column(c).Gap(12).Padding(18).Radius(10).
		Background(t.Surface).Border(1, t.Border).Children(fn)
}

// cardHeader 卡片标题行：标题 + 可选说明 + 右侧动作区。
func cardHeader(c *ui.Context, t *ui.Theme, title, desc string, action func()) {
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
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

// pill 小号状态药丸：淡色底 + 同色文字，用于「已完成 / 运行中 / 失败」等状态。
func pill(c *ui.Context, t *ui.Theme, text string, col ui.Color) ui.Element {
	return ui.Row(c).Shrink(0).AlignItems(ui.Center).
		Padding(2, 8).Radius(999).
		Background(withAlpha(col, 0.14)).
		Children(func() {
			ui.Text(c, text).FontSize(t.FontSize - 2).TextColor(col).SingleLine()
		})
}

// statusColor 把任务状态映射成颜色（成功绿、进行中强调色、排队灰、失败红）。
func statusColor(t *ui.Theme, status string) ui.Color {
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

// kv 一行「标签 → 值」，标签固定宽度保证多行对齐。
func kv(c *ui.Context, t *ui.Theme, key, value string) {
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, key).Width(96).Shrink(0).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		ui.Text(c, value).Grow(1).Selectable().SingleLine()
	})
}

// kpiCard 指标卡：图标 + 名称 + 大数字 + 环比变化。
func kpiCard(c *ui.Context, t *ui.Theme, icon *ui.SVG, label, value, delta string, up bool) {
	ui.Column(c).Gap(10).Padding(16).Radius(10).Grow(1).
		Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon).FontSize(15).TextColor(t.Accent)
			ui.Text(c, label).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1).SingleLine()
		})
		ui.Text(c, value).FontSize(24).Bold().SingleLine()
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

// statRow 服务状态行：状态点 + 名称 + 右侧说明。
func statRow(c *ui.Context, t *ui.Theme, name, desc string, ok bool) {
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
		pill(c, t, map[bool]string{true: "正常", false: "异常"}[ok], col)
	})
}

// greeting 按当前时间给出问候语。
func greeting() string {
	h := time.Now().Hour()
	switch {
	case h < 6:
		return "凌晨好"
	case h < 12:
		return "上午好"
	case h < 14:
		return "中午好"
	case h < 18:
		return "下午好"
	default:
		return "晚上好"
	}
}

// emptyHint 空列表/未选择时的提示。
func emptyHint(c *ui.Context, t *ui.Theme, text string) {
	ui.Text(c, text).TextColor(t.TextMuted).FontSize(t.FontSize-2).Padding(8, 0)
}
