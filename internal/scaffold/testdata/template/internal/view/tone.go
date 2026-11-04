package view

import (
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
)

// 背景色调：只替换背景 / 控件面 / 描边这些「大面积」颜色，不改强调色（配色）与
// 正文颜色。与 Palette 正交：可以「护眼绿 + 青碧」这样组合。
//
// 实现约束：必须在 applyBackdropTheme 之前调用——背景图存在时，backdrop 会给
// Background/Surface* 统一降透明度，先定基色再降透明度才是对的。

// toneDef 一套色调：浅色与深色各一组「背景 / 控件面 / 悬停 / 按下 / 描边」。
type toneDef struct {
	name  string // 存储值（与 internal/settings 的 Tone* 常量一致）
	label string
	light toneColors
	dark  toneColors
}

type toneColors struct {
	bg, surface, hover, pressed, border ui.Color
}

// toneDefs 全部色调，顺序即设置页顺序；第一项是「主题默认」。
var toneDefs = []toneDef{
	{name: settings.ToneDefault, label: "默认"},
	{
		name: settings.ToneWarm, label: "暖白",
		light: toneColors{
			bg: ui.Hex("#fbf6ee"), surface: ui.Hex("#f5ecdd"),
			hover: ui.Hex("#eee2cf"), pressed: ui.Hex("#e5d6bf"), border: ui.Hex("#ddcfb8"),
		},
		dark: toneColors{
			bg: ui.Hex("#1b1712"), surface: ui.Hex("#26201a"),
			hover: ui.Hex("#322a21"), pressed: ui.Hex("#3d3329"), border: ui.Hex("#4a3f32"),
		},
	},
	{
		name: settings.ToneGreen, label: "护眼绿",
		light: toneColors{
			bg: ui.Hex("#f1f7ef"), surface: ui.Hex("#e6f0e2"),
			hover: ui.Hex("#dbe9d6"), pressed: ui.Hex("#cfe1c8"), border: ui.Hex("#c8dbc1"),
		},
		dark: toneColors{
			bg: ui.Hex("#101710"), surface: ui.Hex("#18231a"),
			hover: ui.Hex("#213024"), pressed: ui.Hex("#2a3d2d"), border: ui.Hex("#35503a"),
		},
	},
	{
		name: settings.ToneMidnight, label: "午夜蓝",
		light: toneColors{
			bg: ui.Hex("#eef2f9"), surface: ui.Hex("#e2e9f5"),
			hover: ui.Hex("#d6e0f0"), pressed: ui.Hex("#c9d6ea"), border: ui.Hex("#c3d1e6"),
		},
		dark: toneColors{
			bg: ui.Hex("#0b1220"), surface: ui.Hex("#141d2e"),
			hover: ui.Hex("#1d293d"), pressed: ui.Hex("#26344b"), border: ui.Hex("#31415c"),
		},
	},
}

// toneName 把索引映射成存储值；越界回落到默认色调。
func toneName(i int) string {
	if i < 0 || i >= len(toneDefs) {
		return settings.ToneDefault
	}
	return toneDefs[i].name
}

// toneIndex 把存储值映射成索引；未知值回落到默认色调。
func toneIndex(name string) int {
	for i, d := range toneDefs {
		if d.name == name {
			return i
		}
	}
	return 0
}

// toneLabels 设置页分段控件的文案。
func toneLabels() []string {
	out := make([]string, len(toneDefs))
	for i, d := range toneDefs {
		out[i] = d.label
	}
	return out
}

// applyTone 按当前色调覆盖背景与控件面；「默认」不覆盖。
func applyTone(t *ui.Theme, name string) {
	d := toneDefs[toneIndex(name)]
	if d.name == settings.ToneDefault {
		return
	}
	c := d.light
	if t.Dark {
		c = d.dark
	}
	t.Background = c.bg
	t.Surface = c.surface
	t.SurfaceHover = c.hover
	t.SurfacePressed = c.pressed
	t.Border = c.border
}
