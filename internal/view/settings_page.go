package view

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/appmeta"
	"bestuav-mygo/internal/settings"
)

// settingsPage 设置页：分区域组织用户偏好。
//   - 外观：主题（跟随系统/浅色/深色）、字体大小
//   - 背景：背景图片（选择/恢复默认）、菜单区与内容区的不透明度
//   - 窗口与关闭：关闭按钮行为（每次询问/最小化到托盘/直接退出）
//   - 启动：是否允许多开（单实例 / 多实例）
//   - 关于：应用信息与配置文件路径
//
// 注意：本函数只负责渲染；偏好变更的检测与持久化统一在 App.View 中完成
// （通过对比各绑定值与其“已应用”快照）。
func (a *App) settingsPage(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Gap(16).Children(func() {
		// ===== 外观 =====
		section(c, t, "外观", func() {
			settingRow(c, t, "主题", func() {
				ui.Segmented(c, &a.themeIdx, "跟随系统", "浅色", "深色")
			})
			settingRow(c, t, "字体大小", func() {
				ui.Slider(c, &a.fontSize, 11, 20).Width(220)
				ui.Text(c, fmt.Sprintf("%.0f px", math.Round(a.fontSize))).
					TextColor(t.TextMuted).Padding(8, 0, 0, 0)
			})
		})

		// ===== 背景 =====
		a.backgroundSection(c, t)

		// ===== 窗口与关闭 =====
		section(c, t, "窗口与关闭", func() {
			ui.Text(c, "关闭按钮行为").Bold().Padding(0, 0, 8)
			ui.RadioGroup(c, func() {
				ui.Radio(c, &a.closeModeIdx, 0, "每次询问（首次关闭弹出选择）")
				ui.Radio(c, &a.closeModeIdx, 1, "最小化到托盘")
				ui.Radio(c, &a.closeModeIdx, 2, "直接退出")
			})
			ui.Text(c, "选择后，点击窗口关闭按钮即按此行为执行；也可在此随时修改。").
				TextColor(t.TextMuted).FontSize(t.FontSize-2).Padding(8, 0, 0, 0)
		})

		// ===== 启动 =====
		section(c, t, "启动", func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Switch(c, &a.allowMultiInstance).Shrink(0)
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, "允许多开").Bold()
					ui.Text(c, "关闭时为单实例：重复启动只会唤出已在运行的窗口并退出新进程；"+
						"开启后每次启动都是独立窗口。").
						TextColor(t.TextMuted).FontSize(t.FontSize - 2)
				})
			})
			ui.Text(c, "提示：该设置需重启应用后生效（单实例锁在进程启动时获取）。").
				TextColor(t.TextMuted).FontSize(t.FontSize-2).Padding(6, 0, 0, 0)
		})

		// ===== 关于 =====
		section(c, t, "关于", func() {
			if p, err := settings.Path(); err == nil {
				ui.Text(c, "配置文件："+p).
					TextColor(t.TextMuted).FontSize(t.FontSize-2).Padding(0, 0, 6)
			}
			ui.Text(c, appmeta.Title+" · "+appmeta.Name).Padding(0, 0, 4)
			ui.Text(c, "应用标识："+appmeta.Identifier).
				TextColor(t.TextMuted).FontSize(t.FontSize-2).Padding(0, 0, 4)
			ui.Text(c, "基于 mygo 的纯 Go 原生桌面应用。").
				TextColor(t.TextMuted).FontSize(t.FontSize - 2)
		})
	})
}

