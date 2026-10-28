package view

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
)

// settingsPage 设置页：分区域组织用户偏好。
//   - 外观：主题（跟随系统/浅色/深色）、字体大小
//   - 窗口与关闭：关闭按钮行为（每次询问/最小化到托盘/直接退出）
//   - 关于：应用信息与配置文件路径
//
// 注意：本函数只负责渲染；偏好变更的检测与持久化统一在 App.View 中完成
// （通过对比 themeIdx/closeModeIdx/fontSize 与其“已应用”快照）。
func (a *App) settingsPage(c *ui.Context, t *ui.Theme) {
	ui.Text(c, "设置").FontSize(24).Bold().Padding(0, 0, 16)

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

		// ===== 关于 =====
		section(c, t, "关于", func() {
			if p, err := settings.Path(); err == nil {
				ui.Text(c, "配置文件："+p).
					TextColor(t.TextMuted).FontSize(t.FontSize-2).Padding(0, 0, 6)
			}
			ui.Text(c, "量子科学 · bestuav-mygo").Padding(0, 0, 4)
			ui.Text(c, "基于 mygo 的纯 Go 原生桌面应用。").
				TextColor(t.TextMuted).FontSize(t.FontSize - 2)
		})
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
