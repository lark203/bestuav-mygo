package view

import (
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/view/shell"
)

// 本文件是页头右侧的「自定义图标按钮」工具条。
//
// 它虽然属于骨架的一部分（页头由 shell.Content 布局），但按钮数据来自菜单
// 注册表（registry.go 的 navItem.tools），所以渲染逻辑留在 view，由 shell 通过
// ContentOptions.Tools 回调调用。

// toolAction 页头图标按钮。
//
//	icon：图标（internal/ui/icons 里的 *ui.SVG，16px 线性，跟随主题文本色）
//	label：动作名（默认行为下作为轻量任务名显示在页头标题右侧）
//	tip：悬浮提示
//	run：点击行为；为 nil 时走「演示一条轻量任务」，新增页面可直接只填前三个字段
type toolAction struct {
	icon  *ui.SVG
	label string
	tip   string
	run   func(a *App)
}

// headerTools 页头右侧的图标按钮区。
//
// 布局要点：外层横向滚动容器用 Shrink(1) 参与伸缩——按钮少时按内容宽度靠右
// 排列；空间不足时被压缩到可用宽度，超出部分进横向滚动，而不是把标题挤出屏幕
// 或溢出窗口。横向滚动条同样隐藏（页头是固定高度的一条，出现滚动条会破坏对齐）。
func (a *App) headerTools(c *ui.Context, t *ui.Theme) {
	tools := pageToolsFor(a.page)
	if len(tools) == 0 {
		return
	}
	ui.ScrollHorizontal(c).Shrink(1).
		// 大下内缩 → 横向滚动条被推到页头上边界之外（不可见）。
		ScrollbarInsets(0, 0, shell.DividerSlot+400, 0).
		Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(t.Space(1)).Children(func() {
				for _, tool := range tools {
					a.toolButton(c, t, tool)
				}
			})
		})
}

// toolButton 单个页头图标按钮（28×28 点击区，悬停有圆角底色，悬浮提示说明用途）。
func (a *App) toolButton(c *ui.Context, t *ui.Theme, tool toolAction) {
	btn := ui.Row(c).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).
		Width(28).Height(28).Radius(t.Radius)
	if btn.Hovered() {
		btn.Background(t.SurfaceHover)
	}
	btn.Tooltip(tool.tip)
	btn.Children(func() {
		ui.Icon(c, tool.icon).FontSize(16).TextColor(t.TextMuted)
	})
	if btn.Clicked() {
		// 有具体动作就执行（如「演示彩带」「复制结果」）；没有则默认演示一条
		// 轻量任务（进度画在页头下方的分割线槽位里），便于新页面直接挂按钮。
		if tool.run != nil {
			tool.run(a)
		} else {
			a.startLightTask(tool.label)
		}
	}
}
