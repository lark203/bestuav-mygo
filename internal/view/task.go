package view

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// ---- 异步任务页 ----
//
// 两类任务的区别就是用户要求的那种区别：
//   - 轻量级：进度条画在页头下方那条浅线的槽位里，界面照常可用；
//   - 重量级：整个窗口盖遮罩，任务完成前禁止任何操作。
// 两者完成后都会撒彩带（见 feedback.go 的 celebrate）。

// taskRow 任务列表的一行。
type taskRow struct {
	name   string
	kind   string // 轻量级 / 重量级
	status string // 已完成 / 运行中 / 失败 / 排队
	cost   string
}

// demoTaskRows 任务队列的初始演示数据（贴近真实运维场景）。
func demoTaskRows() []taskRow {
	return []taskRow{
		{"同步授权数据", "轻量级", "已完成", "0.8s"},
		{"刷新证书链缓存", "轻量级", "已完成", "0.3s"},
		{"导出月度报表", "轻量级", "已完成", "2.1s"},
		{"重建检索索引", "重量级", "已完成", "18.4s"},
		{"清理过期凭证", "重量级", "排队", "—"},
	}
}

// pushTaskRow 在队列表头部插入一条记录（最多保留 20 条，避免无限增长）。
func (a *App) pushTaskRow(name, kind, status string) {
	a.tasks = append([]taskRow{{name, kind, status, "刚刚"}}, a.tasks...)
	if len(a.tasks) > 20 {
		a.tasks = a.tasks[:20]
	}
}

// taskPage 异步任务页：两张操作卡 + 任务队列表。
func (a *App) taskPage(c *ui.Context, t *ui.Theme) {
	// 两张卡用 Grid(2) 而非两个 Grow(1) 的 Column：Grid 的列宽严格相等，
	// 不会因为一侧文字更长而变宽（flex 的 grow 只分配剩余空间）。
	ui.Grid(c).Columns(2).Gap(16).Children(func() {
		// ---- 轻量级任务 ----
		ui.Column(c).Children(func() {
			card(c, t, func() {
				cardHeader(c, t, "轻量级任务", "进度显示在页头下方浅线处，界面照常可操作。", nil)
				ui.Progress(c, a.lightProgressOrIdle()).Height(6)
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, a.lightStatusText()).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1)
					if ui.PrimaryButton(c, "运行轻量任务").Clicked() {
						a.startLightTask("同步授权数据")
					}
				})
			})
		})

		// ---- 重量级任务 ----
		ui.Column(c).Children(func() {
			card(c, t, func() {
				cardHeader(c, t, "重量级任务", "执行期间全窗口遮罩，完成前禁止操作。", nil)
				ui.Text(c, "适用于重建索引、批量迁移这类会改写数据、不能被中断的操作。").
					FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, a.heavyStatusText()).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1)
					if ui.Button(c, "运行重量级任务").Clicked() {
						a.startHeavyTask("重建检索索引")
					}
				})
			})
		})
	})

	ui.Row(c).Padding(20, 0, 4).Children(func() {
		card(c, t, func() {
			cardHeader(c, t, "任务队列", a.homeSummary(), func() {
				if ui.Button(c, "清空已完成").Clicked() {
					a.clearFinishedTasks()
				}
			})
			a.taskTable(c, t)
		})
	})
}

// lightProgressOrIdle 空闲时进度条显示 0（不闪动）。
func (a *App) lightProgressOrIdle() float64 {
	if a.taskRunning {
		return a.taskProgress
	}
	return 0
}

// lightStatusText 轻量任务的当前状态文案。
func (a *App) lightStatusText() string {
	if a.taskRunning {
		return a.taskLabel + "：" + percentText(a.taskProgress)
	}
	return "空闲。点击右侧按钮运行一条 2 秒左右的示例任务。"
}

// heavyStatusText 重量级任务的当前状态文案。
func (a *App) heavyStatusText() string {
	if a.heavyRunning {
		return a.heavyLabel + "：" + percentText(a.heavyProgress)
	}
	return "空闲。示例任务约 4.4 秒，期间无法操作界面。"
}

// clearFinishedTasks 清掉已完成的记录，保留排队/运行中的。
func (a *App) clearFinishedTasks() {
	kept := a.tasks[:0]
	for _, r := range a.tasks {
		if r.status != "已完成" {
			kept = append(kept, r)
		}
	}
	a.tasks = kept
	a.toast("已清空完成记录")
}

// taskTable 任务队列表：名称 / 类型 / 状态 / 耗时。
func (a *App) taskTable(c *ui.Context, t *ui.Theme) {
	cols := []ui.TableColumn{
		{Title: "任务", Sortable: false},
		{Title: "类型", Width: 110},
		{Title: "状态", Width: 110},
		{Title: "耗时", Width: 100, Align: ui.End},
	}
	a.taskState.Key = func(i int) any { return i }
	a.taskState.Label = func(i int) string { return a.tasks[i].name }
	a.taskState.Selected = &a.taskSel
	ui.Table(c, &a.taskState, cols, len(a.tasks), func(row, col int) {
		r := a.tasks[row]
		switch col {
		case 0:
			ui.Text(c, r.name).SingleLine()
		case 1:
			ui.Text(c, r.kind).SingleLine()
		case 2:
			pill(c, t, r.status, statusColor(t, r.status))
		case 3:
			ui.Text(c, r.cost).SingleLine()
		}
	}).FillWidth()
}

// 供任务完成时把「耗时」写得更真实一点（重量级任务用）。
func costText(d time.Duration) string {
	return fmt.Sprintf("%.1fs", d.Seconds())
}
