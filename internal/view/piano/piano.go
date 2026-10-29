// Package piano 移植自 mygo examples/piano：一架三个八度的钢琴（C3–C6）。
//
// 玩法：鼠标点按或拖过琴键、电脑键盘（Z–M 与 Q–I 两排）、空格踩延音踏板、
// ← → 换八度，或者放四首内置曲子。
//
// 声音是**合成的**：synth.go 按泛音叠加算出每个琴键的波形（含琴槌的撞击声），
// 不解码任何音频文件，因此零资源依赖。
package piano

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

// 键盘的音域，MIDI 音高编号：C3 到 C6。
const (
	Lowest  = 48
	Highest = 84
)

// Piano 是一架钢琴的状态：谁按着哪些键、踏板、八度、正在放的曲子。
type Piano struct {
	held     map[int]*audio.Player // 正在发声的弦，按音高
	sources  map[int]int           // 每个音被几处按着（指针、电脑键、曲子各算一处）
	keysDown map[ui.Key]int        // 电脑键盘按下时对应的音高
	pointer  int                   // 指针按住的音，没有则 -1
	sustain  bool                  // 延音踏板
	octave   int                   // 下排琴键从哪个八度起
	song     *Song                 // 正在放的曲子，没有则 nil
	songHeld []int                 // 曲子当前按住的音
	scale    float32               // 画出来的键盘相对满尺寸的缩放，输入要除回去
	muted    bool                  // 静音：只记状态不发声（可测，也给用户一个开关）

	// 由页面注入的两个回调：
	//   runOnMain —— 把一段代码放到主线程跑（mygo.RunOnMain）；
	//   redraw    —— 请求重绘。
	// 两者都允许为 nil：单测里没有它们，回调退化成同步执行与空操作。
	runOnMain func(func())
	redraw    func()
}

// New 造一架钢琴。声音是惰性合成的：第一次弹到才算，
// 页面进入时再调 Warm 预热，避免刚进来第一下要等。
func New() *Piano {
	return &Piano{
		held:     map[int]*audio.Player{},
		sources:  map[int]int{},
		keysDown: map[ui.Key]int{},
		pointer:  -1,
		octave:   4,
		scale:    1,
	}
}

// SetMainThread 注入主线程执行器与重绘触发器，由页面装配时调用。
func (p *Piano) SetMainThread(runOnMain func(func()), redraw func()) {
	p.runOnMain = runOnMain
	p.redraw = redraw
}

// SetMuted 开关静音。静音时按键仍然改变状态（键会亮），但不发声。
func (p *Piano) SetMuted(muted bool) {
	p.muted = muted
	if muted {
		p.releaseAll()
	}
}

// Muted 报告当前是否静音。
func (p *Piano) Muted() bool { return p.muted }

// Octave 报告下排琴键的起始八度。
func (p *Piano) Octave() int { return p.octave }

// Playing 报告是否正在放曲子。
func (p *Piano) Playing() bool { return p.song != nil }

func (p *Piano) onMain(fn func()) {
	if p.runOnMain == nil {
		fn()
		p.redrawNow()
		return
	}
	p.runOnMain(fn)
}

func (p *Piano) redrawNow() {
	if p.redraw != nil {
		p.redraw()
	}
}

// Warm 在后台把每个琴键的声音先合成出来，这样弹第一下不用等。
func (p *Piano) Warm() {
	go func() {
		for n := Lowest; n <= Highest; n++ {
			sound(n)
		}
	}()
}

// down 按下 note：响到最后一处按住它的地方松手为止。
func (p *Piano) down(note int) {
	if note < Lowest || note > Highest {
		return
	}
	p.sources[note]++
	if p.sources[note] > 1 {
		return
	}
	if old := p.held[note]; old != nil {
		damp(old)
	}
	if p.muted {
		return
	}
	if pl := sound(note).Play(); pl != nil {
		p.held[note] = pl
	}
}

// up 松开 note：踏板没踩下时，制音器把它止住。
func (p *Piano) up(note int) {
	if p.sources[note] == 0 {
		return
	}
	if p.sources[note]--; p.sources[note] > 0 {
		return
	}
	delete(p.sources, note)
	if !p.sustain {
		if pl := p.held[note]; pl != nil {
			damp(pl)
			delete(p.held, note)
		}
	}
}

// pedal 踩下或抬起延音踏板；抬起时，已经松手的那些弦要止住。
func (p *Piano) pedal(down bool) {
	p.sustain = down
	if down {
		return
	}
	for note, pl := range p.held {
		if p.sources[note] == 0 {
			damp(pl)
			delete(p.held, note)
		}
	}
}

