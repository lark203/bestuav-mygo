// Package effort 移植自 mygo examples/effort-slider：一张「投入多少脑力」的卡片，
// 滑块从 Low 一路到 Max，拇指是一颗大脑；再往右推过 Max 就进 Galaxy —— 大脑炸开、
// 卡片抖一下、轨道里填满闪烁的像素、点状光线从大脑里射出来。
//
// 整个卡片是**自绘**的（ui.Painter 逐帧画路径、渐变、位图光晕），唯一的真控件是
// 压在轨道上的 SliderBase：它不画东西，只负责接指针与方向键、给辅助技术报告
// 「这是个滑块」。所以这里没有任何主题色，颜色是示例自己的一套暗色。
//
// 自绘用到的手法（缓动曲线、模糊字、柔光、圆角路径、SVG 图标）都在
// internal/ui/anim 与 internal/ui/canvas 里，是公共的：本包只留示例自己的
// 状态机与版式。
package effort

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/anim"
	"bestuav-mygo/internal/ui/canvas"
)

// The card's parts, in DIPs from its top left corner, as the demo's page
// lays them out.
const (
	cardW, cardH = 220, 129.5

	cardRadius = 8

	titleSize     = 13.25
	titleBaseline = 26
	effortX       = 12.2
	valueX        = 54.6

	helpX, helpY, helpRadius = 200, 20.9, 6
	helpMarkDX               = 0.3

	labelSize     = 12
	labelBaseline = 60.4
	fasterX       = 12.1
	smarterRight  = 208.1

	trackX, trackY, trackW, trackH, trackRadius = 12, 75, 196, 20, 5

	// The ticks of the levels, 34.8 DIPs apart; the second, the one
	// recommended, is a bar.
	tickX, tickGap, tickSize = 23, 34.8, 3.0
	barH                     = 10

	recommendedX, recommendedBaseline = 24.6, 114

	// The thumb's center goes 36 DIPs a level from stopX.
	stopX, stopGap = 20, 36
	brainSize      = 24
	galaxyBrain    = 32.0 // the brain's size in Galaxy
	galaxyBrainY   = 85.1
	brainY         = 85.5 // the brain's center

	// The fill ends under the brain, short of its center.
	fillInset = 10
)

// CanvasH 画布高度：卡片 129.5 居中，上下留够它的阴影（offsetY 10、模糊 15）。
//
// 页面按这个高度给容器；View 也按它决定卡片在画布里的垂直位置。
const CanvasH = 180

// cardTop 卡片在画布里的顶边（取整 DIP，边缘才不糊）。
var cardTop = float32(math.Round(float64(CanvasH-cardH) / 2))

// Levels 档位名，从低到高；最后一个是 Galaxy。
var Levels = []string{"Low", "Medium", "High", "Extra", "Max", "Galaxy"}

// levels 包内别名，省得把上游代码全改一遍。
var levels = Levels

// Galaxy 是最后一个档位的下标，也是 SliderBase 的上界。
const Galaxy = 5

const galaxy = Galaxy

var (
	pageColor      = ui.Hex("#101010")
	cardColor      = ui.Hex("#171717")
	borderColor    = ui.Hex("#262626")
	trackColor     = ui.Hex("#262626")
	titleColor     = ui.Hex("#767676")
	valueColor     = ui.Hex("#e5e5e5")
	labelColor     = ui.Hex("#6f6f6f")
	recommendColor = ui.Hex("#777777")
	dotColor       = ui.Hex("#525252")
	barColor       = ui.Hex("#5f5f5f")
	brainFill      = ui.Hex("#fd92ba")
	brainStroke    = ui.Hex("#611836")

	// The fill shows a gradient across the whole track, pink to blue
	// violet, more opaque toward the thumb.
	fillFrom, fillTo         = ui.Hex("#9f5a7a"), ui.Hex("#6958a7")
	fillAlpha        float32 = 0.071
)

// The thumb eases to a level in half a second.
const moveTime = 500 * time.Millisecond

var moveEase = anim.EaseOutExpo

