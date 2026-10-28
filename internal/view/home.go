package view

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/appmeta"
)

// 首页：产品级概览。四块内容——欢迎与快捷操作、四个 KPI、近期任务、服务与入口。
//
// 数据都是本地演示数据（没有后端），但结构按真实看板组织：KPI 有环比、任务有
// 状态与耗时、服务有健康度。接后端时只需把这些字面量换成接口返回即可。

// kpi 首页指标项。
type kpi struct {
	icon  *ui.SVG
	label string
	value string
	delta string
	up    bool
}

// homeKPIs 四个核心指标。
func (a *App) homeKPIs() []kpi {
	return []kpi{
		{iconBolt, "今日任务", "37", "12.4%（较昨日）", true},
		{iconChecklist, "任务成功率", "99.2%", "0.3%", true},
		{iconShield, "有效授权", "1,286", "24 份", true},
		{iconClock, "平均耗时", "1.8s", "0.4s（较昨日）", false},
	}
}

func (a *App) homePage(c *ui.Context, t *ui.Theme) {
	// ---- 欢迎区 ----
	ui.Row(c).Gap(12).AlignItems(ui.Center).Padding(0, 0, 4).Children(func() {
		ui.Column(c).Gap(4).Grow(1).Children(func() {
			ui.Text(c, greeting()+"，欢迎使用"+appmeta.Title).
				FontSize(20).Bold().SingleLine()
			ui.Text(c, "授权服务与任务队列运行正常，以下是最近 24 小时的运行概览。").
				FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		if ui.PrimaryButton(c, "发起任务").Clicked() {
			a.page = "task"
		}
		if ui.Button(c, "生成 license").Clicked() {
			a.page = "gen-license"
		}
	})

	// ---- KPI 四卡 ----
	ui.Row(c).Gap(16).Padding(16, 0, 4).Children(func() {
		for _, k := range a.homeKPIs() {
			kpiCard(c, t, k.icon, k.label, k.value, k.delta, k.up)
		}
	})

	// ---- 主体两栏 ----
	ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
		// 左：近期任务
		ui.Column(c).Grow(1).Children(func() {
			card(c, t, func() {
				cardHeader(c, t, "近期任务", "点击任意一行查看任务详情（演示）", func() {
					if ui.Button(c, "查看全部").Clicked() {
						a.page = "task"
					}
				})
				a.recentTaskRows(c, t)
			})
		})

		// 右：服务状态 + 快捷入口
		ui.Column(c).Width(320).Shrink(0).Gap(16).Children(func() {
			card(c, t, func() {
				cardHeader(c, t, "服务状态", "实时（演示）", nil)
				ui.Column(c).Gap(12).Children(func() {
					statRow(c, t, "授权服务", "api.bestuav.local · 12ms", true)
					statRow(c, t, "license 签发", "签名服务 · 正常", true)
					statRow(c, t, "对象存储", "report 桶 · 78% 已用", true)
					statRow(c, t, "任务队列", "0 个失败任务", true)
				})
			})
			card(c, t, func() {
				cardHeader(c, t, "快捷入口", "", nil)
				ui.Grid(c).Columns(2).Gap(10).Children(func() {
					a.quickTile(c, t, "异步任务", iconBolt, "task")
					a.quickTile(c, t, "生成 license", iconFileKey, "gen-license")
					a.quickTile(c, t, "基本信息", iconUser, "basic-info")
					a.quickTile(c, t, "设置", iconGear, "settings")
				})
			})
		})
	})
}

// recentTaskRows 首页的近期任务列表（最多 5 行，手工排版以便控制紧凑度）。
func (a *App) recentTaskRows(c *ui.Context, t *ui.Theme) {
	n := len(a.tasks)
	if n > 5 {
		n = 5
	}
	if n == 0 {
		emptyHint(c, t, "暂无任务记录")
		return
	}
	ui.Column(c).Gap(2).Children(func() {
		for i := 0; i < n; i++ {
			r := a.tasks[i]
			row := ui.Row(c).Gap(12).AlignItems(ui.Center).
				Padding(9, 8).Radius(t.Radius)
			if row.Hovered() {
				row.Background(t.SurfaceHover)
			}
			row.Children(func() {
				ui.Column(c).Gap(1).Grow(1).Children(func() {
					ui.Text(c, r.name).SingleLine()
					ui.Text(c, r.kind+" · "+r.cost).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
				})
				pill(c, t, r.status, statusColor(t, r.status))
			})
			if row.Clicked() {
				a.page = "task"
			}
		}
	})
}

// quickTile 快捷入口小方块：图标 + 文字，点击跳转。
func (a *App) quickTile(c *ui.Context, t *ui.Theme, label string, icon *ui.SVG, page string) {
	tile := ui.Column(c).Gap(6).Padding(12).Radius(8).
		Background(withAlpha(t.SurfaceHover, 0.8)).Border(1, t.Border)
	if tile.Hovered() {
		tile.Background(t.SurfacePressed)
	}
	tile.Children(func() {
		ui.Icon(c, icon).FontSize(16).TextColor(t.Accent)
		ui.Text(c, label).FontSize(t.FontSize - 2).SingleLine()
	})
	if tile.Clicked() {
		a.page = page
	}
}

// homeSummary 供其它页面复用的简短摘要文案（如任务页顶部）。
func (a *App) homeSummary() string {
	return fmt.Sprintf("累计任务 %d 条，失败 0 条", len(a.tasks))
}
