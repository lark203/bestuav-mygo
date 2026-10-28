package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"bestuav-mygo/internal/view"
)

func main() {
	a := view.NewApp()
	// 跟随系统明暗（不自定义标题栏、不使用紫色强调色）。
	mygo.Theme.SetSource(mygo.ThemeSystem)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:     "量子科学 (mygo)",
			Width:     1100,
			Height:    720,
			MinWidth:  820,
			MinHeight: 520,
			StateKey:  "bestuav-mygo",
			Content:   ui.View(a.View),
		})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