// The label's letters come in one after another, rising, sharpening and
// fading in, as CSS's ease eases them; the old label goes up, blurs and
// fades out at once.
const (
	inStagger = 14500 * time.Microsecond
	inTime    = 320 * time.Millisecond
	outDelay  = 25 * time.Millisecond
	outTime   = 355 * time.Millisecond

	inY, inBlur   = 3.0, 1.8
	outY, outBlur = 3.4, 1.87
)

var labelEase = anim.Ease

// Slider 是卡片的状态：选中的档位、还在运动的动画。
type Slider struct {
	value float64 // the level SliderBase sets, 0 to 5

	level   int       // the level the thumb goes to
	from    float32   // where the thumb was when it set off
	movedAt time.Time // when it set off

	prev      int       // the level the label showed before
	changedAt time.Time // when the label changed

	galaxyAt, galaxyLeft time.Time // when the slider went Galaxy, and left it

	sheet *canvas.TextSheet // the label's letters, for blurring them

	// While the pointer holds the thumb, it stays under the pointer, at
	// held levels, and the level is the nearest; let go, it eases there.
	holding bool
	held    float32
}

// New 造一个停在第 0 档（Low）的滑块。
//
// 注意：它内部要建字表，而字表用 ui.Render 离屏把字形先栅格化一遍（模糊滤镜要
// 用）。ui.Render 自己要跑一帧，**不能在窗口（或 Tester）的一帧里调用**，所以
// 页面必须在进入帧之前就把它造好（见 App.effortPage 与 newApp 的注释）。
func New() *Slider {
	return &Slider{sheet: canvas.NewTextSheet(titleFont, levels...)}
}

var (
	titleFont = ui.Font{Family: "system-ui", Size: titleSize, Antialiased: true}
	labelFont = ui.Font{Family: "system-ui", Size: labelSize, Antialiased: true}
)

// Level 当前档位名（"Low"…"Galaxy"）。
func (s *Slider) Level() string { return levels[s.level] }

// LevelIndex 当前档位下标，0…Galaxy。
func (s *Slider) LevelIndex() int { return s.level }

// thumb returns where the thumb is at t, in levels, and whether it is
// still moving.
func (s *Slider) thumb(t time.Time) (float32, bool) {
	if s.holding {
		return s.held, false
	}
	// 用 anim.Progress：还没开始（零时刻）算作早就结束，省一次判空。
	k := anim.Progress(t, s.movedAt, moveTime)
	if k >= 1 {
		return float32(s.level), false
	}
	return anim.Tween(moveEase, s.from, float32(s.level), max(k, 0)), true
}

// SetLevel 在 t 时刻把拇指送到第 l 档（页面上的按钮与键盘都走这里）。
func (s *Slider) SetLevel(l int, t time.Time) {
	if l == s.level {
		return
	}
	s.from, _ = s.thumb(t)
	s.prev, s.changedAt = s.level, t
	switch {
	case l == galaxy:
		s.galaxyAt, s.galaxyLeft = t, time.Time{}
	case s.level == galaxy:
		s.galaxyLeft = t
	}
	s.level, s.movedAt = l, t
	s.value = float64(l)
}

// hold puts the thumb under the pointer, at pos levels, at t.
func (s *Slider) hold(pos float64, t time.Time) {
	s.holding, s.held = true, float32(min(max(pos, 0), galaxy))
	s.SetLevel(int(math.Round(float64(s.held))), t)
}

// release lets the thumb go at t, to ease from where it was held to its
// level.
func (s *Slider) release(t time.Time) {
	if s.holding {
		s.holding = false
		s.from, s.movedAt = s.held, t
	}
}

