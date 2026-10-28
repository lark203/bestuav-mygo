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
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
