package view

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/icons"
	"bestuav-mygo/internal/ui/kit"
)

// ---- 官方示例：浮出面板（Flyout）----
//
// 移植自 mygo examples/flyout。Flyout 是「挂在父窗口旁边、能伸出窗口之外」的
// 独立小窗口：跟随父窗口移动、可以拿到键盘、点别处或按 Esc 自动关闭，空间不够时
// 会自动翻到锚点另一侧或滑动回屏幕内。
//
// 与示例的差别（都是为了塞进本应用的页面框架）：
//   - 示例本身就是那个小窗口（300×240），这里收成一页；
//   - 颜色列表的方位做成下拉（示例写死下方），方便真机验证「翻转/滑动」；
//   - 三个按钮下方加一行状态，记录最近一次弹出了什么（真机排查用）；
//   - 补了一张说明卡，写清它与 ui.Modal 的区别。
//
// ⚠️ 验证方式：Flyout 是**独立子窗口**，`ui.Tester` 离屏渲染不到它，本页只能
// 真机点开肉眼确认。离屏能验证的只有：页面骨架能渲染、纯状态逻辑（hover 取模、
// 选中、方位映射、锚点换算）正确。

// flyoutColors 颜色列表的选项（与示例一致）。
var flyoutColors = []struct {
	name  string
	color ui.Color
}{
	{"Red", ui.RGB(0xff, 0x3b, 0x30)},
	{"Orange", ui.RGB(0xff, 0x95, 0x00)},
	{"Yellow", ui.RGB(0xff, 0xcc, 0x00)},
	{"Green", ui.RGB(0x34, 0xc7, 0x59)},
	{"Blue", ui.RGB(0x00, 0x7a, 0xff)},
	{"Purple", ui.RGB(0xaf, 0x52, 0xde)},
}

// flyoutPlacements 颜色列表可挂的方位，用于真机验证越界翻转。
var flyoutPlacements = []struct {
	label string
	place mygo.Placement
}{
	{"下方·左对齐", mygo.PlacementBottomStart},
	{"下方·居中", mygo.PlacementBottom},
	{"上方·左对齐", mygo.PlacementTopStart},
	{"右侧·顶部对齐", mygo.PlacementRightStart},
	{"左侧·顶部对齐", mygo.PlacementLeftStart},
}

// flyoutDemo 浮出面板示例的状态：三个子窗口句柄 + 颜色列表的键盘高亮。
type flyoutDemo struct {
	list   *mygo.Window // 颜色列表（可键盘上下选择），nil = 未弹出
	editor *mygo.Window // 便签编辑框，nil = 未弹出
	about  *mygo.Window // 关于（macOS 上的 popover 外观），nil = 未弹出

	color int    // 已选中的颜色下标
	hover int    // 键盘在列表里移到的下标（弹开时初始化为 color）
	place string // 颜色列表的方位（flyoutPlacements 的 label）
	note  string // 便签内容
	last  string // 最近一次动作的说明（页面底部状态行展示）
}

// 颜色列表的固定尺寸：行高 28 × 项数 + 上下各 6 的内边距。
const (
	flyoutRowH = 28
	flyListW   = 180
)

// flyoutPage 浮出面板示例页。
func (a *App) flyoutPage(c *ui.Context, t *ui.Theme) {
	if a.fo == nil {
		a.fo = &flyoutDemo{place: flyoutPlacements[0].label, last: "还没有弹出过浮层"}
	}
	fo := a.fo

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "浮出面板（Flyout）",
			"从按钮旁弹出的独立小窗口：跟随锚点、越界自动翻转、可获取键盘、点击别处或 Esc 关闭", nil)
		ui.Column(c).Gap(kit.CardGap).Children(func() {
			ui.Row(c).Gap(kit.CardGap).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Width(14).Height(14).Radius(7).Background(flyoutColors[fo.color].color)
				b := ui.Button(c.Key("color"), flyoutColors[fo.color].name+" ▾")
				if b.Clicked() {
					fo.toggleList(a, b.Bounds())
				}
				b = ui.Button(c.Key("note"), "编辑便签…")
				if b.Clicked() {
					fo.openEditor(a, b.Bounds())
				}
				b = ui.Button(c.Key("about"), "关于")
				if b.Clicked() {
					fo.openAbout(a, b.Bounds())
				}
				ui.Spacer(c).Grow(1)
				ui.Select(c, &fo.place, flyoutPlaceLabels()).Label("颜色列表方位")
			})

			// 便签内容（没有内容时给占位文案）。
			ui.Text(c, fo.noteText()).
				FontSize(t.FontSize - 2).TextColor(t.TextMuted)

			// 状态行：真机上点开浮层后能看到这里变化，便于确认回调链路通了。
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Popover).FontSize(13).TextColor(t.TextMuted)
				ui.Text(c, fo.last).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			})
		})
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "与对话框（Modal）的区别", "", nil)
		ui.Column(c).Gap(kit.CardGap - 4).Children(func() {
			kit.KV(c, t, "边界", "浮层可以伸出主窗口之外；对话框永远被关在窗口内部")
			kit.KV(c, t, "定位", "浮层贴着触发它的元素（锚点）；对话框在窗口里居中")
			kit.KV(c, t, "归属", "浮层是父窗口的子窗口，跟着父窗口移动、父窗口关了它就没")
			kit.KV(c, t, "键盘", "Focusable 的浮层自己拿键盘，按 Esc / 点别处自动关闭")
			kit.KV(c, t, "越界", "屏幕工作区放不下时先翻到锚点另一侧，再滑动回屏内，最后才缩")
			kit.KV(c, t, "验证", "浮层是独立窗口，离屏渲染不到 —— 只能真机点开确认")
		})
	})
}

