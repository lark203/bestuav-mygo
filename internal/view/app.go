package view

import (
	"math"
	"os"
	"time"

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

	// ---- 背景图与浮层透明度 ----
	// 背景图只在路径变化时读盘并解码一次（bgLoaded 记录已加载的路径），
	// 之后每帧直接复用 *ui.Bitmap，避免逐帧解码导致卡顿。
	bgBitmap *ui.Bitmap
	bgLoaded string // 已加载到 bgBitmap 的路径（"" = 未加载）
	bgError  string // 最近一次加载失败的原因（设置页展示）

	// 浮层不透明度（0.2–1.0）：设置背景图后侧栏/内容区半透明，背景图透出。
	// 未设置背景图时恒为不透明，保持主题默认外观。
	menuOpacity      float64
	contentOpacity   float64
	appliedMenuOp    float64
	appliedContentOp float64

	// 多实例开关（保存后需重启生效：单实例锁在进程启动时获取）。
	allowMultiInstance bool
	appliedMulti       bool

	// ---- 轻量级任务进度 ----
	// taskRunning 为 true 时，页头下方那条分割线的槽位改画进度条。
	taskRunning  bool
	taskProgress float64 // 0..1；负值表示“不确定进度”（无限循环动画）
	taskLabel    string  // 当前任务名（显示在页头标题右侧）

	// 由 main 注入的窗口重绘触发器：后台任务更新进度后需主动请求重绘，
	// 否则界面要等下一次交互才刷新。
	invalidate func()

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
		// 背景与浮层透明度
		menuOpacity:    s.MenuOpacity,
		contentOpacity: s.ContentOpacity,
		// 多实例开关
		allowMultiInstance: s.AllowMultiInstance,
	}
	a.themeIdx = themeIdxOf(s.Theme)
	a.appliedThemeIdx = a.themeIdx
	a.closeModeIdx = closeModeIdxOf(s.CloseMode)
	a.appliedCloseModeIdx = a.closeModeIdx
	a.appliedFontPx = math.Round(s.FontSize)
	a.appliedMenuOp = s.MenuOpacity
	a.appliedContentOp = s.ContentOpacity
	a.appliedMulti = s.AllowMultiInstance
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

// SetInvalidator 由 main 注入窗口重绘触发器（win.Invalidate）。后台任务
// 更新进度后调用它请求重绘。
func (a *App) SetInvalidator(fn func()) { a.invalidate = fn }

// AllowMultiInstance 报告用户是否允许同时运行多个实例。main 在启动时读取它
// 决定是否获取单实例锁（设置改动需重启生效）。
func (a *App) AllowMultiInstance() bool { return a.settings.AllowMultiInstance }

// requestFrame 请求一次重绘；未注入（如单测环境）时静默忽略。
func (a *App) requestFrame() {
	if a.invalidate != nil {
		a.invalidate()
	}
}

// ---- 背景图 ----

// ensureBackground 保证 bgBitmap 与设置中的背景图路径一致：路径变化才读盘解码。
func (a *App) ensureBackground() {
	want := a.settings.BackgroundImage
	if want == a.bgLoaded {
		return // 未变化：复用已解码位图
	}
	a.bgLoaded = want
	a.bgError = ""
	if want == "" {
		a.bgBitmap = nil
		return
	}
	data, err := os.ReadFile(want)
	if err != nil {
		a.bgBitmap = nil
		a.bgError = "读取失败：" + err.Error()
		return
	}
	bmp, err := ui.DecodeBitmap(data)
	if err != nil {
		a.bgBitmap = nil
		a.bgError = "解码失败：" + err.Error()
		return
	}
	a.bgBitmap = bmp
}

// chooseBackground 弹出系统原生的文件选择框挑选背景图。
//
// 对话框会阻塞调用方直到用户关闭，故放到独立 goroutine：不阻塞 UI 主循环；
// 选中后回到主线程更新状态并请求重绘。
func (a *App) chooseBackground() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:       "选择背景图片",
			ButtonLabel: "设为背景",
			Filters: []mygo.FileFilter{{
				Name:       "图片",
				Extensions: []string{"png", "jpg", "jpeg", "webp", "bmp", "gif"},
			}},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		chosen := paths[0]
		mygo.RunOnMain(func() {
			a.settings.BackgroundImage = chosen
			a.ensureBackground() // 立即加载，失败原因可在设置页看到
			a.persist()
			a.requestFrame()
		})
	}()
}

