package view

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/appmeta"
	"bestuav-mygo/internal/ui/icons"
	"bestuav-mygo/internal/ui/kit"
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
		{icons.Bolt, "今日任务", "37", "12.4%（较昨日）", true},
		{icons.Checklist, "任务成功率", "99.2%", "0.3%", true},
		{icons.Shield, "有效授权", "1,286", "24 份", true},
		{icons.Clock, "平均耗时", "1.8s", "0.4s（较昨日）", false},
	}
}

func (a *App) homePage(c *ui.Context, t *ui.Theme) {
	// 说明：页面只写「区块」，区块之间的垂直间距与四周留白由 shell.Content
	// 统一施加（kit.PageGap / PagePadX / PagePadY），这里不要再写外边距。

	// ---- 欢迎区 ----
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
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
	// 用栅格而不是 Row+Grow：四张卡严格等宽（内容长度不影响列宽），同一行等高。
	kit.RowGrid(c, 4, func() {
		for _, k := range a.homeKPIs() {
			kit.KPICard(c, t, k.icon, k.label, k.value, k.delta, k.up)
		}
	})

	// ---- 主体两栏（左主右辅，宽度按 2:1 分配，间距与上面的区块一致）----
	ui.Row(c).Gap(kit.RowGap).AlignItems(ui.Start).Children(func() {
		// 左：近期任务
		ui.Column(c).Grow(2).Children(func() {
			kit.Card(c, t, func() {
				kit.CardHeader(c, t, "近期任务", "点击任意一行查看任务详情（演示）", func() {
					if ui.Button(c, "查看全部").Clicked() {
						a.page = "task"
					}
				})
				a.recentTaskRows(c, t)
			})
		})

		// 右：服务状态 + 快捷入口
		ui.Column(c).Grow(1).Gap(kit.PageGap).Children(func() {
			kit.Card(c, t, func() {
				kit.CardHeader(c, t, "服务状态", "实时（演示）", nil)
				ui.Column(c).Gap(12).Children(func() {
					kit.StatRow(c, t, "授权服务", "api.bestuav.local · 12ms", true)
					kit.StatRow(c, t, "license 签发", "签名服务 · 正常", true)
					kit.StatRow(c, t, "对象存储", "report 桶 · 78% 已用", true)
					kit.StatRow(c, t, "任务队列", "0 个失败任务", true)
				})
			})
			kit.Card(c, t, func() {
				kit.CardHeader(c, t, "快捷入口", "", nil)
				ui.Grid(c).Columns(2).Gap(10).Children(func() {
					a.quickTile(c, t, "异步任务", icons.Bolt, "task")
					a.quickTile(c, t, "生成 license", icons.FileKey, "gen-license")
					a.quickTile(c, t, "基本信息", icons.User, "basic-info")
					a.quickTile(c, t, "设置", icons.Gear, "settings")
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
		kit.EmptyHint(c, t, "暂无任务记录")
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
				kit.Pill(c, t, r.status, kit.StatusColor(t, r.status))
			})
			if row.Clicked() {
				a.page = "task"
			}
		}
	})
}

// quickTile 快捷入口小方块：图标 + 文字，点击跳转。
func (a *App) quickTile(c *ui.Context, t *ui.Theme, label string, icon *ui.SVG, page string) {
	tile := ui.Column(c).Gap(6).Padding(12).Radius(kit.TileRadius).
		Background(kit.Alpha(t.SurfaceHover, 0.8)).Border(1, t.Border)
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

// homeSummary 供其它页面复用的简短摘要文案（如任务页顶部）。
func (a *App) homeSummary() string {
	return fmt.Sprintf("累计任务 %d 条，失败 0 条", len(a.tasks))
}
