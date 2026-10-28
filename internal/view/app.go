package view

import (
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
)

// product 产品数据（mock，对应截图“最近产品”表）。
type product struct {
	name     string
	category string
	price    string
}

// App 持有窗口状态；View 在主线程按帧从它构建界面。
type App struct {
	page string // 当前选中页 id（由侧栏写入）

	// 侧栏分组开合状态
	openTools, openProfile, openComponents, openFunctions bool

	// 侧栏整体展开/收缩（false=展开 220px，true=收缩 64px 图标栏）。持久化。
	sidebarCollapsed bool

	// 首页
	welcome  string
	products []product
	showDlg  bool

	// 最近产品表状态
	plist ui.ListState
	pSel  int
	pSort ui.SortOrder

	// 用户偏好（持久化到 per-user 配置目录的 JSON 文件）
	settings *settings.Settings

	// UI 绑定值与其“已应用”快照（用于检测变化并触发应用/持久化）。
	// 主题：0 跟随系统 / 1 浅色 / 2 深色
	themeIdx        int
	appliedThemeIdx int
	// 关闭行为：0 每次询问 / 1 最小化到托盘 / 2 直接退出
	closeModeIdx        int
	appliedCloseModeIdx int
	// 字体大小（Slider 绑定值，px）
	fontSize      float64
	appliedFontPx float64

	// 首次关闭询问弹窗状态
	showCloseChoice bool
	closeChoice     string // 弹窗内临时选择："minimize" / "quit"
	rememberClose   bool   // 弹窗内“记住我的选择”

	// 由 main 注入的关闭执行器：view 包不持有 *mygo.Window，
	// 故隐藏/退出这类需要窗口对象的行为通过回调交给 main 执行。
	closeHandler func(mode string)
}

// NewApp 构造窗口初始状态：加载已保存偏好，并按其应用主题。
func NewApp() *App {
	s := settings.MustLoad()
	a := &App{
		page:             "home",
		openTools:        true,
		openProfile:      true,
		openComponents:   true,
		openFunctions:    true,
		sidebarCollapsed: s.SidebarCollapsed,
		welcome:          "欢迎来到AtlantaFX 世界！",
		products: []product{
			{"笔记本电脑", "电子产品", "¥ 8999"},
			{"机械键盘", "外设", "¥ 699"},
			{"无线鼠标", "外设", "¥ 299"},
			{"显示器", "电子产品", "¥ 2499"},
		},
		settings: s,
		fontSize: s.FontSize,
	}
	a.themeIdx = themeIdxOf(s.Theme)
	a.appliedThemeIdx = a.themeIdx
	a.closeModeIdx = closeModeIdxOf(s.CloseMode)
	a.appliedCloseModeIdx = a.closeModeIdx
	a.appliedFontPx = math.Round(s.FontSize)
	// 注意：此处只初始化索引，不调用 SetSource。主题 source 的切换必须在
	// 应用主循环就绪后（mygo.App.WhenReady 内）进行——SetSource 内部走
	// onMain，未启动 App 时（如单测环境）会永久阻塞。
	return a
}

// ApplyTheme 在应用主循环就绪后（mygo.App.WhenReady 内）应用已保存的主题
// source。请勿在 NewApp 或 App.Run 之前调用。
func (a *App) ApplyTheme() { a.applyThemeSource() }

// ---- 偏好 <-> UI 索引 转换 ----

func themeIdxOf(s string) int {
	switch s {
	case "light":
		return 1
	case "dark":
		return 2
	default:
		return 0
	}
}

func themeName(i int) string {
	switch i {
	case 1:
		return "light"
	case 2:
		return "dark"
	default:
		return "system"
	}
}

func closeModeIdxOf(s string) int {
	switch s {
	case "minimize":
		return 1
	case "quit":
		return 2
	default:
		return 0
	}
}

func closeModeName(i int) string {
	switch i {
	case 1:
		return "minimize"
	case 2:
		return "quit"
	default:
		return "ask"
	}
}

