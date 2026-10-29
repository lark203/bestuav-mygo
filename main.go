package main

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/appmeta"
	"bestuav-mygo/internal/apppath"
	"bestuav-mygo/internal/logging"
	"bestuav-mygo/internal/store"
	"bestuav-mygo/internal/view"
)

// appIconPNG 应用图标（resources/icon.png，1024×1024 PNG），通过 go:embed 编入
// 二进制。在窗口就绪后通过 (*Window).SetIcon 设置，使窗口标题栏左上角与任务栏
// （Windows）/ Dock（macOS）/ 启动器（Linux）均显示该图标——开发模式（mygo dev）
// 与打包后运行时都生效。
//
// 说明：mygo 在构建安装包（go tool mygo build）时也会自动把 resources/icon.png
// 嵌入可执行文件（Windows .syso / macOS .icns / Linux .desktop），此处为运行时兜底，
// 保证不打包、直接 go run / mygo dev 时图标也可见。
//
//go:embed resources/icon.png
var appIconPNG []byte

func main() {
	// 1) 日志最先初始化：后续任何一步失败（日志目录不可写、数据库打不开…）都能留下
	//    可排查的记录。初始化失败不阻断启动（退化为 stderr）。
	closeLog := setupLogging()
	defer closeLog()

	// 2) 本地数据库：失败同样不阻断——界面退化为纯内存，功能照常可用。
	db := openStore()
	if db != nil {
		defer func() { _ = db.Close() }()
	}

	a := view.NewApp()
	if db != nil {
		a.AttachStore(db)
	}
	// 存储与日志都就绪后再写启动审计，保证这条事件能真正入库。
	announceStartup(db != nil)

	// 供「第二个实例被拉起时唤出主窗口」使用：窗口在 WhenReady 中创建，
	// 创建前为 nil（此时收到唤醒请求就忽略）。
	var mainWin *mygo.Window

	// ---- 单实例控制 ----
	// 默认单实例：重复启动的进程会把命令行参数转发给已在运行的实例，然后由
	// RequestSingleInstanceLock 返回 false，本进程直接退出；首个实例在
	// OnSecondInstance 中把窗口显示出来。
	// 在设置里开启「允许多开」后跳过取锁，允许并行运行多个独立实例。
	if !a.AllowMultiInstance() {
		if !mygo.App.RequestSingleInstanceLock() {
			return
		}
		mygo.App.OnSecondInstance(func(args []string, workingDir string) {
			mygo.RunOnMain(func() {
				if mainWin != nil {
					mainWin.Show()
				}
			})
		})
	}

	mygo.App.WhenReady(func() {
		// 主循环已就绪，应用已保存的主题（跟随系统 / 浅色 / 深色）。
		a.ApplyTheme()
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     appmeta.Title,
			Width:     1100,
			Height:    720,
			MinWidth:  820,
			MinHeight: 520,
			StateKey:  appmeta.Name,
			Content:   ui.View(a.View),
		})
		mainWin = win
		// 注入重绘触发器：轻量任务的进度由后台 goroutine 更新，需要主动请求重绘。
		a.SetInvalidator(win.Invalidate)
		// 原生窗口创建完成后再设置图标，避免平台窗口尚未就绪导致设置失败。
		win.OnReadyToShow(func() {
			if err := win.SetIcon(appIconPNG); err != nil {
				// 图标缺失不应阻断启动：仅打印告警（部分平台窗口未就绪时会返回错误）。
				println("warn: 设置窗口图标失败:", err.Error())
			}
		})

		// 关闭按钮行为：交由 App 按已保存偏好（每次询问 / 最小化到托盘 /
		// 直接退出）处理。首次（CloseMode=="ask"）会弹出选择弹窗。
		var quitting bool
		mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })
		win.OnClose(func(e *mygo.CloseEvent) {
			a.HandleClose(e, &quitting)
		})
		// 关闭执行器：隐藏窗口（最小化到托盘）或真正退出应用。
		// view 包不持有 *mygo.Window，故通过回调在此实现。
		a.SetCloseHandler(func(mode string) {
			if mode == "quit" {
				quitting = true
				mygo.App.Quit()
			} else {
				win.Hide()
			}
		})

		// 系统托盘：菜单栏(macOS) / 通知区域(Windows、Linux)常驻图标。
		// 默认不设 Menu，使左键点击走 OnClick 直接打开主面板；右键时再临时
		// 弹出菜单（见 OnRightClick），避免“有 Menu 时 OnClick 不触发”的限制。
		tray, err := mygo.NewTray(mygo.TrayOptions{
			Icon:           appIconPNG, // 复用应用图标（mygo 会缩放至托盘尺寸）
			IconIsTemplate: false,      // 彩色图标按原色显示（macOS 不强制黑白模板）
			ToolTip:        appmeta.Title,
			// Menu 留空：左键点击由 OnClick 接管（打开主面板）；
			// 右键菜单在 OnRightClick 中临时构建并弹出。
		})
		if err != nil {
			// 托盘缺失（如 Linux 缺 AppIndicator 库）不应阻断应用启动。
			println("warn: 创建系统托盘失败:", err.Error())
			return
		}

		// 左键点击打开主面板。
		// 平台差异（mygo 托盘模型决定）：
		//   - Linux（AppIndicator）：双击图标触发激活 → 打开主面板（即用户要求的“双击打开”）
		//   - Windows / macOS：单击图标触发 → 打开主面板（mygo 未暴露独立的“双击”事件，
		//     这是框架在该平台能给出的最接近行为；标准 Windows 托盘双击打开亦常见此语义）
		tray.OnClick(func() { win.Show() })

		// 右键：临时挂载菜单并弹出，调用返回后立即清空，保证左键仍可用于打开主面板。
		tray.OnRightClick(func() {
			tray.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
				{Label: "打开主界面", Click: func(*mygo.MenuItem, *mygo.Window) { win.Show() }},
				mygo.Separator(),
				{Role: mygo.RoleQuit}, // 触发退出流程：OnBeforeQuit→quitting=true→允许关闭→应用退出
			}))
			tray.PopUpMenu()
			tray.SetMenu(nil) // 清空，恢复左键打开主面板的能力
		})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}

