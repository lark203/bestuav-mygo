package view

import (
	"math"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"bestuav-mygo/internal/appmeta"
	"bestuav-mygo/internal/logging"
	"bestuav-mygo/internal/settings"
	"bestuav-mygo/internal/store"
	"bestuav-mygo/internal/ui/icons"
	"bestuav-mygo/internal/ui/kit"
	"bestuav-mygo/internal/view/blackhole"
	"bestuav-mygo/internal/view/fx"
	"bestuav-mygo/internal/view/piano"
	"bestuav-mygo/internal/view/shell"
)

// App 持有窗口状态；View 在主线程按帧从它构建界面。
type App struct {
	page string // 当前选中页 id（由侧栏写入）

	// 侧栏分组的折叠状态（key 为分组标题；缺省视为展开）。
	// 用 map 而不是四个 bool 字段：新增分组时不必再改 App 结构体。
	groupFolded map[string]bool

	// 侧栏整体展开/收缩（false=展开 220px，true=收缩 64px 图标栏）。持久化。
	sidebarCollapsed bool

	// ---- 页面生命周期（见 pagelife.go）----
	pages    map[string]*pageState // 仍在缓存中的页面运行时状态
	pagePrev string                // 上一帧的页面 id，用于检测切换

	// 首页
	welcome string
	showDlg bool

	// 基本信息弹窗开关：由设置页「关于」区的按钮或首页快捷入口打开。
	// 内容见 basicinfo.go（应用标识 / 运行环境 / 页面缓存 / 数据库与日志路径）。
	showBasicInfo bool
	basicTab      int // 弹窗内当前页签（0 应用 / 1 环境 / 2 缓存 / 3 数据与日志 / 4 配置）

	// 最近产品表状态（首页「近期任务」小表）
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

	// 每帧算出的浮层不透明度（0..1）。没有背景图时都是 1（完全不透明），
	// 由 applyBackdropTheme 统一赋值，供 menuSurface / windowSurface 使用。
	menuPanelAlpha    float64
	contentPanelAlpha float64

	// 页面缓存闲置销毁时长（分钟，0=不销毁）。Stepper 绑定 float64，故单列一份。
	pageIdle        float64
	appliedPageIdle float64

	// 多实例开关（保存后需重启生效：单实例锁在进程启动时获取）。
	allowMultiInstance bool
	appliedMulti       bool

	// ---- 轻量级任务进度 ----
	// taskRunning 为 true 时，页头下方那条分割线的槽位改画进度条。
	taskRunning  bool
	taskProgress float64 // 0..1；负值表示“不确定进度”（无限循环动画）
	taskLabel    string  // 当前任务名（显示在页头标题右侧）

	// ---- 重量级任务（全局遮罩）----
	// heavyRunning 为 true 时，整个窗口罩上一层不可关闭的遮罩并拦截所有操作。
	heavyRunning  bool
	heavyProgress float64
	heavyLabel    string

	// ---- 任务列表（异步任务页）----
	tasks      []taskRow
	taskState  ui.ListState
	taskSel    int
	taskSerial int

	// ---- 彩带庆祝（任务完成 / 手动触发）----
	bursts      []*fx.Burst
	nextBurstID uint64

	// ---- 轻提示（toast）----
	toastMsg   string
	toastUntil time.Time

	// ---- 基本信息页的运行时快照 ----
	env   envSnapshot
	envAt time.Time

	// ---- 生成 license 页的表单与结果 ----
	lic licForm

	// ---- 组件演示页的控件状态 ----
	demoToggle, demoToggle2 bool
	demoRating              int
	demoText, demoNote      string
	demoDateT               time.Time
	demoSelect              string
	demoCheck, demoSwitch   bool
	demoSlider, demoStep    float64
	demoTable               ui.ListState
	demoSel                 int
	demoTab, demoSeg        int
	demoCollapse            bool

	// ---- 功能演示状态 ----
	chartSeries  [3][]float32 // 滚动图表的三个序列
	chartTick    int
	chartPaused  bool
	todos        []todoItem
	todoInput    string
	todoSeeded   bool
	vlistState   ui.ListState
	vlistSel     int
	timelinePlay bool

	// 由 main 注入的窗口重绘触发器：后台任务更新进度后需主动请求重绘，
	// 否则界面要等下一次交互才刷新。
	invalidate func()

	// 由 main 注入的关闭执行器：隐藏/退出这类需要「应用级」语义的行为通过回调
	// 交给 main 执行（那里才知道 quitting 标志与托盘状态）。
	closeHandler func(mode string)

	// 由 main 注入的主窗口句柄（WhenReady 里创建窗口后注入）。
	// 只有「开子窗口」这类必须拿父窗口的能力才用它（目前仅浮出面板示例）：
	// 离屏渲染与单测环境下为 nil，相关入口自动降级（不弹出子窗口）。
	win *mygo.Window

	// ---- 本地持久化（见 persist.go）----
	// store 可为 nil：数据库打开失败或单测环境时，全部读写退化为纯内存。
	store     *store.Store
	dbStats   store.Stats // 库计数快照（带 TTL，避免每帧 COUNT）
	dbStatsAt time.Time

	// ---- 官方示例页的运行时状态 ----
	// （页面实现见 page_loading.go / page_markdown_chat.go / page_notes.go）
	loadingSize   int  // 指示器尺寸档位（对应 loadingSizes）
	loadingColor  int  // 颜色档位：正文 / 强调 / 次要
	loadingEasing int  // 缓动档位
	loadingCap    int  // 端点档位
	loadingPaused bool // 暂停动画

	chatMsgs    []*chatMessage // 会话消息
	chatInput   string         // 输入框内容
	chatList    ui.ListState   // 列表状态（跟随末尾）
	chatMD      markdown.Options
	chatNext    int // 消息 id 递增
	chatReplies int // 已用回复数（轮换回复文案）
	chatReveal  int // 流式单位档位

	// 黑洞演示的场景（着色器、相机、两块画布）。离开页面即置 nil，
	// 释放 GPU 画布；下次进入重新创建（着色器是包级编译一次，代价很小）。
	bh *blackhole.Scene

	// 代码编辑器示例的状态（五个示例文件的缓冲区与 diff）。页面缓存期间保留，
	// 以便「切回来还在继续编辑」；整页卸载时随 App 一起没。
	ce *codeEditor

	// 自绘输入框示例的状态（文本、选区、输入法组字）。同样随页面缓存保留。
	tf *textField

	// 钢琴示例的状态（按下的键、踏板、八度、正在放的曲子）。
	// 离开页面要 Stop + 制音，否则后台 goroutine 还在按时间表弹键。
	pi      *piano.Piano
	piMuted bool // 静音开关（Switch 绑在 App 上，再同步给 Piano）

	// 大文档演示的状态（10 万行的索引缓冲 + 撤销栈）。随页面缓存保留，
	// 整页销毁时随 App 一起没 —— 4 MB 的文档不该长期挂在内存里。
	tb *tbDemo

	// 浮出面板示例的状态（颜色选择、便签、关于），含三个子窗口句柄。
	// 离开页面会先关掉已弹出的子窗口（见 closeFlyouts），避免孤儿窗口。
	fo *flyoutDemo

	notes        []*noteItem // 笔记列表
	noteSel      string      // 选中的笔记 id
	noteNext     int         // id 递增
	noteLinkOpen bool        // 链接弹层
	noteLinkURL  string
	notePath     string // notes.json 路径（空 = 不落盘）
	noteLoaded   bool   // 是否已尝试加载
}

