package view

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/appmeta"
	"bestuav-mygo/internal/settings"
	"bestuav-mygo/internal/ui/icons"
	"bestuav-mygo/internal/ui/kit"
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
	ui.Column(c).Gap(kit.PageGap).Children(func() {
		// ===== 外观 =====
		kit.Section(c, t, "外观", func() {
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
		kit.Section(c, t, "窗口与关闭", func() {
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
		kit.Section(c, t, "启动", func() {
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

			// 页面缓存：闲置多久销毁（对应 javafx 版的 idleTime）
			settingRow(c, t, "页面缓存", func() {
				ui.Stepper(c, &a.pageIdle, 0, settings.MaxPageIdleMinutes, 5)
				hint := fmt.Sprintf("闲置 %d 分钟后销毁缓存页面", int(math.Round(a.pageIdle)))
				if int(math.Round(a.pageIdle)) == 0 {
					hint = "不按闲置销毁（常驻页面始终保留）"
				}
				ui.Text(c, hint).TextColor(t.TextMuted).FontSize(t.FontSize - 2)
			})
			ui.Text(c, "「不缓存」的页面（如生成 license）离开即销毁，不受此项影响。").
				TextColor(t.TextMuted).FontSize(t.FontSize - 2)
		})

		// ===== 关于 =====
		// 产品标识与版本留在页面上（一眼可见）；应用标识 / 运行环境 / 页面缓存 /
		// 数据库与日志路径这些「查问题时才看」的信息收进弹窗，点「查看」打开
		// （内容见 basicinfo.go）。
		kit.Section(c, t, "关于", func() {
			ui.Text(c, appmeta.Title+" · "+appmeta.Name).Bold()
			ui.Text(c, "版本 "+appmeta.BuildVersion()).
				TextColor(t.TextMuted).FontSize(t.FontSize - 2)
			ui.Text(c, "基于 mygo 的纯 Go 原生桌面应用。").
				TextColor(t.TextMuted).FontSize(t.FontSize - 2)

			ui.Row(c).Gap(kit.CardGap).AlignItems(ui.Center).Margin(6, 0, 0, 0).Children(func() {
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, "基本信息").Bold()
					ui.Text(c, "应用标识、运行环境、页面缓存、数据库与日志路径").
						TextColor(t.TextMuted).FontSize(t.FontSize - 2)
				})
				if ui.Button(c, "查看").Clicked() {
					a.showBasicInfo = true
				}
			})
		})

		// ===== 更新 =====
		// 进入设置页（onEnter）即后台检查；这里只负责展示状态与让用户决定要不要更新。
		a.updateSection(c, t)
	})
}

// updateSection 「更新」区块：只展示检查/安装状态，动作全部上提到设置页标题栏
// 的更新按钮（无更新=图标按钮，有更新=高亮「更新」胶囊），点击弹窗确认。
//
// 设计依据：Dropbox 等桌面应用把「更新」做成标题栏右侧的图标/胶囊按钮，正文区
// 只保留状态文案与安装进度，避免页面内再出现一套重复的更新按钮。
func (a *App) updateSection(c *ui.Context, t *ui.Theme) {
	kit.Section(c, t, "更新", func() {
		// 状态优先级：安装中 > 检查中 > 已禁用 > 检查失败 > 可更新 > 已最新 > 尚未检查。
		switch {
		case a.updateInstalling:
			ui.Column(c).Gap(8).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					kit.BreathingDot(c, t.Accent, 7, kit.BreathePeriod)
					ui.Text(c, "正在下载并更新…").TextColor(t.TextMuted).Grow(1)
					if a.updateProgress > 0 {
						ui.Text(c, fmt.Sprintf("%.0f%%", a.updateProgress*100)).
							TextColor(t.TextMuted)
					}
				})
				ui.Progress(c, a.updateProgress).Grow(1)
			})
		case a.updateChecking:
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				kit.BreathingDot(c, t.Accent, 7, kit.BreathePeriod)
				ui.Text(c, "正在检查更新…").TextColor(t.TextMuted)
			})
		case a.updateDisabled:
			ui.Text(c, "当前构建未启用自动更新（开发构建 / 未通过 mygo build 产出）。经 go tool mygo build 生成的正式发布包支持自动下载更新。").
				TextColor(t.TextMuted).FontSize(t.FontSize - 2)
		case a.updateErr != nil:
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Bell).TextColor(t.Danger)
				ui.Text(c, "检查失败："+a.updateErr.Error()).
					TextColor(t.Danger).FontSize(t.FontSize - 2).Grow(1).SingleLine()
			})
		case a.availableUpdate != nil && !a.updateDismissed:
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Download).TextColor(t.Accent)
				ui.Text(c, "发现新版本 "+a.availableUpdate.Version+"：点击右上角「更新」查看详情").
					TextColor(t.TextMuted).FontSize(t.FontSize - 2).Grow(1)
			})
		case a.availableUpdate != nil:
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Sparkles).TextColor(t.Success)
				ui.Text(c, "已是最新版本（"+a.availableUpdate.Version+"）").
					TextColor(t.TextMuted).FontSize(t.FontSize - 2)
			})
		default:
			ui.Text(c, "尚未检查。进入本页即自动检查，也可点标题栏更新图标手动检查。").
				TextColor(t.TextMuted).FontSize(t.FontSize - 2)
		}
	})
}

