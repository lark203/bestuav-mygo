package view

import (
	"context"
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
)

// 功能演示：异步数据。
//
// 这页只写两件事——「怎么加载」与「怎么渲染」。加载中卡片、失败卡片 + 重试、
// 迟到结果丢弃、页面销毁时取消加载、每次进入本页自动重载，全部由 pageAsync 负责
// （见 async.go）。想在新业务页里用同一套：抄下面这十来行即可。
//
// 它同时演示了「页面级设置」：设置 → 页面 里的「演示数据条数」由 asyncDemoSettings
// 声明，值存在页面私有状态里（a.PrefsFor）。

// asyncDemoPageID 页面 id（注册表与页面状态共用，避免字符串散落）。
const asyncDemoPageID = "fn-async"

// asyncDemoDelay 让演示页的加载「看得见」；测试里置 0 或小值。
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

// asyncDemoRowCount 演示数据条数：页面级设置（默认 4，范围 1–20），存页面私有状态。
func asyncDemoRowCount(a *App) int {
	n := 4
	a.PrefsFor(asyncDemoPageID).Get("rows", &n) // 没存过就保持默认
	if n < 1 {
		n = 1
	}
	if n > 20 {
		n = 20
	}
	return n
}

// demoAsyncRows 生成 n 行演示数据（循环取基础行，条数由页面设置决定）。
func demoAsyncRows(n int) []asyncDemoRow {
	base := []asyncDemoRow{
		{name: "同步授权数据", kind: "轻量级", status: "已完成", cost: "0.8s"},
		{name: "重建检索索引", kind: "重量级", status: "已完成", cost: "18.4s"},
		{name: "导出月度报表", kind: "轻量级", status: "运行中", cost: "2.1s"},
		{name: "清理过期凭证", kind: "重量级", status: "排队", cost: "—"},
		{name: "刷新证书链缓存", kind: "轻量级", status: "已完成", cost: "0.3s"},
		{name: "归档审计日志", kind: "轻量级", status: "已完成", cost: "1.2s"},
		{name: "校验签名清单", kind: "重量级", status: "已完成", cost: "4.5s"},
		{name: "预热页面缓存", kind: "轻量级", status: "运行中", cost: "0.6s"},
	}
	out := make([]asyncDemoRow, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, base[i%len(base)])
	}
	return out
}

// asyncDemoPage 异步数据演示页。
func (a *App) asyncDemoPage(c *ui.Context, t *ui.Theme) {
	st := pageData(a, asyncDemoPageID, "demo", func() asyncDemoState { return asyncDemoState{} })

	// load 在后台协程里跑：只捕获这一帧的值拷贝，不去读会被同时修改的页面状态。
	fail, empty, rows := st.fail, st.empty, asyncDemoRowCount(a)
	op := pageAsync(a, "rows", func(ctx context.Context) ([]asyncDemoRow, error) {
		if asyncDemoDelay > 0 {
			// 尊重取消：页面销毁时不必等演示用的延时走完。
			select {
			case <-time.After(asyncDemoDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		if fail {
			return nil, fmt.Errorf("模拟失败：服务暂时不可用")
		}
		if empty {
			return nil, nil
		}
		return demoAsyncRows(rows), nil
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
		ui.Text(c, "切到别的页面再回来，也会自动重新加载一次（缓存页的「进入即刷新」）；"+
			"条数可在「设置 → 页面」里改。").
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

// asyncDemoSettings 页面级设置（设置 → 页面）：演示数据条数。
//
// 这是「页面自己声明设置项」的样例：设置页只按注册表把区块画出来，字段与持久化
// 都由页面自己负责（a.PrefsFor 读写页面私有状态）。
func (a *App) asyncDemoSettings(c *ui.Context, t *ui.Theme) {
	n := float64(asyncDemoRowCount(a))
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "演示数据条数").Width(kit.SettingsLabelWidth).Shrink(0).
			TextColor(t.TextMuted)
		ui.Stepper(c, &n, 1, 20, 1)
		ui.Text(c, fmt.Sprintf("%d 条", int(n))).TextColor(t.TextMuted)
	})
	// 绑定值在本帧构造后已更新：与已存的值不同就写回（只写一次，不会每帧写）。
	if got := int(n); got != asyncDemoRowCount(a) {
		if err := a.PrefsFor(asyncDemoPageID).Set("rows", got); err != nil {
			a.toast("保存页面设置失败：" + err.Error())
		}
	}
	ui.Text(c, "只影响「异步数据」演示页生成的条数；存在页面私有状态里，"+
		"可在「关于与诊断 → 数据与诊断」清理。").
		FontSize(t.FontSize - 2).TextColor(t.TextMuted)
}