// noteText 便签的展示文案（空内容时给占位）。
func (f *flyoutDemo) noteText() string {
	if f.note == "" {
		return "（便签为空）"
	}
	return f.note
}

// flyoutPlacement 把下拉里的文案换回 Placement；未知文案退回默认（下方）。
func flyoutPlacement(label string) mygo.Placement {
	for _, p := range flyoutPlacements {
		if p.label == label {
			return p.place
		}
	}
	return mygo.PlacementBottomStart
}

// flyoutPlacementLabel 取方位的展示名（未知文案给「默认」）。
func flyoutPlacementLabel(label string) string {
	for _, p := range flyoutPlacements {
		if p.label == label {
			return p.label
		}
	}
	return "默认"
}

// flyoutPlaceLabels 下拉的选项文案。
func flyoutPlaceLabels() []string {
	out := make([]string, len(flyoutPlacements))
	for i, p := range flyoutPlacements {
		out[i] = p.label
	}
	return out
}

// ---- 三个浮层 ----

// toggleList 在按钮下方弹出颜色列表，或关掉已弹出的那个。
func (f *flyoutDemo) toggleList(a *App, at ui.Rect) {
	if f.list != nil {
		f.list.Close()
		f.last = "已关闭颜色列表"
		return
	}
	if a.win == nil {
		f.last = "未注入窗口句柄，无法弹出浮层（离屏渲染 / 单测环境）"
		return
	}
	f.hover = f.color
	p := flyoutPlacement(f.place)
	f.list = mygo.NewFlyout(mygo.FlyoutOptions{
		Parent:    a.win,
		Anchor:    anchorRect(at),
		Placement: p,
		Gap:       4,
		Width:     flyListW,
		Height:    len(flyoutColors)*flyoutRowH + 12,
		Focusable: true,
		Shadow:    true,
		Content:   ui.View(func(c *ui.Context) { f.listView(a, c) }),
	})
	f.list.OnClosed(func() {
		f.list = nil
		a.requestFrame()
	})
	f.last = "已弹出颜色列表（" + flyoutPlacementLabel(f.place) + "）"
	a.requestFrame()
}

// listView 颜色列表：方向键移动高亮，回车选中，悬停同步高亮。
func (f *flyoutDemo) listView(a *App, c *ui.Context) {
	t := c.Theme()
	if c.Shortcut(0, ui.KeyDown) {
		f.move(1)
	}
	if c.Shortcut(0, ui.KeyUp) {
		f.move(-1)
	}
	if c.Shortcut(0, ui.KeyEnter) {
		f.choose(a, f.hover)
		return
	}
	c.Root().Background(ui.Transparent)
	flyPanel(c, t).Padding(6).Children(func() {
		for i, col := range flyoutColors {
			row := ui.Row(c.Key(i)).Height(flyoutRowH).Gap(8).Padding(0, 8).
				AlignItems(ui.Center).Radius(6)
			if i == f.hover {
				row.Background(t.Accent)
			}
			if row.Hovered() && f.hover != i {
				f.hover = i
				c.Invalidate()
			}
			if row.Clicked() {
				f.choose(a, i)
				return
			}
			row.Children(func() {
				ui.Box(c).Width(12).Height(12).Radius(6).Background(col.color)
				txt := ui.Text(c, col.name)
				if i == f.hover {
					txt.TextColor(t.AccentText)
				}
			})
		}
	})
}