// releaseAll 立刻止住所有弦（切静音、离开页面时用）。
func (p *Piano) releaseAll() {
	for note, pl := range p.held {
		damp(pl)
		delete(p.held, note)
	}
}

// damp 让一根弦在 150 ms 内淡出并停止，像制音器落下那样。
func damp(pl *audio.Player) {
	if pl == nil {
		return
	}
	go func() {
		v := pl.Volume()
		for i := 9; i >= 0; i-- {
			pl.SetVolume(v * float64(i) / 10)
			time.Sleep(15 * time.Millisecond)
		}
		pl.Close()
	}()
}

// computerKeys 是电脑键盘上两排键对应的音，按音高顺序排列（tracking 的摆法）。
var computerKeys = []struct {
	key   ui.Key
	label string
}{
	{ui.KeyZ, "Z"}, {ui.KeyS, "S"}, {ui.KeyX, "X"}, {ui.KeyD, "D"}, {ui.KeyC, "C"}, {ui.KeyV, "V"},
	{ui.KeyG, "G"}, {ui.KeyB, "B"}, {ui.KeyH, "H"}, {ui.KeyN, "N"}, {ui.KeyJ, "J"}, {ui.KeyM, "M"},
	{ui.KeyQ, "Q"}, {ui.Key2, "2"}, {ui.KeyW, "W"}, {ui.Key3, "3"}, {ui.KeyE, "E"}, {ui.KeyR, "R"},
	{ui.Key5, "5"}, {ui.KeyT, "T"}, {ui.Key6, "6"}, {ui.KeyY, "Y"}, {ui.Key7, "7"}, {ui.KeyU, "U"},
	{ui.KeyI, "I"},
}

// keyNotes 是 computerKeys 各自相对本八度 C 的半音数；逗号键接在 M 之后，
// 和 Q 同一个音。
var keyNotes = func() map[ui.Key]int {
	m := map[ui.Key]int{ui.KeyComma: 12}
	for i, k := range computerKeys {
		m[k.key] = i
	}
	return m
}()

// input 处理键盘与指针事件；返回 true 表示这个事件被钢琴吃掉了。
func (p *Piano) input(ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputKeyDown, ui.InputKeyUp:
		if ev.Mods&(ui.Super|ui.Ctrl|ui.Alt) != 0 {
			return false
		}
		down := ev.Kind == ui.InputKeyDown
		switch k := ev.PhysicalKey; k {
		case ui.KeySpace:
			p.pedal(down)
		case ui.KeyLeft, ui.KeyRight:
			if down && !ev.Repeat {
				if k == ui.KeyLeft {
					p.octave = max(p.octave-1, 3)
				} else {
					p.octave = min(p.octave+1, 4)
				}
			}
		default:
			off, ok := keyNotes[k]
			if !ok {
				return false
			}
			if ev.Repeat {
				return true
			}
			if down {
				note := (p.octave+1)*12 + off
				p.keysDown[k] = note
				p.down(note)
			} else if note, ok := p.keysDown[k]; ok {
				delete(p.keysDown, k)
				p.up(note)
			}
		}
	case ui.InputPointerDown, ui.InputPointerMove:
		if ev.Kind == ui.InputPointerMove && ev.Button != 0 {
			return false
		}
		// 拖过琴键时，进入哪个就弹哪个。
		note := keyAt(ev.X/p.scale, ev.Y/p.scale)
		if note != p.pointer {
			if p.pointer >= 0 {
				p.up(p.pointer)
			}
			p.pointer = note
			if note >= 0 {
				p.down(note)
			}
		}
	case ui.InputPointerUp:
		if p.pointer >= 0 {
			p.up(p.pointer)
			p.pointer = -1
		}
	default:
		return false
	}
	p.redrawNow()
	return true
}

// ---- 键盘的几何（单位 DIP）----

const (
	whiteWidth     = 36
	blackWidth     = 22
	keyboardHeight = 180
	blackHeight    = 110
)

// KeyboardWidth 是键盘满尺寸的宽度；页面用它限制最大宽度、draw 用它算缩放。
var KeyboardWidth = float32(Whites()) * whiteWidth

// aspect 是键盘的宽高比，给 Box 的 AspectRatio 用（满宽时正好是设计高度）。
var aspect = KeyboardWidth / keyboardHeight

// Black 报告 note 是不是黑键。
func Black(note int) bool { return "010100101010"[note%12] == '1' }

