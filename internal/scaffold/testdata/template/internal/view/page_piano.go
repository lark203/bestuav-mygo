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

// pianoStateKey 钢琴实例在页面私有状态里的键名。
const pianoStateKey = "piano"

// pianoPage 钢琴演示页。
//
// 钢琴实例存在本页的 pageState.data 里：切走再回来接着弹，离开页面时由
// closePiano 停声（onDestroy 会在 data 被丢弃之前调用它）。
func (a *App) pianoPage(c *ui.Context, t *ui.Theme) {
	p := pageStateOf(a, "official-piano", pianoStateKey, func() *piano.Piano {
		inst := piano.New()
		inst.SetMainThread(func(fn func()) { mygo.RunOnMain(fn) }, a.requestFrame)
		// 声音是惰性的：先把 37 个键都合成好，免得弹第一下要等。
		inst.Warm()
		return inst
	})
	// 静音开关是页面级的布尔（Switch 要 *bool），变了才同步给钢琴。
	if p.Muted() != a.piMuted {
		p.SetMuted(a.piMuted)
	}
	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "钢琴（合成音）",
			"37 键：点按或拖过琴键、电脑键盘 Z–M 与 Q–I、空格延音。声音由 audio 包实时合成", func() {
				ui.Text(c, "静音").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				ui.Switch(c, &a.piMuted).Label("静音")
			})
		p.View(c)
	})
}

// pianoInstance 取当前钢琴实例（可能为 nil，例如还没进过该页）。
func (a *App) pianoInstance() *piano.Piano {
	st, ok := a.pages["official-piano"]
	if !ok {
		return nil
	}
	p, _ := st.data[pianoStateKey].(*piano.Piano)
	return p
}

// stopPiano 只停曲子，保留状态（离开页面时用：键盘与档位要留着）。
func (a *App) stopPiano() {
	if p := a.pianoInstance(); p != nil {
		p.Stop()
	}
}

// closePiano 释放钢琴：先停曲子、止住所有弦，再丢掉状态。
// 不这样做的话，离开页面后 goroutine 还在按时间表弹键，声音会继续响。
func (a *App) closePiano() {
	st, ok := a.pages["official-piano"]
	if !ok {
		return
	}
	p := a.pianoInstance()
	if p == nil {
		return
	}
	p.Stop()
	p.SetMuted(true) // 顺带把所有弦制住
	delete(st.data, pianoStateKey)
}
