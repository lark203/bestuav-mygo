package view

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
	"bestuav-mygo/internal/view/piano"
)

// ---- 官方示例：钢琴（合成音）----
//
// 移植自 mygo examples/piano，逻辑在 internal/view/piano 包里。
// 本文件只负责把它装进卡片，并注入两个回调：
//   - mygo.RunOnMain：放曲子时按键要回到主线程改状态（上游用的是 win.Update）；
//   - a.requestFrame：改完状态要请求重绘，键才会亮。

// pianoPage 钢琴演示页。
func (a *App) pianoPage(c *ui.Context, t *ui.Theme) {
	if a.pi == nil {
		a.pi = piano.New()
		a.pi.SetMainThread(func(fn func()) { mygo.RunOnMain(fn) }, a.requestFrame)
		// 声音是惰性的：先把 37 个键都合成好，免得弹第一下要等。
		a.pi.Warm()
	}
	// 静音开关是页面级的布尔（Switch 要 *bool），变了才同步给钢琴。
	if a.pi.Muted() != a.piMuted {
		a.pi.SetMuted(a.piMuted)
	}
	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "钢琴（合成音）",
			"37 键：点按或拖过琴键、电脑键盘 Z–M 与 Q–I、空格延音。声音由 audio 包实时合成", func() {
				ui.Text(c, "静音").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				ui.Switch(c, &a.piMuted).Label("静音")
			})
		a.pi.View(c)
	})
}

// closePiano 释放钢琴：先停曲子、止住所有弦，再丢掉状态。
// 不这样做的话，离开页面后 goroutine 还在按时间表弹键，声音会继续响。
func (a *App) closePiano() {
	if a.pi == nil {
		return
	}
	a.pi.Stop()
	a.pi.SetMuted(true) // 顺带把所有弦制住
	a.pi = nil
}