// move 用方向键在列表里移动高亮（两端循环）。
func (f *flyoutDemo) move(delta int) {
	n := len(flyoutColors)
	f.hover = ((f.hover+delta)%n + n) % n
}

// choose 选中颜色并关闭列表。
func (f *flyoutDemo) choose(a *App, i int) {
	if i < 0 || i >= len(flyoutColors) {
		return
	}
	f.color = i
	f.last = "已选颜色 " + flyoutColors[i].name
	if f.list != nil {
		f.list.Close()
	}
	a.requestFrame()
}

// openEditor 在按钮右侧弹出便签编辑框（自己拿键盘，回车关闭）。
func (f *flyoutDemo) openEditor(a *App, at ui.Rect) {
	if f.editor != nil || a.win == nil {
		return
	}
	f.editor = mygo.NewFlyout(mygo.FlyoutOptions{
		Parent:    a.win,
		Anchor:    anchorRect(at),
		Placement: mygo.PlacementRightStart,
		Gap:       8,
		Width:     260,
		Height:    96,
		Focusable: true,
		Shadow:    true,
		Content:   ui.View(func(c *ui.Context) { f.editorView(c) }),
	})
	f.editor.OnClosed(func() {
		f.editor = nil
		f.last = "便签已保存：" + f.noteText()
		a.requestFrame()
	})
	f.last = "已弹出便签编辑框（右侧）"
	a.requestFrame()
}

func (f *flyoutDemo) editorView(c *ui.Context) {
	c.Root().Background(ui.Transparent)
	flyPanel(c, c.Theme()).Padding(12).Gap(8).Children(func() {
		ui.Text(c, "便签").Bold()
		ui.TextInput(c.Key("note"), &f.note).Placeholder("写点什么").AutoFocus()
		if c.Shortcut(0, ui.KeyEnter) {
			f.editor.Close()
		}
	})
}

// openAbout 在按钮上方弹出「关于」，macOS 上是系统 popover 的材质（毛玻璃）。
func (f *flyoutDemo) openAbout(a *App, at ui.Rect) {
	if f.about != nil || a.win == nil {
		return
	}
	f.about = mygo.NewFlyout(mygo.FlyoutOptions{
		Parent:    a.win,
		Anchor:    anchorRect(at),
		Placement: mygo.PlacementTop,
		Gap:       6,
		Width:     240,
		Height:    84,
		Focusable: true,
		Shadow:    true,
		Popover:   true,
		Content:   ui.View(func(c *ui.Context) { f.aboutView(c) }),
	})
	f.about.OnClosed(func() {
		f.about = nil
		a.requestFrame()
	})
	f.last = "已弹出「关于」（上方 · popover）"
	a.requestFrame()
}

func (f *flyoutDemo) aboutView(c *ui.Context) {
	t := c.Theme()
	c.Root().Background(ui.Transparent)
	// 有毛玻璃时直接铺在材质上（窗口自己就是 popover）；没有才自己画面板。
	box := ui.Column(c).Fill()
	if !c.Vibrancy() {
		box = flyPanel(c, t)
	}
	box.Padding(12).Gap(4).Children(func() {
		ui.Text(c, "Flyout").Bold()
		ui.Text(c, "子窗口归父窗口所有，贴着锚点摆放。").
			FontSize(12).TextColor(t.TextMuted)
	})
}

// ---- 工具 ----

// flyPanel 浮层里那块面板：窗口自身不画东西，面板才画。
// macOS 上窗口自带阴影与圆角，Windows 自己会圆角描边，其余平台补边框。
func flyPanel(c *ui.Context, t *ui.Theme) ui.Element {
	box := ui.Column(c).Fill().Background(t.Surface)
	switch runtime.GOOS {
	case "darwin":
		box.Radius(10)
	case "windows":
	default:
		box.Radius(10).Border(1, t.Border)
	}
	return box
}

// anchorRect 把元素的矩形换算成 Flyout 的锚点（整数 DIP，相对父窗口内容区）。
func anchorRect(r ui.Rect) mygo.Rectangle {
	return mygo.Rectangle{X: int(r.X), Y: int(r.Y), Width: int(r.W), Height: int(r.H)}
}

// closeFlyouts 关掉所有已弹出的浮层并清空状态。页面离开时调用，
// 避免留下孤儿子窗口。
func (a *App) closeFlyouts() {
	if a.fo == nil {
		return
	}
	for _, w := range []*mygo.Window{a.fo.list, a.fo.editor, a.fo.about} {
		if w != nil {
			w.Close()
		}
	}
	a.fo = nil
}