// Whites 数出键盘上有几个白键。
func Whites() int {
	n := 0
	for note := Lowest; note <= Highest; note++ {
		if !Black(note) {
			n++
		}
	}
	return n
}

// keyRect 给出 note 的琴键在键盘上的位置。
func keyRect(note int) ui.Rect {
	w := 0
	for n := Lowest; n < note; n++ {
		if !Black(n) {
			w++
		}
	}
	if Black(note) {
		return ui.Rect{X: float32(w)*whiteWidth - blackWidth/2, W: blackWidth, H: blackHeight}
	}
	return ui.Rect{X: float32(w) * whiteWidth, W: whiteWidth, H: keyboardHeight}
}

func inside(r ui.Rect, x, y float32) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// keyAt 返回 x, y 处的琴键音高；黑键压在白键之上，都没有则 -1。
func keyAt(x, y float32) int {
	for note := Lowest; note <= Highest; note++ {
		if Black(note) && inside(keyRect(note), x, y) {
			return note
		}
	}
	for note := Lowest; note <= Highest; note++ {
		if !Black(note) && inside(keyRect(note), x, y) {
			return note
		}
	}
	return -1
}

// labels 是每个音在电脑键盘上对应的键名。
func (p *Piano) labels() map[int]string {
	m := map[int]string{}
	for i, k := range computerKeys {
		m[(p.octave+1)*12+i] = k.label
	}
	return m
}

// ---- 界面 ----

// View 画出钢琴：曲子按钮、键盘、操作提示。页面负责把它放进卡片。
func (p *Piano) View(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			for _, s := range Songs {
				if ui.Button(c.Key(s.Name), s.Name).Clicked() {
					p.Play(s)
				}
			}
			if ui.Button(c.Key("stop"), "停止").Disabled(p.song == nil).Clicked() {
				p.Stop()
			}
		})
		// 键盘宽度跟随容器、按比例定高：容器窄时缩小，宽时放大，永不溢出。
		// （不要写 MaxWidth：它会把宽度钉成满尺寸，容器比它窄时直接画到卡片外。）
		ui.Box(c).FillWidth().AspectRatio(aspect).
			Focusable().Focus().Label("钢琴").
			HandleInput(p.input).
			Draw(func(pt *ui.Painter, r ui.Rect) { p.draw(pt, r, t) })
		ui.Text(c, fmt.Sprintf("琴键 Z–M 与 Q–I 从 C%d 起 · 空格延音 · ← → 换八度", p.octave)).
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		if err := audio.Err(); err != nil {
			ui.Text(c, "音频设备不可用："+err.Error()).
				FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		}
	})
}

// draw 画出琴键；按下的键染上强调色。
func (p *Piano) draw(pt *ui.Painter, r ui.Rect, t *ui.Theme) {
	labels := p.labels()
	sounding := func(note int) bool { return p.sources[note] > 0 }
	// 键盘随容器宽度缩放；输入坐标要除回这个比例。
	s := r.W / KeyboardWidth
	p.scale = s
	at := func(k ui.Rect) ui.Rect {
		return ui.Rect{X: r.X + k.X*s, Y: r.Y + k.Y*s, W: k.W * s, H: k.H * s}
	}
	for note := Lowest; note <= Highest; note++ {
		if Black(note) {
			continue
		}
		k := at(keyRect(note))
		face := ui.Hex("#fbfbf8")
		if sounding(note) {
			face = t.Accent.Mix(face, 0.55)
		}
		pt.Fill(k, face, 4*s)
		pt.Stroke(k, ui.Hex("#a1a1aa"), 4*s, 1)
		if l := labels[note]; l != "" {
			pt.Text(k.X+k.W/2-4*s, k.Y+k.H-24*s, l, 11*s, ui.Hex("#71717a"))
		}
		if note%12 == 0 {
			pt.Text(k.X+4*s, k.Y+k.H-42*s, fmt.Sprintf("C%d", note/12-1), 10*s, ui.Hex("#a1a1aa"))
		}
	}
	for note := Lowest; note <= Highest; note++ {
		if !Black(note) {
			continue
		}
		k := at(keyRect(note))
		face := ui.Hex("#18181b")
		if sounding(note) {
			face = t.Accent
		}
		pt.Fill(k, face, 3*s)
		if l := labels[note]; l != "" {
			pt.Text(k.X+k.W/2-4*s, k.Y+k.H-20*s, l, 10*s, ui.Hex("#d4d4d8"))
		}
	}
}
