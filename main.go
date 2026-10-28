package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// ============================================================================
// 图标：简洁线条 SVG，stroke=currentColor 自动跟随主题文字色（选中时变强调色）。
// ============================================================================

var (
	iconHome    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" d="M3 11l9-8 9 8M5 10v10h14V10"/></svg>`))
	iconFolder  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>`))
	iconKey     = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="8" cy="8" r="4" fill="none" stroke="currentColor" stroke-width="2"/><path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" d="M11 11l8 8M16 16l3-3M14 14l3-3"/></svg>`))
	iconBug     = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><rect x="6" y="6" width="12" height="12" rx="3" fill="none" stroke="currentColor" stroke-width="2"/><path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" d="M9 3v3M15 3v3M9 18v3M15 18v3M3 9h3M3 15h3M18 9h3M18 15h3"/></svg>`))
	iconBolt    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M13 2L4 14h7l-1 8 9-12h-7z"/></svg>`))
	iconBox     = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" d="M3 7l9-4 9 4v10l-9 4-9-4z"/><path fill="none" stroke="currentColor" stroke-width="2" d="M3 7l9 4 9-4M12 11v10"/></svg>`))
	iconList    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" d="M8 6h12M8 12h12M8 18h12M4 6h.01M4 12h.01M4 18h.01"/></svg>`))
	iconStar    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M12 3l2.6 5.6 6.1.7-4.5 4.2 1.2 6L12 16.8 6.6 19.5l1.2-6L3.3 9.3l6.1-.7z"/></svg>`))
	iconGear    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3.5" fill="none" stroke="currentColor" stroke-width="2"/><path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" d="M12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M19 5l-2 2M7 17l-2 2"/></svg>`))
	iconBell    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" d="M6 9a6 6 0 1 1 12 0c0 5 2 6 2 6H4s2-1 2-6M10 20a2 2 0 0 0 4 0"/></svg>`))
	iconGlobe   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2"/><path fill="none" stroke="currentColor" stroke-width="2" d="M3 12h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18"/></svg>`))
	iconPackage = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" d="M3 8l9-4 9 4-9 4z"/><path fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" d="M3 8v8l9 4 9-4V8"/></svg>`))
	iconCpu     = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><rect x="6" y="6" width="12" height="12" rx="2" fill="none" stroke="currentColor" stroke-width="2"/><path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" d="M9 3v3M15 3v3M9 18v3M15 18v3M3 9h3M3 15h3M18 9h3M18 15h3"/></svg>`))
	// 左上角标志用：原子（量子）图标，白色描边，放在紫色方块里。
	iconAtom = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g fill="none" stroke="currentColor" stroke-width="1.6"><ellipse cx="12" cy="12" rx="9" ry="3.6"/><ellipse cx="12" cy="12" rx="9" ry="3.6" transform="rotate(60 12 12)"/><ellipse cx="12" cy="12" rx="9" ry="3.6" transform="rotate(120 12 12)"/></g><circle cx="12" cy="12" r="2.1" fill="currentColor"/></svg>`))
)

// 产品数据（mock，对应截图“最近产品”表）。
type product struct {
	name     string
	category string
	price    string
}

// app 持有窗口状态；view 在主线程按帧从它构建界面。
type app struct {
	page string // 当前选中页 id

	// 侧栏文件夹开合状态
	openIcons, openProfile, openComponents, openFunctions bool

	// 首页
	welcome  string
	products []product
	showDlg  bool

	// 最近产品表状态
	plist  ui.ListState
	pSel   int
	pSort  ui.SortOrder
}

func newApp() *app {
	return &app{
		page:    "home",
		welcome: "欢迎来到AtlantaFX 世界！",
		products: []product{
			{"笔记本电脑", "电子产品", "¥ 8999"},
			{"机械键盘", "外设", "¥ 699"},
			{"无线鼠标", "外设", "¥ 299"},
			{"显示器", "电子产品", "¥ 2499"},
		},
	}
}

// pageTitles 各页面的标题（用于右侧页头与占位页）。
var pageTitles = map[string]string{
	"home":            "首页",
	"icons-font":      "字体图标",
	"icons-gallery":   "图标清单",
	"gen-license":     "生成license",
	"gen-pwd":         "生成用户密码",
	"profile-basic":   "基本信息",
	"profile-security": "安全设置",
	"debug":           "调试中心",
	"keypair":         "公钥私钥",
	"async":           "异步任务",
	"comp-buttons":    "按钮组件",
	"comp-inputs":     "输入框组件",
	"comp-table":      "表格组件",
	"comp-card":       "卡片组件",
	"comp-tabs":       "标签页组件",
	"fn-chart":        "图表功能",
	"fn-form":         "表单功能",
	"fn-dialog":       "弹窗功能",
	"task":            "Task演示",
	"virtuallist":     "虚拟列表演示",
	"grand":           "GRAND_DEMO",
	"async-dev":       "异步开发",
	"settings":        "设置",
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	// 紫色强调色（截图签名色），覆盖系统默认。
	t.Accent = ui.Hex("#7c5cff")
	t.AccentHover = ui.Hex("#8f73ff")
	t.AccentPressed = ui.Hex("#6a4ce0")
	t.AccentText = ui.Hex("#ffffff")

	ui.Column(c).Fill().Children(func() {
		a.titleBar(c, t)
		ui.Row(c).Fill().Children(func() {
			a.sidebar(c, t)
			ui.Scroll(c).Fill().Padding(24, 28).Children(func() {
				switch a.page {
				case "home":
					a.homePage(c, t)
				default:
					placeholderPage(c, t, pageTitles[a.page], a.page)
				}
			})
		})
	})
	a.dialog(c, t)
}

// ---- 自定义标题栏（去系统边框，自行绘制） ----

func (a *app) titleBar(c *ui.Context, t *ui.Theme) {
	bar := c.TitleBar()
	ui.Row(c).
		Height(max(bar.Height, float32(44))).
		Padding(0, bar.Right, 0, bar.Left). // 让出 macOS 左侧交通灯 / Windows 右侧窗口控制
		Background(t.Surface).
		Border(0, t.Border).
		AlignItems(ui.Center). // 垂直居中：与系统最小化/最大化/关闭按钮对齐，消除上下留白
		Gap(4).
		DragWindow(). // 整条标题栏可拖动窗口
		Children(func() {
			a.appLogo(c, t)
			ui.Text(c, "量子科学").FontSize(15).Bold().Padding(10, 0, 0, 0)
			ui.Spacer(c).Grow(1)
			titleIconBtn(c, t, iconGlobe)
			titleIconBtn(c, t, iconBell)
		})
	// 标题栏下分隔线
	ui.Row(c).Height(1).Background(t.Border)
}

// appLogo 左上角标志：紫色圆角方块 + 白色原子（量子）图标，撑满方块、垂直居中。
func (a *app) appLogo(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Width(30).Height(30).Radius(8).Background(t.Accent).
		AlignItems(ui.Center).Justify(ui.Center).
		Children(func() {
			ui.Image(c, iconAtom).Size(19, 19).TextColor(ui.Hex("#ffffff"))
		})
}

// titleIconBtn 标题栏右侧的方形图标按钮（网络 / 通知），与系统按钮等高对齐、hover 有反馈。
func titleIconBtn(c *ui.Context, t *ui.Theme, svg *ui.SVG) {
	ui.Button(c, "").
		Width(34).Height(34).Radius(8).Padding(0).
		Background(t.Surface).TextColor(t.TextMuted).
		AlignItems(ui.Center).Justify(ui.Center).
		Children(func() {
			ui.Image(c, svg).Size(18, 18).TextColor(t.TextMuted)
		})
}

// ---- 左侧导航 ----

func (a *app) sidebar(c *ui.Context, t *ui.Theme) {
	ui.Sidebar(c, &a.page, func() {
		ui.SidebarItem(c, "home", iconHome, "首页")
		ui.SidebarSection(c, "图标", &a.openIcons, func() {
			ui.SidebarItem(c, "icons-font", iconFolder, "字体图标")
			ui.SidebarItem(c, "icons-gallery", iconFolder, "图标清单")
		})
		ui.SidebarItem(c, "gen-license", iconKey, "生成license")
		ui.SidebarItem(c, "gen-pwd", iconKey, "生成用户密码")
		ui.SidebarSection(c, "个人资料", &a.openProfile, func() {
			ui.SidebarItem(c, "profile-basic", iconGear, "基本信息")
			ui.SidebarItem(c, "profile-security", iconKey, "安全设置")
		})
		ui.SidebarItem(c, "debug", iconBug, "调试中心")
		ui.SidebarItem(c, "keypair", iconKey, "公钥私钥")
		ui.SidebarItem(c, "async", iconBolt, "异步任务")
		ui.SidebarSection(c, "组件演示", &a.openComponents, func() {
			ui.SidebarItem(c, "comp-buttons", iconBox, "按钮组件")
			ui.SidebarItem(c, "comp-inputs", iconFolder, "输入框组件")
			ui.SidebarItem(c, "comp-table", iconList, "表格组件")
			ui.SidebarItem(c, "comp-card", iconBox, "卡片组件")
			ui.SidebarItem(c, "comp-tabs", iconList, "标签页组件")
		})
		ui.SidebarSection(c, "功能演示", &a.openFunctions, func() {
			ui.SidebarItem(c, "fn-chart", iconStar, "图表功能")
			ui.SidebarItem(c, "fn-form", iconFolder, "表单功能")
			ui.SidebarItem(c, "fn-dialog", iconBox, "弹窗功能")
		})
		ui.SidebarItem(c, "task", iconBox, "Task演示")
		ui.SidebarItem(c, "virtuallist", iconList, "虚拟列表演示")
		ui.SidebarItem(c, "grand", iconStar, "GRAND_DEMO")
		ui.SidebarItem(c, "async-dev", iconCpu, "异步开发")
		ui.SidebarItem(c, "settings", iconGear, "设置")
	}).Width(220)
}

// ---- 首页（对应截图主体） ----

func (a *app) homePage(c *ui.Context, t *ui.Theme) {
	// 页头
	ui.Text(c, "首页").FontSize(24).Bold().Padding(0, 0, 16)

	// 项目概览 区块标题行
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "项目概览").FontSize(20).Bold().Grow(1)
		ui.Text(c, a.welcome).TextColor(t.TextMuted)
		// 紫框按钮（outline）
		if ui.Button(c, "点击交互").
			Background(t.Background).
			Border(1, t.Accent).
			TextColor(t.Accent).
			Clicked() {
			a.showDlg = true
		}
	})

	ui.Text(c, "概览卡片").FontSize(15).Bold().Padding(20, 0, 10)

	// 三张概览卡
	ui.Row(c).Gap(16).Children(func() {
		overviewCard(c, t, iconPackage, "总销售额", "¥ 12,345")
		overviewCard(c, t, iconBell, "新订单", "120 笔")
		overviewCard(c, t, iconStar, "活跃用户", "5000+")
	})

	ui.Text(c, "最近产品").FontSize(15).Bold().Padding(24, 0, 10)
	a.productTable(c, t)
}

func overviewCard(c *ui.Context, t *ui.Theme, svg *ui.SVG, label, value string) {
	ui.Column(c).Gap(8).Padding(18).Width(220).Radius(10).
		Background(t.Surface).Border(1, t.Border).
		Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.18)).Children(func() {
		ui.Image(c, svg).Size(26, 26).TextColor(t.Accent)
		ui.Text(c, label).FontSize(13).TextColor(t.TextMuted)
		ui.Text(c, value).FontSize(26).Bold()
	})
}

func (a *app) productTable(c *ui.Context, t *ui.Theme) {
	cols := []ui.TableColumn{
		{Title: "名称", Sortable: true},
		{Title: "分类", Sortable: true},
		{Title: "价格", Width: 120, Align: ui.End, Sortable: true},
	}
	a.plist.Key = func(i int) any { return a.products[i].name }
	a.plist.Label = func(i int) string { return a.products[i].name }
	a.plist.Selected = &a.pSel
	a.plist.Sort = &a.pSort
	ui.Table(c, &a.plist, cols, len(a.products), func(row, col int) {
		p := a.products[row]
		switch col {
		case 0:
			ui.Text(c, p.name).SingleLine()
		case 1:
			ui.Text(c, p.category).SingleLine()
		case 2:
			ui.Text(c, p.price).SingleLine()
		}
	}).Grow(1)
}

// ---- 占位页（其余菜单项） ----

func placeholderPage(c *ui.Context, t *ui.Theme, title, id string) {
	if title == "" {
		title = id
	}
	ui.Text(c, title).FontSize(24).Bold().Padding(0, 0, 12)
	ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(0, 0, 16).Children(func() {
		ui.Column(c).Width(8).Height(8).Radius(999).Background(t.Accent)
		ui.Text(c, "（演示占位页面）").TextColor(t.TextMuted)
	})
	ui.Text(c, "该页面为整体框架占位，后续可接入对应功能模块。").TextColor(t.TextMuted)
}

// ---- 弹窗（点击交互后弹出，对应 JavaFX 的 FXCustomDialog） ----

func (a *app) dialog(c *ui.Context, t *ui.Theme) {
	ui.Modal(c, &a.showDlg, func() {
		ui.Text(c, "系统通知").FontSize(18).Bold()
		ui.Text(c, "您保存的配置已成功应用到本地 config.json 文件。").
			TextColor(t.TextMuted).Padding(0, 12, 12)
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.PrimaryButton(c, "我知道了").Clicked() {
				a.showDlg = false
			}
		})
	})
}

func main() {
	a := newApp()
	// 截图整体为深色主题，故强制深色。
	mygo.Theme.SetSource(mygo.ThemeDark)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:          "量子科学 (mygo)",
			Width:          1100,
			Height:         720,
			MinWidth:       820,
			MinHeight:      520,
			TitleBarStyle:  mygo.TitleBarHidden, // 自绘标题栏
			TitleBarHeight: 44,
			BackgroundColor: "#15131f",
			StateKey:       "bestuav-mygo",
			Content:        ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