// applyThemeSource 把当前 themeIdx 映射到 mygo 主题源并应用。
func (a *App) applyThemeSource() {
	switch a.themeIdx {
	case 1:
		mygo.Theme.SetSource(mygo.ThemeLight)
	case 2:
		mygo.Theme.SetSource(mygo.ThemeDark)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}

// persist 保存偏好；写盘失败仅告警，不阻断界面（桌面偏好属于可丢失的非关键数据）。
func (a *App) persist() {
	if err := a.settings.Save(); err != nil {
		println("warn: 保存设置失败:", err.Error())
	}
}

// toggleSidebar 切换侧栏展开/收缩，并持久化状态，使下次启动保持现状。
func (a *App) toggleSidebar() {
	a.sidebarCollapsed = !a.sidebarCollapsed
	a.settings.SidebarCollapsed = a.sidebarCollapsed
	a.persist()
}

// SetCloseHandler 由 main 注入关闭执行器（隐藏或退出窗口）。
func (a *App) SetCloseHandler(fn func(mode string)) { a.closeHandler = fn }

// runClose 通过注入的回调执行关闭行为。
func (a *App) runClose(mode string) {
	if a.closeHandler != nil {
		a.closeHandler(mode)
	}
}

// HandleClose 是窗口关闭按钮的回调（由 main 的 win.OnClose 转发）。
//   - CloseMode=="minimize"：最小化到托盘
//   - CloseMode=="quit"：直接退出
//   - CloseMode=="ask"（默认）：弹出首次关闭询问，交由用户本次决定
func (a *App) HandleClose(e *mygo.CloseEvent, quitting *bool) {
	if *quitting {
		return // 系统退出流程进行中，放行
	}
	switch a.settings.CloseMode {
	case "minimize":
		e.PreventDefault()
		a.runClose("minimize")
	case "quit":
		e.PreventDefault()
		a.runClose("quit")
	default: // "ask"
		e.PreventDefault()
		a.closeChoice = "minimize"
		a.rememberClose = false
		a.showCloseChoice = true
	}
}

// pageTitles 各页面的标题（用于右侧页头与占位页）。
var pageTitles = map[string]string{
	"home":             "首页",
	"gen-license":      "生成license",
	"gen-pwd":          "生成用户密码",
	"profile-basic":    "基本信息",
	"profile-security": "安全设置",
	"debug":            "调试中心",
	"keypair":          "公钥私钥",
	"async":            "异步任务",
	"comp-buttons":     "按钮组件",
	"comp-inputs":      "输入框组件",
	"comp-table":       "表格组件",
	"comp-card":        "卡片组件",
	"comp-tabs":        "标签页组件",
	"fn-chart":         "图表功能",
	"fn-form":          "表单功能",
	"fn-dialog":        "弹窗功能",
	"task":             "Task演示",
	"virtuallist":      "虚拟列表演示",
	"grand":            "GRAND_DEMO",
	"async-dev":        "异步开发",
	"settings":         "设置",
}

// View 由 mygo 每帧调用，按当前状态构建整个界面。
func (a *App) View(c *ui.Context) {
	t := c.Theme()

	// 1) 主题源：仅在变化时切换（SetSource 会触发整树重建，不可每帧调用）。
	if a.themeIdx != a.appliedThemeIdx {
		a.applyThemeSource()
		a.appliedThemeIdx = a.themeIdx
		a.settings.Theme = themeName(a.themeIdx)
		a.persist()
	}
	// 2) 关闭行为：变化时持久化（关闭时实时读取 settings.CloseMode）。
	if a.closeModeIdx != a.appliedCloseModeIdx {
		a.settings.CloseMode = closeModeName(a.closeModeIdx)
		a.appliedCloseModeIdx = a.closeModeIdx
		a.persist()
	}
	// 3) 字体大小：整数 px 变化时更新并持久化（避免拖动时频繁写盘）。
	rounded := math.Round(a.fontSize)
	if rounded != a.appliedFontPx {
		a.settings.FontSize = rounded
		a.appliedFontPx = rounded
		a.persist()
	}
	// 4) 每帧应用字号（绝对赋值，幂等；确保各页面一致缩放）。
	if a.settings.FontSize > 0 {
		t.FontSize = float32(a.settings.FontSize)
	}
	// 5) 隐藏滚动条但保留滚动（滚轮/触控板）：将滚动条着色为完全透明。
	//    每次重建主题后重设，确保始终生效。
	t.Scrollbar = t.Scrollbar.Alpha(0)

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Fill().Children(func() {
			a.sidebar(c, t)
			ui.Scroll(c).Fill().Padding(24, 28).Children(func() {
				switch a.page {
				case "home":
					a.homePage(c, t)
				case "settings":
					a.settingsPage(c, t)
				default:
					placeholderPage(c, t, pageTitles[a.page], a.page)
				}
			})
		})
	})
	a.dialog(c, t)
	a.closeChoiceDialog(c, t)
}
