package view

import "github.com/egoist/mygo/ui"

// 侧栏尺寸与图标规格。
const (
	sidebarExpandedWidth  = 220 // 展开态宽度（与历史布局一致）
	sidebarCollapsedWidth = 64  // 收缩态（图标栏 / rail）宽度
	navIconSize           = 16  // 菜单项与设置项图标尺寸；品牌头与之对齐
	brandIconSize         = 20  // 品牌标志略大，居中后仍与菜单图标中心对齐
	chevronSize           = 14  // 分组标题右侧箭头尺寸（比原文字字形更大更清晰）
)

// navInset 导航项（菜单项 / 设置项）图标的水平内缩：让它们的图标中心与
// 品牌头图标中心落在同一条竖线上（对齐基准）。
func navInset(t *ui.Theme) float32 { return t.Space(2.5) + t.Space(2) }

// brandInset 品牌头图标内缩：在 navInset 基础上回退 (brandIconSize-navIconSize)/2，
// 使 20px 品牌图标与 16px 菜单图标的中心点对齐。
func brandInset(t *ui.Theme) float32 {
	return navInset(t) - (brandIconSize-navIconSize)/2
}

// collapsedInset 收缩态下图标在 rail 内的居中内缩：左右相等，使图标水平居中。
func collapsedInset(iconSize float32) float32 {
	return (sidebarCollapsedWidth - iconSize) / 2
}

// navItemDef 单个导航项的静态描述。
type navItemDef struct {
	id    string
	icon  *ui.SVG
	label string
}

// ---- 侧栏整体 ----

// sidebar 顶部品牌头固定，中部菜单区用 Scroll 包裹（超出窗口高度自动滚动，
// 滚动条已透明化但仍可滚动），底部“设置”项固定不随菜单滚动。
// a.sidebarCollapsed 控制整体展开/收缩：
//   - 展开：220px，展示「图标 + 文字」，同原布局；
//   - 收缩：64px（rail），仅展示图标，悬浮提示菜单名。
func (a *App) sidebar(c *ui.Context, t *ui.Theme) {
	width := float32(sidebarExpandedWidth)
	if a.sidebarCollapsed {
		width = sidebarCollapsedWidth
	}
	ui.Column(c).Width(width).HeightPercent(100).Background(t.Surface).Children(func() {
		a.brandHeader(c, t)
		ui.Row(c).Height(1).Shrink(0).Background(t.Border)
		scroll := ui.Scroll(c).Grow(1)
		if a.sidebarCollapsed {
			// 收缩态项更紧凑，项间留 1 份间距；横向不内缩（图标由 Justify 居中）。
			scroll.Padding(t.Space(1.5), 0, t.Space(1.5), 0).Gap(t.Space(1))
		} else {
			// 展开态：横向内缩 navInset，使菜单项/分组标题的“药丸”背景不贴边，
			// 四周自然留白；项间留 0.5 份间距。
			scroll.Padding(t.Space(1.5), navInset(t), t.Space(1.5), navInset(t)).Gap(t.Space(0.5))
		}
		scroll.Children(func() {
			a.navMenu(c, t)
		})
		ui.Row(c).Height(1).Shrink(0).Background(t.Border)
		a.settingsItem(c, t)
	})
}

// ---- 品牌头（兼作收缩/展开开关） ----

func (a *App) brandHeader(c *ui.Context, t *ui.Theme) {
	collapsed := a.sidebarCollapsed
	row := ui.Row(c).Height(52).Shrink(0).AlignItems(ui.Center).Radius(t.Radius)
	if collapsed {
		row.Padding(0, collapsedInset(brandIconSize), 0, collapsedInset(brandIconSize)).Justify(ui.Center)
	} else {
		row.Padding(0, brandInset(t), 0, brandInset(t)).Gap(t.Space(2))
	}
	if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	row.Tooltip("量子科学 · 点击" + map[bool]string{true: "展开", false: "收起"}[collapsed] + "侧栏")
	row.Children(func() {
		ui.Icon(c, iconAtom).FontSize(brandIconSize).TextColor(t.Accent)
		if !collapsed {
			ui.Text(c, "量子科学").FontSize(15).Bold().Grow(1).SingleLine()
		}
	})
	if row.Clicked() {
		a.toggleSidebar()
	}
}

// ---- 中部导航 ----

