package view

import "github.com/egoist/mygo/ui"

// ---- 侧栏品牌头（替代原自定义标题栏） ----

func (a *App) brandHeader(c *ui.Context, t *ui.Theme) {
	ui.Row(c).Height(52).Shrink(0).Padding(0, t.Space(3), 0, t.Space(3)).
		Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Image(c, iconAtom).Size(22, 22).TextColor(t.Accent)
		ui.Text(c, "量子科学").FontSize(15).Bold()
	})
	ui.Row(c).Height(1).Shrink(0).Background(t.Border)
}

// ---- 左侧导航 ----

// sidebar 顶部品牌头固定，下方菜单区（ui.Sidebar 本质是一个 Scroll）用
// Grow(1) 吃掉侧栏剩余高度；菜单展开超出窗口高度时自动滚动。
func (a *App) sidebar(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Width(220).HeightPercent(100).Background(t.Surface).Children(func() {
		a.brandHeader(c, t)
		ui.Sidebar(c, &a.page, func() {
			// 主导航
			ui.SidebarItem(c, "home", iconHome, "首页")

			// 工具箱
			ui.SidebarSection(c, "工具箱", &a.openTools, func() {
				ui.SidebarItem(c, "gen-license", iconFileKey, "生成license")
				ui.SidebarItem(c, "gen-pwd", iconLock, "生成用户密码")
				ui.SidebarItem(c, "keypair", iconKey, "公钥私钥")
				ui.SidebarItem(c, "async", iconBolt, "异步任务")
				ui.SidebarItem(c, "debug", iconBug, "调试中心")
			})

			// 个人中心
			ui.SidebarSection(c, "个人中心", &a.openProfile, func() {
				ui.SidebarItem(c, "profile-basic", iconUser, "基本信息")
				ui.SidebarItem(c, "profile-security", iconShield, "安全设置")
			})

			// 组件演示
			ui.SidebarSection(c, "组件演示", &a.openComponents, func() {
				ui.SidebarItem(c, "comp-buttons", iconSquare, "按钮组件")
				ui.SidebarItem(c, "comp-inputs", iconType, "输入框组件")
				ui.SidebarItem(c, "comp-table", iconTable, "表格组件")
				ui.SidebarItem(c, "comp-card", iconIdCard, "卡片组件")
				ui.SidebarItem(c, "comp-tabs", iconPanel, "标签页组件")
			})

			// 功能演示
			ui.SidebarSection(c, "功能演示", &a.openFunctions, func() {
				ui.SidebarItem(c, "fn-chart", iconChart, "图表功能")
				ui.SidebarItem(c, "fn-form", iconClipboard, "表单功能")
				ui.SidebarItem(c, "fn-dialog", iconChat, "弹窗功能")
				ui.SidebarItem(c, "task", iconChecklist, "Task演示")
				ui.SidebarItem(c, "virtuallist", iconList, "虚拟列表演示")
				ui.SidebarItem(c, "async-dev", iconCpu, "异步开发")
				ui.SidebarItem(c, "grand", iconStar, "GRAND_DEMO")
			})

		}).Width(220).Grow(1)
		// 底部固定分隔线 + “设置”项（不随菜单滚动，始终贴在侧栏底部）
		ui.Row(c).Height(1).Shrink(0).Background(t.Border)
		a.settingsItem(c, t)
	})
}

// settingsItem 侧栏底部固定的“设置”项：不参与上方菜单区的滚动，
// 选中/点击行为与其它菜单项一致（点击切到设置页、选中态高亮）。
func (a *App) settingsItem(c *ui.Context, t *ui.Theme) {
	chosen := a.page == "settings"
	row := ui.Row(c).Shrink(0).
		AlignItems(ui.Center).
		Gap(t.Space(2)).
		Padding(t.Space(1.5), t.Space(2))
	if chosen {
		row.Background(t.SurfacePressed)
	} else if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	if row.Children(func() {
		ui.Image(c, iconGear).Size(16, 16).TextColor(t.Accent)
		ui.Text(c, "设置").Grow(1).SingleLine()
	}).Clicked() {
		a.page = "settings"
	}
}