// View 把卡片画进当前容器：画布铺满容器，卡片在画布正中。
//
// 为什么中间要套一层「宽度钉死为 cardW」的容器
// --------------------------------------------
// Draw 回调拿到的是**页面绝对坐标**，而 Absolute 定位的子元素（下面的
// SliderBase）是相对**父容器**的 —— 两者基准不同，想对上就得知道容器在页面上
// 的位置，而 Context 只给得出整窗尺寸（c.Size）。把 SliderBase 放进一个宽度
// 固定为 cardW 的小容器里，它的坐标就成了常数（trackX、cardTop+trackY），
// 与页面位置无关；水平居中交给两侧各占一半剩余空间的 Spacer。
func (s *Slider) View(c *ui.Context) {
	// 示例自己是暗色调的，不跟主题：容器整块铺成页面的底色，卡片（连同阴影）
	// 才浮在上面。铺满矩形而不是画圆角 —— 圆角与裁剪交给页面那一层。
	ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, pageColor, 0)
	}).Children(func() {
		ui.Row(c).FillWidth().Children(func() {
			ui.Spacer(c).Grow(1)
			ui.Box(c).Width(cardW).Height(CanvasH).Children(func() {
				ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
					if s.drawCard(p, r.X, r.Y+cardTop, p.Now()) {
						p.AnimationFrame()
					}
				})
				// The slider takes the keys, as assistive technology sees
				// it, and the pointer across the track; its content box
				// spans the thumb's travel. A drag moves the thumb with the
				// pointer rather than from level to level.
				sb := ui.SliderBase(c, &s.value, 0, galaxy).Step(1).FocusRing(false).
					Label("脑力投入").Absolute().
					Left(trackX).Top(cardTop+trackY-6).Size(trackW, trackH+12).
					PaddingX(stopX - trackX)
				t := c.Now()
				if sb.Changed() && !s.holding {
					s.SetLevel(int(math.Round(s.value)), t)
				}
				if _, _, held := sb.Dragged(); held {
					x, _, _ := sb.PointerPosition()
					s.hold(float64(x-(stopX-trackX))/stopGap, t)
				} else {
					s.release(t)
				}
			})
			ui.Spacer(c).Grow(1)
		})
	})
}

// drawCard draws the card with its top left corner at (ox, oy), as it is
// at t, and reports whether it is still animating.
func (s *Slider) drawCard(p *ui.Painter, ox, oy float32, t time.Time) bool {
	pos, moving := s.thumb(t)
	g := s.galaxyLevel(t)
	secs := t.Sub(s.galaxyAt).Seconds()
	if g > 0 {
		moving = true
	}
	dx, dy := s.shakeAt(t)
	ox, oy = ox+dx, oy+dy

	// The card, as CSS rounds it, with a 1 DIP border, on Tailwind's
	// shadow-lg.
	card := ui.Rect{X: ox, Y: oy, W: cardW, H: cardH}
	p.Shadow(card, cardRadius, 0, 10, 15, -3, ui.RGBA(0, 0, 0, 0.1))
	p.Shadow(card, cardRadius, 0, 4, 6, -4, ui.RGBA(0, 0, 0, 0.1))
	var path ui.Path
	canvas.RoundRect(&path, ox, oy, cardW, cardH, cardRadius)
	p.FillPath(&path, borderColor)
	path = ui.Path{}
	canvas.RoundRect(&path, ox+1, oy+1, cardW-2, cardH-2, cardRadius-1)
	p.FillPath(&path, cardColor)

	canvas.Kerned("Effort", titleFont).Draw(p, ox+effortX, oy+titleBaseline, titleColor)
	if s.drawLabel(p, ox+valueX, oy+titleBaseline, t) {
		moving = true
	}
	drawHelp(p, ox+helpX, oy+helpY, titleColor)
	canvas.Kerned("Faster", labelFont).Draw(p, ox+fasterX, oy+labelBaseline, labelColor)
	smarter := canvas.Kerned("Smarter", labelFont)
	smarter.Draw(p, ox+smarterRight-smarter.Width(), oy+labelBaseline, labelColor)

	// The track, and the fill up to the thumb.
	path = ui.Path{}
	canvas.RoundRect(&path, ox+trackX, oy+trackY, trackW, trackH, trackRadius)
	p.FillPath(&path, trackColor)
	thumbX := float32(stopX + stopGap*pos)
	if end := thumbX - fillInset; end > trackX {
		w := end - trackX
		path = ui.Path{}
		canvas.RoundRect(&path, ox+trackX, oy+trackY, w, trackH, min(trackRadius, w/2))
		p.FillPathGradient(&path, ui.LinearGradient{From: fillFrom, To: fillTo, Angle: 90, End: trackW / w, Oklab: true})
		p.FillPathGradient(&path, ui.LinearGradient{From: trackColor.Alpha(1 - fillAlpha), To: trackColor.Alpha(0), Angle: 90})
	}
	gt := g
	if s.galaxyLeft.IsZero() && !s.galaxyAt.IsZero() {
		gt = min(float32(t.Sub(s.galaxyAt))/float32(trackIn), 1)
	}
	drawGalaxyTrack(p, ox, oy, thumbX, gt, secs)

	// The ticks.
	for i := range len(levels) {
		x := ox + tickX + tickGap*float32(i)
		if i == 1 {
			var bar ui.Path
			canvas.RoundRect(&bar, x-tickSize/2, oy+trackY+trackH/2-barH/2, tickSize, barH, tickSize/2)
			p.FillPath(&bar, barColor.Mix(ui.Hex("#c8c4dc"), g))
			continue
		}
		var dot ui.Path
		dot.Circle(x, oy+trackY+trackH/2, tickSize/2)
		p.FillPath(&dot, dotColor.Alpha(1-g))
	}
	canvas.Kerned("Recommended", labelFont).Draw(p, ox+recommendedX, oy+recommendedBaseline, recommendColor)

	// The brain, and in Galaxy its rays and glow behind it, the burst
	// and the flash over it. In Galaxy the brain grows by a third, a
	// little past it at first.
	grow := float32(1 + (galaxyBrain/brainSize-1)*anim.At(anim.BackOut, g))
	bx, by := ox+thumbX, oy+brainY+(galaxyBrainY-brainY)*g
	if g > 0 {
		drawRays(p, bx, by, g, secs)
		canvas.Glow(p, bx, by, 34, ui.Hex("#7c74ff"), 0.4*g)
	}
	if g < 1 {
		drawBrain(p, bx, by, brainSize*grow, brainFill, brainStroke)
	}
	drawGalaxyBrain(p, bx, by, brainSize*grow, g)
	if g > 0 {
		drawSparkles(p, bx, by, g, secs)
	}
	if s.galaxyLeft.IsZero() && !s.galaxyAt.IsZero() {
		since := t.Sub(s.galaxyAt)
		drawBurst(p, bx, by, float32(since)/float32(burstTime))
		if k := float32(since) / float32(flashTime); k < 1 {
			canvas.Glow(p, bx, by, 44, ui.Hex("#f4f2ff"), 0.6*float32(math.Pow(float64(1-k), 1.6)))
		}
	}
	return moving
}

