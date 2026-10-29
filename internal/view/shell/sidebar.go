package shell

import (
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/icons"
)

// 侧栏几何：让所有行（含高亮背景）只留 4px 边距、接近满宽，对应 javafx 版
// .sidebar{padding:4} + 行背景铺满整行。

// sideMargin 侧栏根容器的水平内边距：唯一一处水平内缩。
func sideMargin(t *ui.Theme) float32 { return t.Space(1) }

// contentInset 菜单项「内容」（图标 + 文字）相对近满宽高亮的左缩进：
// 背景铺满整行，内容再左缩 12px（对应 javafx 的 -fx-padding: 10px 0 10px 13px）。
func contentInset(t *ui.Theme) float32 { return t.Space(3) } // = 12px

// brandInset 品牌头图标的左缩进：在 contentInset 基础上回退
// (BrandIconSize-NavIconSize)/2，使品牌图标与菜单图标的中心点对齐。
func brandInset(t *ui.Theme) float32 {
	return contentInset(t) - (BrandIconSize-NavIconSize)/2
}

// Item 一个导航项（仅描述，不含行为）。
type Item struct {
	ID    string  // 页面 id，点击后写回 NavState.Page
	Label string  // 展开态文字 + 悬浮提示
	Icon  *ui.SVG // 16px 线性图标
}

// Group 一个可折叠分组；Title 为空的分组表示「置顶项，不画分组标题」。
type Group struct {
	Title string
	Items []Item
}

// NavState 侧栏的可变状态，由调用方（App）持有并以指针方式注入：
// 侧栏直接读写这些字段，因此不需要认识 App 类型。
type NavState struct {
	// Page 当前选中的页面 id；点击菜单项时由侧栏写回。
	Page *string
	// Collapsed 侧栏是否收缩为图标栏；点击品牌头时由侧栏翻转。
	Collapsed *bool
	// Folded 分组折叠状态（key 为分组标题；缺省视为展开）。
	Folded map[string]bool
}

// SidebarOptions 侧栏的静态配置。
type SidebarOptions struct {
	Title     string  // 品牌文字（收缩态不显示）
	BrandIcon *ui.SVG // 品牌标志
	Groups    []Group // 中部导航分组
	Footer    Item    // 底部固定项（设置），不随中部菜单滚动

	// Background 侧栏浮层填充色（有背景图时由调用方给半透明色）。
	Background ui.Color

	// OnCollapseChange 折叠状态变化回调：调用方据此持久化偏好。
	OnCollapseChange func(collapsed bool)
}

// Sidebar 绘制左侧导航栏。
//
// 结构：顶部品牌头固定 → 分割线 → 中部菜单（可滚动，滚动条隐藏）→ 分割线 →
// 底部固定项。中部菜单用 Scroll 包裹，窗口再矮也能滚到每一项。
func Sidebar(c *ui.Context, t *ui.Theme, st *NavState, o SidebarOptions) {
	width := float32(ExpandedWidth)
	if st.collapsed() {
		width = CollapsedWidth
	}
	ui.Column(c).Width(width).HeightPercent(100).Background(o.Background).
		Padding(t.Space(1), sideMargin(t), t.Space(1), sideMargin(t)).Children(func() {
		brandHeader(c, t, st, o)
		// 品牌头下分割线：上边距为 0，使浅线正好落在 y = Space(1)+HeaderHeight
		// 处，与内容区页头下的分割线保持同一水平线（见 content.go）。
		ui.Row(c).Height(1).Shrink(0).Margin(0, 0, t.Space(1), 0).Background(t.Border)

		scroll := ui.Scroll(c).Grow(1).
			// 纵向 6px 上下留白、项间 8px 行距（对齐 javafx spacing(8)+行 padding 10）。
			Padding(t.Space(1.5), 0, t.Space(1.5), 0).Gap(t.Space(2)).
			// 侧栏滚动条隐藏（内容区不受影响）。
			ScrollbarInsets(0, scrollbarHideInset, 0, 0)
		scroll.Children(func() {
			navMenu(c, t, st, o.Groups)
		})

		ui.Row(c).Height(1).Shrink(0).Margin(t.Space(1), 0, t.Space(1), 0).Background(t.Border)
		navFooter(c, t, st, o)
	})
}

// collapsed 读折叠态（未注入时按展开处理）。
func (st *NavState) collapsed() bool { return st.Collapsed != nil && *st.Collapsed }

// setPage 写回选中的页面 id（未注入时忽略）。
func (st *NavState) setPage(id string) {
	if st.Page != nil {
		*st.Page = id
	}
}

// toggle 翻转折叠态并通知调用方持久化。
func (st *NavState) toggle(o SidebarOptions) {
	if st.Collapsed == nil {
		return
	}
	*st.Collapsed = !*st.Collapsed
	if o.OnCollapseChange != nil {
		o.OnCollapseChange(*st.Collapsed)
	}
}

// ---- 品牌头（兼作收缩/展开开关） ----

func brandHeader(c *ui.Context, t *ui.Theme, st *NavState, o SidebarOptions) {
	collapsed := st.collapsed()
	row := ui.Row(c).Height(HeaderHeight).Shrink(0).AlignItems(ui.Center).Radius(t.Radius)
	if collapsed {
		row.Justify(ui.Center)
	} else {
		row.Padding(0, brandInset(t), 0, brandInset(t)).Gap(t.Space(2))
	}
	if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	row.Tooltip(o.Title + " · 点击" + map[bool]string{true: "展开", false: "收起"}[collapsed] + "侧栏")
	row.Children(func() {
		ui.Icon(c, o.BrandIcon).FontSize(BrandIconSize).TextColor(t.Accent)
		if !collapsed {
			ui.Text(c, o.Title).FontSize(15).Bold().Grow(1).SingleLine()
		}
	})
	if row.Clicked() {
		st.toggle(o)
	}
}

