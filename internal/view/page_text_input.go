package view

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
)

// ---- 官方示例：自绘输入框（文本原语）----
//
// 移植自 mygo examples/text-input。它演示的不是 ui.TextInput 这类成品控件
// （那些在「组件演示 → 输入框组件」里），而是更底层的**文本输入原语**：
// 文本存在应用自己的缓冲区里，选区、输入法组字（marked range）、剪贴板策略
// 和绘制全部由应用负责，原语只提供系统级文本服务（输入法、编辑菜单、光标）。
//
// 与示例的差别：
//   - 去掉窗口级菜单（RoleEditMenu），编辑菜单命令由原语直接派发给本页的输入处理；
//   - 加了中文示例文本（原示例只有 é / 😀 / 阿拉伯语 / 希伯来语）；
//   - 下方加了一行实时状态（选区与 UTF-16 长度），方便看懂「选区是 UTF-16 下标」。

// textField 自持有文本的输入框状态：文本、选区、输入法正在组的字、以及排版结果。
//
// 更大的编辑器可以把 text 换成自己的 buffer / 属性 / 撤销栈，接口不变。
type textField struct {
	text      string
	selection ui.TextInputSelection
	marked    *ui.TextInputRange
	layout    *ui.TextLayout
}

// newTextField 造一个示例输入框，光标停在文本末尾。
func newTextField(text string) *textField {
	f := &textField{text: text}
	end := ui.UTF16Len(f.text)
	f.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: end, End: end}}
	f.shape()
	return f
}

// shape 重新排版（文本变了就得调一次，后面取光标/选区矩形都靠它）。
func (f *textField) shape() { f.layout = ui.ShapeText(f.text, ui.Font{Size: 18}, 0) }

// ---- ui.TextInputClient 的实现 ----

func (f *textField) TextForRange(r ui.TextInputRange) (string, ui.TextInputRange) {
	a, z := ui.UTF16ByteOffset(f.text, r.Start), ui.UTF16ByteOffset(f.text, r.End)
	return f.text[a:z], ui.TextInputRange{Start: ui.UTF16Len(f.text[:a]), End: ui.UTF16Len(f.text[:z])}
}

func (f *textField) Selection() ui.TextInputSelection { return f.selection }

func (f *textField) MarkedRange() (ui.TextInputRange, bool) {
	if f.marked != nil {
		return *f.marked, true
	}
	return ui.TextInputRange{}, false
}

// target 决定一次替换落在哪：给了范围就用它，否则用输入法组字的范围，最后才是选区。
func (f *textField) target(r *ui.TextInputRange) ui.TextInputRange {
	if r != nil {
		return *r
	}
	if f.marked != nil {
		return *f.marked
	}
	return f.selection.Range
}

// ReplaceText 替换一段文本（r 为 nil 时替换当前选区 / 组字区）。
func (f *textField) ReplaceText(r *ui.TextInputRange, text string) {
	wanted := f.target(r)
	a, z := ui.UTF16ByteOffset(f.text, wanted.Start), ui.UTF16ByteOffset(f.text, wanted.End)
	f.text = f.text[:a] + text + f.text[z:]
	end := wanted.Start + ui.UTF16Len(text)
	f.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: end, End: end}}
	f.marked = nil
	f.shape()
}

// SetMarkedText 输入法组字：先落文本，再把这段标成「正在组」，选区落在组字串内。
func (f *textField) SetMarkedText(r *ui.TextInputRange, text string, selected ui.TextInputRange) {
	wanted := f.target(r)
	f.ReplaceText(&wanted, text)
	mark := ui.TextInputRange{Start: wanted.Start, End: wanted.Start + ui.UTF16Len(text)}
	f.marked = &mark
	f.selection = ui.TextInputSelection{Range: ui.TextInputRange{
		Start: mark.Start + selected.Start, End: mark.Start + selected.End,
	}}
}

func (f *textField) UnmarkText() { f.marked = nil }

// BoundsForRange 返回一段文本的位置，系统用它摆放输入法候选窗。
// 绘制时整体偏移了 (12, 12)，这里要补回来。
func (f *textField) BoundsForRange(r ui.TextInputRange) (ui.Rect, ui.TextInputRange, bool) {
	b := f.layout.Caret(r.Start)
	if r.Start != r.End {
		if rects := f.layout.SelectionRects(r); len(rects) > 0 {
			b = rects[0]
		}
	}
	b.X += fieldPadX
	b.Y += fieldPadY
	return b, r, true
}

// IndexForPoint 把点击位置换算成文本下标（同样是减掉绘制偏移）。
func (f *textField) IndexForPoint(p ui.Point) (int, bool) {
	return f.layout.IndexAt(ui.Point{X: p.X - fieldPadX, Y: p.Y - fieldPadY}), true
}

// 文本在框内的内边距：绘制偏移与命中测试都用它，必须一致（否则输入法候选窗会飘）。
// 纵向比横向大，是为了让 18px 的单行文本在 64 高的框里视觉居中。
const (
	fieldPadX = 12
	fieldPadY = 20
)