// resetBackground 移除背景图并恢复默认浮层不透明度（设置页的「恢复默认」）。
func (a *App) resetBackground() {
	a.settings.BackgroundImage = ""
	a.settings.MenuOpacity = settings.DefaultMenuOpacity
	a.settings.ContentOpacity = settings.DefaultContentOpacity
	a.menuOpacity = a.settings.MenuOpacity
	a.contentOpacity = a.settings.ContentOpacity
	a.appliedMenuOp = a.menuOpacity
	a.appliedContentOp = a.contentOpacity
	a.ensureBackground() // 释放位图（路径为空）
	a.persist()
}

// windowSurface 内容区浮层的填充色：有背景图时按「内容区不透明度」半透明，
// 让背景图透出；没有背景图时用主题默认背景（不透明）。
func (a *App) windowSurface(t *ui.Theme) ui.Color {
	if a.bgBitmap == nil {
		return t.Background
	}
	return t.Background.Alpha(float32(math.Round(a.contentOpacity*100) / 100))
}

// menuSurface 侧栏浮层的填充色，语义同 windowSurface（用菜单区不透明度）。
func (a *App) menuSurface(t *ui.Theme) ui.Color {
	if a.bgBitmap == nil {
		return t.Surface
	}
	return t.Surface.Alpha(float32(math.Round(a.menuOpacity*100) / 100))
}

// ---- 轻量级任务进度 ----

// startLightTask 演示一条「轻量级任务」：进度条画在页头下方分割线的槽位里。
// 真实业务只需把 goroutine 里的模拟循环换成实际工作，并在完成时调用 endTask。
func (a *App) startLightTask(label string) {
	if a.taskRunning {
		return // 同一时刻只显示一条轻量任务进度
	}
	a.taskRunning = true
	a.taskProgress = 0
	a.taskLabel = label

	go func() {
		const steps = 24
		for i := 1; i <= steps; i++ {
			time.Sleep(90 * time.Millisecond)
			v := float64(i) / steps
			mygo.RunOnMain(func() {
				a.taskProgress = v
				a.requestFrame()
			})
		}
		mygo.RunOnMain(a.endTask)
	}()
}

// endTask 结束轻量任务，分割线恢复常态。
func (a *App) endTask() {
	a.taskRunning = false
	a.taskProgress = 0
	a.taskLabel = ""
	a.requestFrame()
}

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
	// 6) 浮层不透明度：保留两位小数比较，避免拖动滑块时每个像素都写盘。
	mo := math.Round(a.menuOpacity*100) / 100
	co := math.Round(a.contentOpacity*100) / 100
	if mo != a.appliedMenuOp || co != a.appliedContentOp {
		a.settings.MenuOpacity = mo
		a.settings.ContentOpacity = co
		a.appliedMenuOp, a.appliedContentOp = mo, co
		a.persist()
	}
	// 7) 多实例开关：仅持久化，重启后由 main 在启动时读取生效。
	if a.allowMultiInstance != a.appliedMulti {
		a.settings.AllowMultiInstance = a.allowMultiInstance
		a.appliedMulti = a.allowMultiInstance
		a.persist()
	}
	// 8) 背景图：路径变化时重新读取并解码（只做一次，非每帧）。
	a.ensureBackground()

	if a.bgBitmap != nil {
		// 有背景图时：背景图作为常规流的第一个子元素（因此绘制在最底层），
		// 界面主体改为绝对定位铺满并叠在其上。
		//
		// 为什么不用「背景图 Absolute + 主体常规流」：mygo 的绝对定位元素
		// 是在所有常规流兄弟之后绘制的（见 ui/paint.go 的绘制顺序），会盖住
		// 界面，故反过来做。
		ui.Column(c).Fill().Children(func() {
			ui.Image(c, a.bgBitmap).Fit(ui.Cover).Fill()
			ui.Row(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Children(func() {
				a.shell(c, t)
			})
		})
	} else {
		ui.Row(c).Fill().Children(func() {
			a.shell(c, t)
		})
	}
	a.dialog(c, t)
	a.closeChoiceDialog(c, t)
}

// shell 应用主体骨架：左侧栏 + 右侧内容区（页头 / 分割线 / 可滚动页面）。
func (a *App) shell(c *ui.Context, t *ui.Theme) {
	a.sidebar(c, t)
	a.content(c, t)
}