// ---- 中部导航 ----

// navMenu 中部导航：完全由传入的分组数据驱动，shell 不认识具体业务菜单。
func navMenu(c *ui.Context, t *ui.Theme, st *NavState, groups []Group) {
	collapsed := st.collapsed()
	for _, g := range groups {
		// 无标题分组 = 首页这类置顶项，不画分组标题/分割线。
		if g.Title == "" {
			for _, it := range g.Items {
				navRow(c, t, st, it, collapsed)
			}
			continue
		}

		if collapsed {
			// 收缩态：用组间细分割线提示分组，不占文字宽度。
			ui.Row(c).Height(1).Shrink(0).
				Margin(t.Space(2), t.Space(1.5), t.Space(2), t.Space(1.5)).
				Background(t.Border)
		} else {
			sectionHeader(c, t, st, g.Title)
		}

		if collapsed || !st.Folded[g.Title] {
			for _, it := range g.Items {
				navRow(c, t, st, it, collapsed)
			}
		}
	}
}

// sectionHeader 展开态的分组标题（纯文字）：点击折叠/展开，右侧箭头提示状态。
// 悬停高亮、圆角、铺满程度与菜单项完全一致；文字左缩与菜单图标对齐。
func sectionHeader(c *ui.Context, t *ui.Theme, st *NavState, title string) {
	head := ui.Row(c).Shrink(0).AlignItems(ui.Center).
		Padding(t.Space(2.5), 0, t.Space(2.5), contentInset(t)).Radius(t.Radius)
	if head.Hovered() {
		head.Background(t.SurfaceHover)
	}
	if head.Clicked() {
		if st.Folded == nil {
			st.Folded = map[string]bool{}
		}
		st.Folded[title] = !st.Folded[title]
	}
	folded := st.Folded[title]
	head.Children(func() {
		ui.Text(c, title).FontSize(t.FontSize - 1).FontWeight(600).
			TextColor(t.TextMuted).Grow(1).SingleLine()
		ui.Icon(c, icons.Chevron(!folded)).FontSize(ChevronSize).TextColor(t.TextMuted)
	})
}

// navRow 单个导航项。
//   - 展开态：背景铺满整行（仅由根容器的 4px sideMargin 收边），内容左缩 contentInset；
//     悬停/选中时背景只在该行范围内着色。
//   - 收缩态：整行作为点击/悬停命中区，内部图标药丸居中，悬浮提示菜单名。
func navRow(c *ui.Context, t *ui.Theme, st *NavState, it Item, collapsed bool) {
	chosen := st.Page != nil && *st.Page == it.ID

	if collapsed {
		outer := ui.Row(c).Shrink(0).AlignItems(ui.Center).Justify(ui.Center)
		outer.Children(func() {
			pill := ui.Row(c).AlignItems(ui.Center).
				Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius)
			if chosen {
				pill.Background(t.SurfacePressed)
			} else if outer.Hovered() {
				pill.Background(t.SurfaceHover)
			}
			pill.Children(func() {
				ui.Icon(c, it.Icon).FontSize(NavIconSize).TextColor(t.Accent)
			})
		})
		outer.Tooltip(it.Label)
		if outer.Clicked() {
			st.setPage(it.ID)
		}
		return
	}

	row := ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(t.Space(2)).
		Padding(t.Space(2.5), 0, t.Space(2.5), contentInset(t)).Radius(t.Radius)
	if chosen {
		row.Background(t.SurfacePressed)
	} else if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	row.Children(func() {
		ui.Icon(c, it.Icon).FontSize(NavIconSize).TextColor(t.Accent)
		ui.Text(c, it.Label).Grow(1).SingleLine().FontSize(t.FontSize)
	})
	if row.Clicked() {
		st.setPage(it.ID)
	}
}

// ---- 底部固定项（设置） ----

func navFooter(c *ui.Context, t *ui.Theme, st *NavState, o SidebarOptions) {
	it := o.Footer
	if it.ID == "" {
		return
	}
	collapsed := st.collapsed()
	chosen := st.Page != nil && *st.Page == it.ID

	if collapsed {
		outer := ui.Row(c).Shrink(0).AlignItems(ui.Center).Justify(ui.Center)
		outer.Children(func() {
			pill := ui.Row(c).AlignItems(ui.Center).
				Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius)
			if chosen {
				pill.Background(t.SurfacePressed)
			} else if outer.Hovered() {
				pill.Background(t.SurfaceHover)
			}
			pill.Children(func() {
				ui.Icon(c, it.Icon).FontSize(NavIconSize).TextColor(t.Accent)
			})
		})
		outer.Tooltip(it.Label)
		if outer.Clicked() {
			st.setPage(it.ID)
		}
		return
	}

	row := ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(t.Space(2)).
		Padding(t.Space(2.5), 0, t.Space(2.5), contentInset(t)).Radius(t.Radius)
	if chosen {
		row.Background(t.SurfacePressed)
	} else if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	row.Children(func() {
		ui.Icon(c, it.Icon).FontSize(NavIconSize).TextColor(t.Accent)
		ui.Text(c, it.Label).Grow(1).SingleLine()
	})
	if row.Clicked() {
		st.setPage(it.ID)
	}
}