// move 把光标移到 to；extend 为真时保留锚点做「Shift 选择」。
func (f *textField) move(to int, extend bool) {
	if !extend {
		f.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: to, End: to}}
		return
	}
	anchor := f.selection.Range.Start
	if f.selection.Reversed {
		anchor = f.selection.Range.End
	}
	f.selection = ui.TextInputSelection{
		Range:    ui.TextInputRange{Start: min(anchor, to), End: max(anchor, to)},
		Reversed: to < anchor,
	}
}

// input 处理按键、指针与编辑菜单命令（复制/剪切/粘贴/全选/删除）。
func (f *textField) input(ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputPointerDown:
		to, _ := f.IndexForPoint(ui.Point{X: ev.X, Y: ev.Y})
		f.move(to, ev.Mods&ui.Shift != 0)
		return true
	case ui.InputPointerMove:
		if ev.Button == 0 {
			to, _ := f.IndexForPoint(ui.Point{X: ev.X, Y: ev.Y})
			f.move(to, true)
			return true
		}
	case ui.InputPointerUp:
		return true
	case ui.InputKeyDown:
		if ev.Mods == ui.Cmd && ev.Key == ui.KeyA {
			f.selectAll()
			return true
		}
		switch ev.Key {
		case ui.KeyHome:
			f.move(0, ev.Mods&ui.Shift != 0)
			return true
		case ui.KeyEnd:
			f.move(ui.UTF16Len(f.text), ev.Mods&ui.Shift != 0)
			return true
		case ui.KeyBackspace, ui.KeyDelete:
			r := f.selection.Range
			if r.Start == r.End {
				next := f.layout.PreviousBoundary(r.Start)
				if ev.Key == ui.KeyDelete {
					next = f.layout.NextBoundary(r.End)
				}
				r = ui.TextInputRange{Start: min(next, r.Start), End: max(next, r.End)}
			}
			f.ReplaceText(&r, "")
			return true
		}
	case ui.InputCommand:
		switch ev.Text {
		case "selectAll":
			f.selectAll()
		case "copy", "cut":
			s, _ := f.TextForRange(f.selection.Range)
			mygo.Clipboard.WriteText(s)
			if ev.Text == "cut" {
				f.ReplaceText(nil, "")
			}
		case "paste":
			f.ReplaceText(nil, mygo.Clipboard.ReadText())
		case "delete":
			f.ReplaceText(nil, "")
		default:
			return false
		}
		return true
	}
	return false
}

// selectAll 全选：范围是「从头到 UTF-16 长度」，不是字节长度。
func (f *textField) selectAll() {
	f.selection = ui.TextInputSelection{Range: ui.TextInputRange{End: ui.UTF16Len(f.text)}}
}

// ---- 页面 ----

// textInputPage 自绘输入框页。
func (a *App) textInputPage(c *ui.Context, t *ui.Theme) {
	if a.tf == nil {
		a.tf = newTextField("在这里输入：é, 😀, العربية, 中文, 混合文本")
	}
	f := a.tf

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "自绘输入框（文本原语）",
			"文本存在应用自己的缓冲区里：选区、输入法组字、剪贴板与绘制都由应用负责，"+
				"原语只提供系统级文本服务。试试中文输入法。", nil)

		// 输入框本体：一个盒子 + 自己画选区/文本/光标。
		ui.Box(c).Height(64).FillWidth().
			Background(t.Surface).Border(1, t.Border).Radius(t.Radius).
			Label("自绘文本输入框").Role(ui.RoleTextField).Value(f.text).
			HandleTextInput(f).AutoFocus().
			HandleInput(func(ev ui.InputEvent) bool { return f.input(ev) }).
			Draw(func(p *ui.Painter, r ui.Rect) {
				// 选区底色
				for _, b := range f.layout.SelectionRects(f.selection.Range) {
					b.X += r.X + fieldPadX
					b.Y += r.Y + fieldPadY
					p.Fill(b, t.Selection, 0)
				}
				p.TextLayout(f.layout, r.X+fieldPadX, r.Y+fieldPadY, t.Text)
				// 光标
				b := f.layout.Caret(f.selection.Caret())
				b.X += r.X + fieldPadX
				b.Y += r.Y + fieldPadY
				p.Fill(b, t.Accent, 0)
			})

		// 实时状态：让人看懂「选区是 UTF-16 下标、不是字节下标」。
		s := f.selection.Range
		ui.Text(c, "选区 "+itoa(s.Start)+"–"+itoa(s.End)+
			" · 文本 UTF-16 长度 "+itoa(ui.UTF16Len(f.text))+
			" · 字节长度 "+itoa(len(f.text))).
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "这一页和「输入框组件」有什么不同",
			"那边是成品控件（ui.TextInput / TextArea / DateInput），这边是控件下面那一层原语", nil)
		kit.KV(c, t, "文本归属", "存在应用自己的 string 里，可以换成任意缓冲区、属性与撤销栈")
		kit.KV(c, t, "下标单位", "选区与光标都是 UTF-16 下标（中文 1 个、emoji 2 个），取文本时要换算成字节")
		kit.KV(c, t, "输入法组字", "SetMarkedText / UnmarkText 标记「正在组」的范围，候选窗位置由 BoundsForRange 给出")
		kit.KV(c, t, "剪贴板策略", "copy / cut / paste 以编辑菜单命令的形式派发过来，写什么、删什么由应用决定")
		kit.KV(c, t, "绘制", "选区底色、文本、光标全部自己画：Painter.TextLayout / Fill")
	})
}
