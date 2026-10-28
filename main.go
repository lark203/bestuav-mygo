package main

import (
	"bestuav-mygo/internal/view"
	_ "embed"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
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
	a := view.NewApp()
	// 跟随系统明暗（不自定义标题栏、不使用紫色强调色）。
	mygo.Theme.SetSource(mygo.ThemeSystem)
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     "量子科学 (mygo)",
			Width:     1100,
			Height:    720,
			MinWidth:  820,
			MinHeight: 520,
			StateKey:  "bestuav-mygo",
			Content:   ui.View(a.View),
		})
		// 原生窗口创建完成后再设置图标，避免平台窗口尚未就绪导致设置失败。
		win.OnReadyToShow(func() {
			if err := win.SetIcon(appIconPNG); err != nil {
				// 图标缺失不应阻断启动：仅打印告警（部分平台窗口未就绪时会返回错误）。
				println("warn: 设置窗口图标失败:", err.Error())
			}
		})

		// 关闭窗口时最小化到托盘，而非直接退出（标准托盘应用行为）。
		// 仅当通过托盘“退出”触发退出流程时（quitting=true）才允许真正关闭。
		var quitting bool
		mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })
		win.OnClose(func(e *mygo.CloseEvent) {
			if !quitting {
				e.PreventDefault()
				win.Hide()
			}
		})

		// 系统托盘：菜单栏(macOS) / 通知区域(Windows、Linux)常驻图标。
		// 默认不设 Menu，使左键点击走 OnClick 直接打开主面板；右键时再临时
		// 弹出菜单（见 OnRightClick），避免“有 Menu 时 OnClick 不触发”的限制。
		tray, err := mygo.NewTray(mygo.TrayOptions{
			Icon:           appIconPNG, // 复用应用图标（mygo 会缩放至托盘尺寸）
			IconIsTemplate: false,      // 彩色图标按原色显示（macOS 不强制黑白模板）
			ToolTip:        "量子科学",
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