// ---- 启动期的两项基础设施 ----

// setupLogging 初始化日志系统，返回关闭函数。
//
// 日志目录不可写时不能让应用起不来：退回「不写文件」并打印到 stderr——用户至少
// 还能看到界面，且 Go 运行时仍会输出默认 panic 堆栈。
func setupLogging() func() {
	dir, err := apppath.LogDir()
	if err != nil {
		println("warn: 无法定位日志目录:", err.Error())
		return func() {}
	}
	closeFn, err := logging.Setup(logging.Options{
		Dir:        dir,
		AppName:    appmeta.Name,
		AppVersion: appmeta.BuildVersion(),
		Level:      slog.LevelInfo,
	})
	if err != nil {
		println("warn: 初始化日志失败:", err.Error())
		return func() {}
	}
	// 启动即写环境快照（只进日志文件：此时事件入库管道尚未挂载，见下方 announceStartup）。
	logging.Event(slog.LevelInfo, "app", "应用启动", logging.EnvSnapshot()...)
	return func() { _ = closeFn() }
}

// announceStartup 在「日志 + 数据库」都就绪后写一条启动审计（入库，界面可查）。
//
// 为什么单独一步：入库靠 logging.SetRecorder 挂载，而它必须等数据库打开成功；
// 若在 Setup 阶段就 Audit，事件只会落到文件、进不了库（实测踩过）。
func announceStartup(dbReady bool) {
	storeState := "未启用"
	if dbReady {
		storeState = "已启用"
	}
	logging.Audit("app", "应用启动",
		"build", appmeta.BuildVersion(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"go", runtime.Version(),
		"cpus", runtime.NumCPU(),
		"store", storeState,
	)
}

// openStore 打开本地数据库并接上「错误事件入库」的管道；失败返回 nil（不阻断启动）。
func openStore() *store.Store {
	path, err := apppath.DBPath()
	if err != nil {
		logging.Errv("store", "无法定位数据库路径", err)
		return nil
	}
	s, err := store.Open(path)
	if err != nil {
		logging.Errv("store", "打开数据库失败", err, "path", path)
		return nil
	}
	// ERROR/CRASH 级日志 + 显式 Audit 事件写入 app_event 表。
	// 注意：入库失败时只能写 stderr —— 再调 logging 会再次触发本回调，形成递归。
	logging.SetRecorder(func(level, kind, msg, detail string) {
		if err := s.RecordEvent(level, kind, msg, detail); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "warn: 事件入库失败:", err)
		}
	})
	logging.Event(slog.LevelInfo, "store", "数据库已就绪", "path", path)
	return s
}