func (a *App) navMenu(c *ui.Context, t *ui.Theme) {
	collapsed := a.sidebarCollapsed

	// 主导航：首页（无分组标题）。
	a.navRow(c, t, "home", iconHome, "首页", collapsed)

	// 各分组：展开态显示可点击折叠的标题；收缩态改用细分割线分隔。
	groups := []struct {
		title string
		open  *bool
		items []navItemDef
	}{
		{"工具箱", &a.openTools, []navItemDef{
			{"gen-license", iconFileKey, "生成license"},
			{"gen-pwd", iconLock, "生成用户密码"},
			{"keypair", iconKey, "公钥私钥"},
			{"async", iconBolt, "异步任务"},
			{"debug", iconBug, "调试中心"},
		}},
		{"个人中心", &a.openProfile, []navItemDef{
			{"profile-basic", iconUser, "基本信息"},
			{"profile-security", iconShield, "安全设置"},
		}},
		{"组件演示", &a.openComponents, []navItemDef{
			{"comp-buttons", iconSquare, "按钮组件"},
			{"comp-inputs", iconType, "输入框组件"},
			{"comp-table", iconTable, "表格组件"},
			{"comp-card", iconIdCard, "卡片组件"},
			{"comp-tabs", iconPanel, "标签页组件"},
		}},
		{"功能演示", &a.openFunctions, []navItemDef{
			{"fn-chart", iconChart, "图表功能"},
			{"fn-form", iconClipboard, "表单功能"},
			{"fn-dialog", iconChat, "弹窗功能"},
			{"task", iconChecklist, "Task演示"},
			{"virtuallist", iconList, "虚拟列表演示"},
			{"async-dev", iconCpu, "异步开发"},
			{"grand", iconStar, "GRAND_DEMO"},
		}},
	}

	for _, g := range groups {
		if collapsed {
			// 组间细分割线，提示不同分组。
			ui.Row(c).Height(1).Shrink(0).
				Margin(t.Space(2), t.Space(2), t.Space(2), t.Space(2)).
				Background(t.Border)
		} else {
			a.sectionHeader(c, t, g.title, g.open)
		}
		if collapsed || *g.open {
			for _, it := range g.items {
				a.navRow(c, t, it.id, it.icon, it.label, collapsed)
			}
		}
	}
}

// navRow 单个导航项。
//   - 展开态：药丸即本行——内缩对齐图标、上下内边距留呼吸、圆角背景；
//     悬停/选中时背景只在药丸范围内着色，四周自然留白。
//   - 收缩态：整行作为点击/悬停命中区，内部图标药丸居中，左右留呼吸；
//     悬停/选中时仅图标药丸着色，悬浮提示菜单名。
func (a *App) navRow(c *ui.Context, t *ui.Theme, id string, icon *ui.SVG, label string, collapsed bool) {
	chosen := a.page == id

	if collapsed {
		outer := ui.Row(c).Shrink(0).AlignItems(ui.Center).Justify(ui.Center)
		outer.Children(func() {
			pill := ui.Row(c).AlignItems(ui.Center).
				Padding(t.Space(1), t.Space(1.5)).Radius(t.Radius)
			if chosen {
				pill.Background(t.SurfacePressed)
			} else if outer.Hovered() {
				pill.Background(t.SurfaceHover)
			}
			pill.Children(func() {
				ui.Icon(c, icon).FontSize(navIconSize).TextColor(t.Accent)
			})
		})
		outer.Tooltip(label)
		if outer.Clicked() {
			a.page = id
		}
		return
	}

	row := ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(t.Space(2)).
		Padding(t.Space(1.5), 0, t.Space(1.5), 0).Radius(t.Radius)
	if chosen {
		row.Background(t.SurfacePressed)
	} else if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	row.Children(func() {
		ui.Icon(c, icon).FontSize(navIconSize).TextColor(t.Accent)
		ui.Text(c, label).Grow(1).SingleLine().FontSize(t.FontSize)
	})
	if row.Clicked() {
		a.page = id
	}
}

// sectionHeader 展开态的分组标题：点击折叠/展开，右侧箭头提示状态。
// 同样采用药丸式悬停背景，与菜单项视觉一致。
func (a *App) sectionHeader(c *ui.Context, t *ui.Theme, title string, open *bool) {
	head := ui.Row(c).Shrink(0).AlignItems(ui.Center).Margin(t.Space(1.5), 0, 0, 0).
		Padding(t.Space(1), 0, t.Space(1), 0).Radius(t.Radius)
	if head.Hovered() {
		head.Background(t.SurfaceHover)
	}
	if head.Clicked() {
		*open = !*open
	}
	head.Children(func() {
		ui.Text(c, title).FontSize(t.FontSize - 1).FontWeight(600).
			TextColor(t.TextMuted).Grow(1).SingleLine()
		ui.Icon(c, openChevron(*open)).FontSize(chevronSize).TextColor(t.TextMuted)
	})
}

// ---- 底部设置项（始终固定，不随菜单滚动） ----

func (a *App) settingsItem(c *ui.Context, t *ui.Theme) {
	collapsed := a.sidebarCollapsed
	chosen := a.page == "settings"

	if collapsed {
		outer := ui.Row(c).Shrink(0).AlignItems(ui.Center).Justify(ui.Center)
		outer.Children(func() {
			pill := ui.Row(c).AlignItems(ui.Center).
				Padding(t.Space(1), t.Space(1.5)).Radius(t.Radius)
			if chosen {
				pill.Background(t.SurfacePressed)
			} else if outer.Hovered() {
				pill.Background(t.SurfaceHover)
			}
			pill.Children(func() {
				ui.Icon(c, iconGear).FontSize(navIconSize).TextColor(t.Accent)
			})
		})
		outer.Tooltip("设置")
		if outer.Clicked() {
			a.page = "settings"
		}
		return
	}

	row := ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(t.Space(2)).
		Padding(t.Space(1.5), 0, t.Space(1.5), 0).Radius(t.Radius)
	if chosen {
		row.Background(t.SurfacePressed)
	} else if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	row.Children(func() {
		ui.Icon(c, iconGear).FontSize(navIconSize).TextColor(t.Accent)
		ui.Text(c, "设置").Grow(1).SingleLine()
	})
	if row.Clicked() {
		a.page = "settings"
	}
}