// updateDialog 点击标题栏「更新」胶囊后弹出的确认弹窗：展示新版本号与发布说明，
// 由用户决定是否立即下载安装（mygo.Updater 负责签名校验与就地替换，完成后重启）。
//
// 绝不做「自动更新」：发现新版本只提示，是否更新完全由用户点「立即更新」决定；
// 「忽略」隐藏本次会话提示（下次进入设置重新出现）。
func (a *App) updateDialog(c *ui.Context, t *ui.Theme) {
	modalPanel(c, t, &a.showUpdateDialog, func() {
		up := a.availableUpdate
		if up == nil {
			return
		}
		ui.Text(c, "发现新版本").FontSize(18).Bold()
		ui.Text(c, "有新版本可用，是否立即更新？").
			TextColor(t.TextMuted).Padding(0, 6, 10)

		// 版本 + 发布说明卡
		ui.Column(c).Gap(8).Padding(12).Radius(kit.CardRadius).
			Background(kit.Alpha(t.Accent, 0.10)).Border(1, t.Accent).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Download).TextColor(t.Accent)
				ui.Text(c, up.Version).Bold().Grow(1)
			})
			if notes := strings.TrimSpace(up.Notes); notes != "" {
				ui.Text(c, notes).TextColor(t.TextMuted).FontSize(t.FontSize - 2)
			}
			if !up.Date.IsZero() {
				// manifest 里的 date 是构建时的 **UTC** 时间（mygo 写的是
				// time.Now().UTC().Format(time.RFC3339)），直接 Format 会按 UTC 显示——
				// 北京时间凌晨构建时就会显示成"前一天"，看着像日期不更新。
				// 这里转成本地时区，并带上时分，避免"怎么一直是昨天"的困惑。
				ui.Text(c, "发布时间："+up.Date.Local().Format("2006-01-02 15:04")).
					TextColor(t.TextMuted).FontSize(t.FontSize - 2)
			}
		})

		ui.Text(c, "当前版本 "+appmeta.BuildVersion()).
			TextColor(t.TextMuted).FontSize(t.FontSize-2).Padding(4, 0, 0, 0)

		ui.Row(c).Gap(8).Justify(ui.End).Padding(0, 12, 0).Children(func() {
			if ui.Button(c, "忽略").Clicked() {
				a.updateDismissed = true
				a.showUpdateDialog = false
				a.requestFrame()
			}
			install := ui.PrimaryButton(c, "立即更新")
			if a.updateInstalling {
				install.Disabled(true)
			}
			if install.Clicked() {
				a.showUpdateDialog = false
				a.installUpdate()
			}
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
	kit.Section(c, t, "背景", func() {
		// 背景图片：展示当前路径 + 选择 / 恢复默认
		ui.Row(c).Gap(kit.CardGap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "背景图片").Width(kit.SettingsLabelWidth).Shrink(0).SingleLine()

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

// settingRow 一行「标签 + 控件」的布局（标签宽度取全局规格，保证多行对齐）。
func settingRow(c *ui.Context, t *ui.Theme, label string, fn func()) {
	ui.Row(c).Gap(kit.RowGap).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).Width(kit.SettingsLabelWidth).Shrink(0).SingleLine()
		fn()
	})
}

// closeChoiceDialog 首次关闭（CloseMode=="ask"）时弹出的询问弹窗：
// 让用户选择本次关闭行为，并可勾选“记住我的选择”以便后续按该行为执行。
func (a *App) closeChoiceDialog(c *ui.Context, t *ui.Theme) {
	modalPanel(c, t, &a.showCloseChoice, func() {
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