// drawLabel draws the level's label, and the last one while it goes, with
// the pen at x on the baseline at y, and reports whether they move.
func (s *Slider) drawLabel(p *ui.Painter, x, y float32, t time.Time) bool {
	since := t.Sub(s.changedAt)
	if s.changedAt.IsZero() || since >= outDelay+outTime && since >= inStagger*time.Duration(len(levels[s.level])-1)+inTime {
		canvas.Letters(levels[s.level], titleFont).Draw(p, x, y, labelColorOf(levels[s.level]))
		return false
	}
	// text draws str with its pen at x in color c, k opaque, dy below the
	// baseline, blurred by blur DIPs (the sheet draws it sharp when the
	// blur is under half a pixel).
	text := func(str string, glyphs []ui.Glyph, x float32, c ui.Color, k, dy, blur float32) {
		if k <= 0 {
			return
		}
		s.sheet.DrawGlyphs(p, str, glyphs, x, y+dy, blur, c.Alpha(k))
	}
	// The old label goes as a whole, kerned, as a text of its own.
	old := levels[s.prev]
	k := anim.At(labelEase, anim.Progress(t, s.changedAt.Add(outDelay), outTime))
	text(old, ui.Shape(old, titleFont), x, labelColorOf(old), 1-k, -outY*k, outBlur*k)
	// The new one comes letter by letter.
	cur := canvas.Letters(levels[s.level], titleFont)
	c := labelColorOf(levels[s.level])
	for i, r := range []rune(levels[s.level]) {
		at := s.changedAt.Add(inStagger * time.Duration(i))
		k := anim.At(labelEase, anim.Progress(t, at, inTime))
		text(string(r), cur.Glyphs(i), x+cur.Advance(i), c, k, inY*(1-k), inBlur*(1-k))
	}
	return true
}

// labelColorOf returns the color of a level's label: Galaxy's is sky blue.
func labelColorOf(label string) ui.Color {
	if label == levels[galaxy] {
		return galaxyLabel
	}
	return valueColor
}
