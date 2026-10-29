package view

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
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
	// doneAt 完成的时刻：状态列据此脉冲一下（见 kit.PillPulse）。只有「已完成」
	// 才有值，其余为零值 —— 零值的语义是「这件事没发生过」，不画脉冲。
	doneAt time.Time
}

// demoTaskRows 任务队列的初始演示数据（贴近真实运维场景）。
func demoTaskRows() []taskRow {
	// doneAt 一律零值：这五条是历史记录，不是「刚刚完成」，不该脉冲。
	return []taskRow{
		{name: "同步授权数据", kind: "轻量级", status: "已完成", cost: "0.8s"},
		{name: "刷新证书链缓存", kind: "轻量级", status: "已完成", cost: "0.3s"},
		{name: "导出月度报表", kind: "轻量级", status: "已完成", cost: "2.1s"},
		{name: "重建检索索引", kind: "重量级", status: "已完成", cost: "18.4s"},
		{name: "清理过期凭证", kind: "重量级", status: "排队", cost: "—"},
	}
}

// pushTaskRow 在队列表头部插入一条记录（内存 + 数据库，最多保留 maxTaskRows 条）。
//
// 完成时刻用 time.Now() 而不是帧时钟：调用点都在帧外（后台任务结束、按钮回
// 调），拿不到 c；而脉冲只活 1.1 秒，两个时钟的毫秒级差异不影响观感。
func (a *App) pushTaskRow(name, kind, status string) {
	r := taskRow{name: name, kind: kind, status: status, cost: "刚刚"}
	if status == "已完成" {
		r.doneAt = time.Now()
	}
	a.tasks = append([]taskRow{r}, a.tasks...)
	if len(a.tasks) > maxTaskRows {
		a.tasks = a.tasks[:maxTaskRows]
	}
	a.persistTask(name, kind, status, "刚刚")
}

// taskPage 异步任务页：两张操作卡 + 任务队列表。
func (a *App) taskPage(c *ui.Context, t *ui.Theme) {
	// 两张卡用等宽栅格（kit.RowGrid）而非两个 Grow(1) 的 Column：
	// 栅格列宽严格相等，不会因为一侧文字更长而变宽；同一行还会自动等高。
	kit.RowGrid(c, 2, func() {
		// ---- 轻量级任务 ----
		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "轻量级任务", "进度显示在页头下方浅线处，界面照常可操作。", nil)
			ui.Progress(c, a.lightProgressOrIdle()).Height(6)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.lightStatusText()).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1)
				if ui.PrimaryButton(c, "运行轻量任务").Clicked() {
					a.startLightTask("同步授权数据")
				}
			})
		})

		// ---- 重量级任务 ----
		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "重量级任务", "执行期间全窗口遮罩，完成前禁止操作。", nil)
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

	// 任务队列表：直接作为页面区块（宽度由内容区统一决定，无需再套一层 Row）。
	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "任务队列", a.homeSummary(), func() {
			if ui.Button(c, "清空已完成").Clicked() {
				a.clearFinishedTasks()
			}
		})
		a.taskTable(c, t)
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

// clearFinishedTasks 清掉已完成的记录（内存 + 数据库），保留排队/运行中的。
func (a *App) clearFinishedTasks() {
	kept := a.tasks[:0]
	for _, r := range a.tasks {
		if r.status != "已完成" {
			kept = append(kept, r)
		}
	}
	a.tasks = kept
	a.persistClearFinished()
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
			// 刚完成的那一行脉冲一下，1.1 秒后自然停止（见 kit.PillPulse）。
			kit.PillPulse(c, t, r.status, kit.StatusColor(t, r.status), r.doneAt)
		case 3:
			ui.Text(c, r.cost).SingleLine()
		}
	}).FillWidth()
}

// 供任务完成时把「耗时」写得更真实一点（重量级任务用）。
func costText(d time.Duration) string {
	return fmt.Sprintf("%.1fs", d.Seconds())
}
