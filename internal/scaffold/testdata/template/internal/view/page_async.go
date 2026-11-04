package view

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
)

// 功能演示：异步数据。
//
// 这页只写两件事——「怎么加载」与「怎么渲染」。加载中卡片、失败卡片 + 重试、
// 迟到结果丢弃、每次进入本页自动重载，全部由 pageAsync 负责（见 async.go）。
// 想在新业务页里用同一套：抄下面这十来行即可。

// asyncDemoPageID 页面 id（注册表与页面状态共用，避免字符串散落）。
const asyncDemoPageID = "fn-async"

// asyncDemoDelay 让演示页的加载「看得见」；测试里置 0，避免等 0.7 秒。
var asyncDemoDelay = 700 * time.Millisecond

// asyncDemoState 演示页的可调开关（页面私有状态，页面销毁即丢弃）。
type asyncDemoState struct {
	fail  bool // 下一次加载模拟失败
	empty bool // 下一次加载返回空列表
}

// asyncDemoRow 演示数据的一行。
type asyncDemoRow struct {
	name   string
	kind   string
	status string
	cost   string
}

// demoAsyncRows 每次加载返回的数据（重新加载时重新生成，便于观察「进入即刷新」）。
func demoAsyncRows() []asyncDemoRow {
	return []asyncDemoRow{
		{name: "同步授权数据", kind: "轻量级", status: "已完成", cost: "0.8s"},
		{name: "重建检索索引", kind: "重量级", status: "已完成", cost: "18.4s"},
		{name: "导出月度报表", kind: "轻量级", status: "运行中", cost: "2.1s"},
		{name: "清理过期凭证", kind: "重量级", status: "排队", cost: "—"},
	}
}

// asyncDemoPage 异步数据演示页。
func (a *App) asyncDemoPage(c *ui.Context, t *ui.Theme) {
	st := pageData(a, asyncDemoPageID, "demo", func() asyncDemoState { return asyncDemoState{} })

	// load 在后台协程里跑：只捕获这一帧的值拷贝，不去读会被同时修改的页面状态。
	fail, empty := st.fail, st.empty
	op := pageAsync(a, "rows", func() ([]asyncDemoRow, error) {
		if asyncDemoDelay > 0 {
			time.Sleep(asyncDemoDelay)
		}
		if fail {
			return nil, fmt.Errorf("模拟失败：服务暂时不可用")
		}
		if empty {
			return nil, nil
		}
		return demoAsyncRows(), nil
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "异步数据",
			"加载中 / 失败重试 / 结果只采纳最新一次——都由 pageAsync 处理", nil)
		ui.Row(c).Wrap().Gap(12).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "重新加载").Clicked() {
				op.Reload()
			}
			ui.Checkbox(c, &st.fail, "下次加载失败")
			ui.Checkbox(c, &st.empty, "下次返回空列表")
		})
		ui.Text(c, "切到别的页面再回来，也会自动重新加载一次（缓存页的「进入即刷新」）。").
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})

	op.View(c, t, func(rows []asyncDemoRow) {
		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "任务列表",
				fmt.Sprintf("共 %d 条 · 数据在后台加载，界面不会卡住", len(rows)), nil)
			if len(rows) == 0 {
				kit.EmptyHint(c, t, "没有数据（空列表也是一种正常结果）")
				return
			}
			for _, r := range rows {
				ui.Row(c).Gap(12).AlignItems(ui.Center).Padding(6, 0).Children(func() {
					ui.Text(c, r.name).Grow(1).SingleLine()
					ui.Text(c, r.kind).Width(64).Shrink(0).
						FontSize(t.FontSize - 2).TextColor(t.TextMuted)
					kit.Pill(c, t, r.status, kit.StatusColor(t, r.status))
					ui.Text(c, r.cost).Width(56).Shrink(0).TextColor(t.TextMuted).
						FontSize(t.FontSize - 2)
				})
			}
		})
	})
}