// backgroundSection 「背景」区块：背景图片选择、菜单区/内容区不透明度、
// 恢复默认。
//
// 设计取舍：
//   - 透明度拆成「菜单区」「内容区」两个独立滑块，用户可以只让侧栏更透、内容区
//     更实（阅读区域通常需要更实）；
//   - 滑块调节的是「不透明度」而非「透明度」：100% = 完全遮蔽背景图（观感与未设
//     背景图时一致），数值越小背景图越透出；
//   - 未设置背景图时两个滑块置灰（无意义），并提供「恢复默认」一键清除背景图。
func (a *App) backgroundSection(c *ui.Context, t *ui.Theme) {
	hasBg := a.settings.BackgroundImage != ""
	section(c, t, "背景", func() {
		// 背景图片：展示当前路径 + 选择 / 恢复默认
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "背景图片").Width(90).Shrink(0).SingleLine()

			label := "未设置（使用主题默认外观）"
			if hasBg {
				label = a.settings.BackgroundImage
			}
			ui.Text(c, label).TextColor(t.TextMuted).Grow(1).SingleLine()

			if ui.Button(c, "选择图片…").Shrink(0).Clicked() {
				a.chooseBackground()
			}
			reset := ui.Button(c, "恢复默认").Shrink(0)
			if !hasBg {
				// 未设置背景图时「恢复默认」无意义：禁用，避免误操作。
				reset.Disabled(true)
			} else if reset.Clicked() {
				a.resetBackground()
			}
		})

		if a.bgError != "" {
			ui.Text(c, a.bgError).TextColor(t.Danger).FontSize(t.FontSize - 2)
		}

		// 菜单区 / 内容区不透明度
		settingRow(c, t, "菜单区", func() {
			sl := ui.Slider(c, &a.menuOpacity, settings.MinOpacity, settings.MaxOpacity).Width(220)
			if !hasBg {
				sl.Disabled(true)
			}
			ui.Text(c, fmt.Sprintf("%.0f%%", a.menuOpacity*100)).
				TextColor(t.TextMuted).Padding(8, 0, 0, 0)
		})
		settingRow(c, t, "内容区", func() {
			sl := ui.Slider(c, &a.contentOpacity, settings.MinOpacity, settings.MaxOpacity).Width(220)
			if !hasBg {
				sl.Disabled(true)
			}
			ui.Text(c, fmt.Sprintf("%.0f%%", a.contentOpacity*100)).
				TextColor(t.TextMuted).Padding(8, 0, 0, 0)
		})

		hint := "「不透明度」越大遮罩越实、背景图越淡；100% 即完全遮蔽背景图。"
		if !hasBg {
			hint = "先设置背景图片，菜单区 / 内容区的不透明度才会生效。"
		}
		ui.Text(c, hint).TextColor(t.TextMuted).FontSize(t.FontSize-2).Padding(4, 0, 0, 0)
	})
}

// section 一个带标题的设置区块卡片。
func section(c *ui.Context, t *ui.Theme, title string, fn func()) {
	ui.Column(c).Gap(10).Padding(18).
		Background(t.Surface).Border(1, t.Border).Radius(10).Children(func() {
		ui.Text(c, title).FontSize(16).Bold()
		fn()
	})
}

// settingRow 一行「标签 + 控件」的布局。
func settingRow(c *ui.Context, t *ui.Theme, label string, fn func()) {
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).Width(90).Shrink(0).SingleLine()
		fn()
	})
}

// closeChoiceDialog 首次关闭（CloseMode=="ask"）时弹出的询问弹窗：
// 让用户选择本次关闭行为，并可勾选“记住我的选择”以便后续按该行为执行。
func (a *App) closeChoiceDialog(c *ui.Context, t *ui.Theme) {
	ui.Modal(c, &a.showCloseChoice, func() {
		ui.Text(c, "关闭窗口").FontSize(18).Bold()
		ui.Text(c, "您希望点击关闭按钮时执行什么操作？").
			TextColor(t.TextMuted).Padding(0, 10, 10)

		ui.RadioGroup(c, func() {
			ui.Radio(c, &a.closeChoice, "minimize", "最小化到托盘")
			ui.Radio(c, &a.closeChoice, "quit", "直接退出")
		})

		ui.Checkbox(c, &a.rememberClose, "记住我的选择")

		ui.Row(c).Gap(8).Justify(ui.End).Padding(0, 12, 0).Children(func() {
			if ui.Button(c, "取消").Clicked() {
				a.showCloseChoice = false
			}
			if ui.PrimaryButton(c, "确定").Clicked() {
				if a.rememberClose {
					a.settings.CloseMode = a.closeChoice
					a.persist()
				}
				a.showCloseChoice = false
				a.runClose(a.closeChoice)
			}
		})
	})
}