// NewApp 构造窗口初始状态：加载已保存偏好，并按其应用主题。
func NewApp() *App { return newApp(settings.MustLoad()) }

// newApp 用给定偏好构造 App，不碰磁盘。
//
// 单独拆出来是为了让测试能用 settings.Default() 构造确定的初始状态：直接
// NewApp() 会读取开发机上的真实 settings.json，测试结果会随「我本机是否收起
// 过侧栏 / 是否设过背景图」而变化，属于典型的测试不确定源。
func newApp(s *settings.Settings) *App {
	a := &App{
		page:             "home",
		groupFolded:      map[string]bool{},
		pages:            map[string]*pageState{},
		sidebarCollapsed: s.SidebarCollapsed,
		welcome:          "欢迎回来",
		settings:         s,
		fontSize:         s.FontSize,
		// 背景与浮层透明度
		menuOpacity:    s.MenuOpacity,
		contentOpacity: s.ContentOpacity,
		// 多实例开关
		allowMultiInstance: s.AllowMultiInstance,
		// 页面缓存闲置销毁
		pageIdle:        float64(s.PageIdleMinutes),
		appliedPageIdle: float64(s.PageIdleMinutes),
		// 任务列表初始演示数据
		tasks: demoTaskRows(),
		// 表单初始值
		lic: defaultLicForm(),
		// 组件演示页的默认值（选中项用 -1 表示「未选中」）
		demoSelect: "专业版",
		demoRating: 3,
		demoSlider: 42,
		demoStep:   4,
		demoSel:    -1,
		vlistSel:   -1,
		taskSel:    -1,
		// 加载指示器默认 24px（loadingSizes 第 2 档）
		loadingSize: 1,
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

// SetCloseHandler 由 main 注入关闭执行器（隐藏或退出窗口）。
func (a *App) SetCloseHandler(fn func(mode string)) { a.closeHandler = fn }

// SetInvalidator 由 main 注入窗口重绘触发器（win.Invalidate）。后台任务
// 更新进度后调用它请求重绘。
func (a *App) SetInvalidator(fn func()) { a.invalidate = fn }

// SetWindow 由 main 注入主窗口句柄。浮出面板等子窗口需要它作为 Parent；
// 未注入（离屏渲染 / 单测）时为 nil，相关入口自动降级。
func (a *App) SetWindow(w *mygo.Window) { a.win = w }

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
	return kit.Alpha(t.Background, a.contentPanelAlpha)
}

// menuSurface 侧栏浮层的填充色，语义同 windowSurface（用菜单区不透明度）。
func (a *App) menuSurface(t *ui.Theme) ui.Color {
	return kit.Alpha(t.Surface, a.menuPanelAlpha)
}

// applyBackdropTheme 设置背景图时，把主题里的「控件面」相关颜色整体降透明度，
// 让卡片、按钮、输入框、表格与各种悬停底色不再像白板一样挡住背景图。
//
// 为什么这样做：mygo 的主题是整帧共享的（c.Theme() 返回同一份 *Theme），
// 因此无法只对某个子树生效；反过来，这也正好让我们「一处调整、全局一致」。
// 各组件（ui.Button / ui.Table / ui.Segmented / ui.Slider / 我们自己的卡片）
// 都从这些字段取色，所以无需逐个改组件。
//
// 不参与调整的：Text / Border / Accent 等语义色——文字、描边、强调色与选中态
// 需要保持对比度，半透明化会直接伤害可读性。
func (a *App) applyBackdropTheme(t *ui.Theme) {
	// 默认（无背景图）：全部不透明。
	a.menuPanelAlpha, a.contentPanelAlpha = 1, 1
	surface, hover, pressed := 1.0, 1.0, 1.0

	if a.bgBitmap != nil {
		a.menuPanelAlpha = kit.Clamp01(a.menuOpacity)
		a.contentPanelAlpha = kit.Clamp01(a.contentOpacity)

		// 控件面（卡片/按钮/输入框/分段控件底）：比它所在的浮层「再实一点」。
		// 若与浮层同透明度，卡片会与背景浮层融为一体，层次只剩 1px 描边。
		// 取「向不透明方向补一半剩余差值」：内容区 54% → 卡片 77%。
		surface = kit.Clamp01(a.contentPanelAlpha + (1-a.contentPanelAlpha)*0.5)
		// 悬停底比控件面「虚一点」但不弱于浮层：既能在照片上看出悬停，
		// 又让「选中的分段/页签」（用 Surface 画）比它更实，选中态才读得出来。
		hover = kit.Clamp01(a.contentPanelAlpha + (surface-a.contentPanelAlpha)*0.35)
		// 按下比控件面更实，给足点击反馈。
		pressed = kit.Clamp01(surface + (1-surface)*0.5)
	}

	t.Surface = kit.Alpha(t.Surface, surface)
	t.SurfaceHover = kit.Alpha(t.SurfaceHover, hover)
	t.SurfacePressed = kit.Alpha(t.SurfacePressed, pressed)
	t.Background = kit.Alpha(t.Background, a.contentPanelAlpha)
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
		// 后台协程里的 panic 不会经过 View 的 recover，单独兜一层。
		defer logging.Recover("light-task")
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

// endTask 结束轻量任务：分割线恢复常态，并撒彩带庆祝（用户要求的「完成时烟花/彩带秀」）。
func (a *App) endTask() {
	a.taskRunning = false
	a.taskProgress = 0
	a.taskLabel = ""
	a.celebrate(3)
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

// View 由 mygo 每帧调用，按当前状态构建整个界面。
func (a *App) View(c *ui.Context) {
	// 渲染期 panic 会直接崩掉整个进程：先落一条 CRASH 日志（含堆栈）并入库，
	// 再按原样抛出，保持既有崩溃行为不变（见 logging.Recover 的说明）。
	defer logging.Recover("view")

	t := c.Theme()

	// 0) 页面生命周期：处理「进入 / 离开 / 闲置销毁」，必须在构建界面前完成，
	//    这样 keepFresh 页面在本帧就是全新的状态（见 pagelife.go）。
	a.syncPages()

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
	// 5) 滚动条：保留主题默认的可见性（内容区需要滚动条）；侧栏的滚动条在
	//    shell.Sidebar 内部单独隐藏（见 internal/view/shell/layout.go）。
	// 6) 浮层不透明度：保留两位小数比较，避免拖动滑块时每个像素都写盘。
	mo := math.Round(a.menuOpacity*100) / 100
	co := math.Round(a.contentOpacity*100) / 100
	if mo != a.appliedMenuOp || co != a.appliedContentOp {
		a.settings.MenuOpacity = mo
		a.settings.ContentOpacity = co
		a.appliedMenuOp, a.appliedContentOp = mo, co
		a.persist()
	}
	// 6.5) 页面缓存闲置销毁时长（分钟，0=不销毁）：整数变化才写盘。
	pi := math.Round(a.pageIdle)
	if pi != a.appliedPageIdle {
		a.settings.PageIdleMinutes = int(pi)
		a.appliedPageIdle = pi
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
	// 9) 有背景图时把「控件面」整体降透明度（卡片/按钮/输入框不再挡住背景），
	//    同时算出侧栏与内容区两层浮层的透明度；无背景图时全部为不透明。
	a.applyBackdropTheme(t)

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
	a.basicInfoDialog(c, t)
	// 重量级任务遮罩：盖在最上层并拦截一切操作（用户要求「完成前禁止操作」）。
	a.heavyTaskMask(c, t)
	// 彩带庆祝：任务完成或手动触发时撒一把，飘完自动清空。
	a.drawCelebration(c)
	// 轻提示（已复制 / 已重置等）。
	a.drawToast(c, t)
}

// shell 应用主体骨架：左侧导航栏 + 右侧内容区。
//
// 两块都由 internal/view/shell 绘制，这里只做「装配」——把当前页、折叠态、
// 页面内容与工具条以指针/回调的形式交给骨架：
//   - 改骨架（间距、页头、分割线、滚动行为）不牵扯任何业务页面；
//   - 新增业务页面也不需要碰骨架，只在 registry.go 登记即可。
func (a *App) shell(c *ui.Context, t *ui.Theme) {
	shell.Sidebar(c, t,
		&shell.NavState{
			Page:      &a.page,
			Collapsed: &a.sidebarCollapsed,
			Folded:    a.groupFolded,
		},
		shell.SidebarOptions{
			Title:     appmeta.Title,
			BrandIcon: icons.Atom,
			Groups:    navGroups(),
			Footer: shell.Item{
				ID:    settingsNav.id,
				Label: settingsNav.label,
				Icon:  settingsNav.icon,
			},
			Background: a.menuSurface(t),
			// 折叠状态变化后持久化，使下次启动保持现状。
			OnCollapseChange: func(collapsed bool) {
				a.settings.SidebarCollapsed = collapsed
				a.persist()
			},
		})

	// 轻量任务执行时，标题右侧显示任务名（与分割线槽位里的进度条呼应）。
	subtitle := ""
	if a.taskRunning && a.taskLabel != "" {
		subtitle = a.taskLabel
	}
	shell.Content(c, t, shell.ContentOptions{
		Title:      pageTitle(a.page),
		Subtitle:   subtitle,
		Tools:      func() { a.headerTools(c, t) },
		Body:       func() { a.pageBody(c, t) },
		Scroll:     a.ScrollFor(a.page),
		Running:    a.taskRunning,
		Progress:   a.taskProgress,
		Background: a.windowSurface(t),
	})
}
