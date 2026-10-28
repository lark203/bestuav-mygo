package view

import "github.com/egoist/mygo/ui"

// 内容区几何常量。
//
// 三条约束共同决定这两条分割线「看起来是一条线」：
//   - 侧栏：根容器上内边距 Space(1)=4px + 品牌头 headerHeight + 分割线上边距 0；
//   - 内容区：根容器上内边距同为 Space(1)，页头同为 headerHeight，分割线紧随其后；
//   - 两者相加都等于 4 + 52 = 56px，故浅线落在同一水平线上。
const (
	headerHeight = 52 // 页头高度（与侧栏品牌头一致）
	dividerSlot  = 3  // 分割线槽位高度：常态画 1px 浅线，任务执行时同槽位画进度条
	contentPadX  = 28 // 内容区左右内边距（页头与页面内容共用，左边缘对齐）
)

// content 右侧内容区，自上而下分三段：
//
//  1. 固定高度的页头：页面标题 + 右侧「自定义图标按钮」工具条。工具条放在横向
//     滚动容器里，按钮多到放不下时可横向滚动查看（滚动条已全局透明化，不可见）。
//  2. 分割线槽位：一条 1px 浅线；执行轻量级任务时，同一槽位改画进度条
//     （对应 javafx 版 HeaderBar 底部那条进度条）。槽位高度固定，故进度条出现
//     与消失都不会让下方内容跳动。
//  3. 可滚动的页面内容。
func (a *App) content(c *ui.Context, t *ui.Theme) {
	// 内容区浮层：无背景图时是主题默认背景（不透明），有背景图时按「内容区
	// 不透明度」半透明，让背景透出（alpha 由 applyBackdropTheme 每帧设定）。
	col := ui.Column(c).Grow(1).HeightPercent(100).Padding(t.Space(1), 0, 0, 0).
		Background(a.windowSurface(t))
	col.Children(func() {
		a.contentHeader(c, t)
		a.contentDivider(c, t)
		// TrackScroll 让每页记住自己的滚动位置：切走再切回停在原处（缓存命中），
		// 页面被销毁后自动归零（等价于重新加载）。见 pagelife.go。
		ui.Scroll(c).Grow(1).Padding(24, contentPadX).TrackScroll(a.ScrollFor(a.page)).
			Children(func() {
				a.pageBody(c, t)
			})
	})
}

// pageBody 渲染当前页面：注册表里有 build 就用它，否则落到占位页。
//
// 这是「菜单 → 页面」唯一的映射点：新增页面只需改 registry.go 一处。
func (a *App) pageBody(c *ui.Context, t *ui.Theme) {
	if it, ok := pageByID(a.page); ok && it.build != nil {
		it.build(a, c, t)
		return
	}
	placeholderPage(c, t, pageTitle(a.page), a.page)
}

// contentHeader 页头：左标题（固定，不参与滚动），右侧工具条。
func (a *App) contentHeader(c *ui.Context, t *ui.Theme) {
	ui.Row(c).Height(headerHeight).Shrink(0).AlignItems(ui.Center).
		Padding(0, contentPadX).Gap(t.Space(2)).Children(func() {
		// 标题来自注册表（registry.go），不再单独维护一份 pageTitles。
		ui.Text(c, pageTitle(a.page)).FontSize(16).Bold().Shrink(0).SingleLine()

		// 正在执行轻量任务时，标题右侧显示任务名（与进度条呼应）。
		if a.taskRunning && a.taskLabel != "" {
			ui.Text(c, a.taskLabel).FontSize(t.FontSize - 2).
				TextColor(t.TextMuted).Shrink(0).SingleLine()
		}

		ui.Row(c).Grow(1) // 弹性空白，把工具条推到右端
		a.headerTools(c, t)
	})
}

// headerTools 页头右侧的自定义图标按钮区。
//
// 布局要点：外层 ScrollHorizontal 用 Shrink(1) 参与伸缩——按钮少时按内容宽度
// 靠右排列；空间不足时被压缩到可用宽度，超出部分进横向滚动，而不是把标题挤出
// 屏幕或溢出窗口。
func (a *App) headerTools(c *ui.Context, t *ui.Theme) {
	tools := pageToolsFor(a.page)
	if len(tools) == 0 {
		return
	}
	ui.ScrollHorizontal(c).Shrink(1).Children(func() {
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

// contentDivider 分割线槽位：常态 1px 浅线，任务执行时同槽位画进度条。
func (a *App) contentDivider(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Height(dividerSlot).Shrink(0).Children(func() {
		if a.taskRunning {
			// 进度条与浅线同槽位、同起点：任务跑起来时“线变成进度条”，
			// 结束后再变回线，视觉上不产生位移。
			ui.Progress(c, a.taskProgress).Height(dividerSlot).Radius(0)
			return
		}
		ui.Row(c).Height(1).Shrink(0).Background(t.Border)
	})
}

// ---- 页头工具条的数据定义 ----

// toolAction 页头图标按钮。
//
//	icon：图标（icons.go 里的 *ui.SVG，16px 线性，跟随主题文本色）
//	label：动作名（默认行为下作为轻量任务名显示在页头标题右侧）
//	tip：悬浮提示
//	run：点击行为；为 nil 时走「演示一条轻量任务」，新增页面可直接只填前三个字段
type toolAction struct {
	icon  *ui.SVG
	label string
	tip   string
	run   func(a *App)
}
