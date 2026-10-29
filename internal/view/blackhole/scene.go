// Package blackhole 是「黑洞与吸积盘」的 GPU 着色器演示。
//
// 移植自 mygo 官方示例 examples/blackhole（MIT License）：
//   - blackhole.kage：每个像素的光线被反向追踪穿过引力场（含多普勒增亮与爱因斯坦环）
//   - bloom.kage    ：把画面里最亮的部分模糊成一层辉光
//   - post.kage     ：合成上面两遍，加镜头色散与胶片颗粒
//
// 相对原示例的改动：
//   - 原示例占满整个窗口；这里作为页面里的一个「画布区域」渲染（见 view 包的
//     page_blackhole.go），因此去掉了窗口级设置（标题栏隐藏、窗口背景色）；
//   - 覆盖文字改为中文，并加了一行「无 GPU 时」的说明；
//   - 其余逻辑（相机、三遍绘制、自适应分辨率、拖拽/滚轮输入）保持原样。
package blackhole

import (
	_ "embed"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

var (
	//go:embed blackhole.kage
	holeSource string
	//go:embed bloom.kage
	bloomSource string
	//go:embed post.kage
	postSource string

	hole  = shader.MustCompile(holeSource)
	bloom = shader.MustCompile(bloomSource)
	post  = shader.MustCompile(postSource)
)

const (
	// 摄像机的距离（以黑洞半径为单位）：最近、默认、最远（着色器的逃逸范围是 42）；
	// 以及仰角的上下界（不到极点）。
	near, rest, far = 8, 22, 36
	maxPitch        = 1.4
	// orbitSpeed 是自转的角速度（弧度/秒），settle 是「甩出去」后衰减到自转的速度。
	orbitSpeed, settle = 0.025, 1.2
	// 场景渲染分辨率是窗口像素的 quality 倍，最低 minQuality：
	// 帧时间超过 slowFrame 就降一档，持续快于 fastFrame 再升回去。辉光层是它的 1/glowScale。
	minQuality, qualityStep = 0.5, 0.125
	slowFrame, fastFrame    = 1.0 / 50, 1.0 / 90
	glowScale               = 4
)

var (
	white = ui.RGBA(255, 255, 255, 0.92)
	muted = ui.RGBA(255, 255, 255, 0.5)
)

// Scene 是这次演示的全部状态（相机、时间、指针拖拽、两遍绘制的画布与分辨率自适应）。
type Scene struct {
	cam camera
	// time 是吸积盘的时钟（系统要求减弱动效时它会停住），last 是上一帧的时间。
	time float64
	last time.Time
	// 指针拖拽：是否按下、上一次的位置与时间；touched 表示用户动过相机，提示会淡出。
	down         bool
	lastX, lastY float32
	lastMove     time.Time
	touched      bool
	hint         float32
	// 两遍绘制的画布：场景（窗口像素的 quality 倍）与它的辉光；
	// frame 是平滑后的帧时长，slow/fast 累计「太慢/太快」的时长。
	sceneCanvas, glowCanvas *ui.Canvas
	quality                 float64
	frame, slow, fast       float64
	// gpu 是本帧是否有硬件加速（nil 表示还不知道，即一帧都没画过）。
	// 着色器只有 GPU 能算，没有它时右上角的提示会换成一句说明。
	gpu *bool

	values                                           *shader.Values
	timeVar, eyeVar, rightVar, upVar, fwdVar, focVar shader.Uniform
	postValues                                       *shader.Values
	postTime                                         shader.Uniform
}

// New 创建演示状态（纯内存，不依赖 GPU：着色器在包初始化时已编译成 GPU 程序源码）。
func New() *Scene {
	return &Scene{
		cam:     camera{yaw: 0.6, pitch: 0.1, dist: rest, spinYaw: orbitSpeed},
		hint:    1,
		quality: 1, frame: 1.0 / 60,
		values:  hole.NewValues(),
		timeVar: hole.Uniform("Time"), eyeVar: hole.Uniform("Eye"), rightVar: hole.Uniform("Right"),
		upVar: hole.Uniform("Up"), fwdVar: hole.Uniform("Forward"), focVar: hole.Uniform("Focal"),
		postValues: post.NewValues(), postTime: post.Uniform("Time"),
	}
}

// View 在给定的上下文里铺满可用区域绘制黑洞，并叠上标题与操作提示。
func (s *Scene) View(c *ui.Context) {
	reduce := c.Preferences().ReduceMotion
	root := ui.Row(c).Fill().AlignItems(ui.End).Padding(0, 20, 18, 20).Background(ui.RGB(0, 0, 0))
	if s.down {
		root.Cursor(ui.CursorGrabbing)
	} else {
		root.Cursor(ui.CursorGrab)
	}
	root.Draw(func(p *ui.Painter, r ui.Rect) { s.draw(p, r, reduce) })
	root.HandleInput(s.input)
	root.Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, "Gargantua").FontSize(20).FontWeight(600).TextColor(white)
			ui.Text(c, "黑洞与吸积盘：光线在引力场中被逐像素反向追踪").
				FontSize(12).TextColor(muted)
		})
		ui.Spacer(c).Grow(1)
		if s.gpu != nil && !*s.gpu {
			// 没有 GPU 时交互没有意义，把操作提示换成一句说明。
			ui.Text(c, "当前环境没有硬件加速，需要 GPU 才能渲染").FontSize(12).TextColor(muted)
		} else {
			ui.Text(c, "拖拽旋转 · 滚轮缩放").FontSize(12).TextColor(muted).Opacity(s.hint)
		}
	})
}

// draw 把场景推进到本帧时间，并分三遍绘制。
func (s *Scene) draw(p *ui.Painter, r ui.Rect, reduce bool) {
	now := p.Now()
	dt := 0.0
	if !s.last.IsZero() {
		dt = min(now.Sub(s.last).Seconds(), 0.1)
	}
	s.last = now
	if !reduce {
		s.time += dt
	}
	s.step(dt, reduce)

	// 有没有硬件加速只有画的时候才知道；记下来，下一帧的布局据此换文案。
	acc := p.Accelerated()
	s.gpu = &acc
	if !acc {
		// 没有 GPU（或离屏渲染）：着色器算不了，直接留空（文案由 View 里的节点显示）。
		return
	}
	scale := float64(p.Scale())
	w := max(1, int(math.Round(float64(r.W)*scale*s.quality)))
	h := max(1, int(math.Round(float64(r.H)*scale*s.quality)))
	s.sceneCanvas = sized(s.sceneCanvas, w, h)
	s.glowCanvas = sized(s.glowCanvas, max(1, w/glowScale), max(1, h/glowScale))

	eye, right, up, forward := s.cam.axes()
	s.values.Float(s.timeVar, float32(s.time)).Float(s.focVar, focal)
	for _, v := range [...]struct {
		u shader.Uniform
		d vec3
	}{{s.eyeVar, eye}, {s.rightVar, right}, {s.upVar, up}, {s.fwdVar, forward}} {
		x := v.d.f32()
		s.values.Vec3(v.u, x[0], x[1], x[2])
	}
	s.sceneCanvas.Draw(p, func(cp *ui.Painter) {
		hole.Draw(cp, ui.Rect{W: float32(w), H: float32(h)}, shader.Options{Values: s.values})
	})
	gw, gh := s.glowCanvas.Size()
	s.glowCanvas.Draw(p, func(cp *ui.Painter) {
		bloom.Draw(cp, ui.Rect{W: float32(gw), H: float32(gh)},
			shader.Options{Images: [4]*ui.Bitmap{s.sceneCanvas.Bitmap()}})
	})
	post.Draw(p, r, shader.Options{
		Values: s.postValues.Float(s.postTime, float32(s.time)),
		Images: [4]*ui.Bitmap{s.sceneCanvas.Bitmap(), s.glowCanvas.Bitmap()},
	})
	if !reduce || s.down || s.cam.spinning() || (s.touched && s.hint > 0) {
		p.AnimationFrame()
	}
}

// sized 返回尺寸匹配的画布（不匹配就新建）。
func sized(cv *ui.Canvas, w, h int) *ui.Canvas {
	if cv != nil {
		if cw, ch := cv.Size(); cw == w && ch == h {
			return cv
		}
	}
	return ui.NewCanvas(w, h)
}

// step 推进相机、淡出提示，并按帧速调整渲染分辨率。
func (s *Scene) step(dt float64, reduce bool) {
	cam := &s.cam
	if !s.down {
		goal := orbitSpeed
		if reduce {
			goal = 0
		}
		k := 1 - math.Exp(-dt*settle)
		cam.spinYaw += (goal - cam.spinYaw) * k
		cam.spinPitch -= cam.spinPitch * min(1, 2*k)
		cam.yaw += cam.spinYaw * dt
		cam.pitch = clamp(cam.pitch+cam.spinPitch*dt, -maxPitch, maxPitch)
	}
	if s.touched {
		s.hint = max(0, s.hint-float32(dt)/0.6)
	}

	if dt <= 0 {
		return
	}
	s.frame += (dt - s.frame) * 0.1
	if s.frame > slowFrame {
		s.slow += dt
	} else {
		s.slow = 0
	}
	if s.frame < fastFrame {
		s.fast += dt
	} else {
		s.fast = 0
	}
	switch {
	case s.slow > 0.5 && s.quality > minQuality:
		s.quality = max(minQuality, s.quality-qualityStep)
		s.slow, s.frame = 0, fastFrame
	case s.fast > 3 && s.quality < 1:
		s.quality = min(1, s.quality+qualityStep)
		s.fast = 0
	}
}

// input 处理拖拽与滚轮：拖拽转动相机，甩出去后保持惯性、逐渐衰减到自转。
func (s *Scene) input(ev ui.InputEvent) bool {
	cam := &s.cam
	now := time.Now()
	switch ev.Kind {
	case ui.InputPointerDown:
		if ev.Button != 0 {
			return false
		}
		s.down, s.touched = true, true
		s.lastX, s.lastY, s.lastMove = ev.X, ev.Y, now
		cam.spinYaw, cam.spinPitch = 0, 0
		return true
	case ui.InputPointerMove:
		if !s.down {
			return false
		}
		dt := max(now.Sub(s.lastMove).Seconds(), 1.0/240)
		dyaw, dpitch := -float64(ev.X-s.lastX)*0.005, float64(ev.Y-s.lastY)*0.004
		cam.yaw += dyaw
		cam.pitch = clamp(cam.pitch+dpitch, -maxPitch, maxPitch)
		// 甩出去的速度：用最近几次移动的角速度平滑得到。
		k := min(1, dt/0.06)
		cam.spinYaw += (dyaw/dt - cam.spinYaw) * k
		cam.spinPitch += (dpitch/dt - cam.spinPitch) * k
		s.lastX, s.lastY, s.lastMove = ev.X, ev.Y, now
		return true
	case ui.InputPointerUp:
		if !s.down {
			return false
		}
		s.down = false
		if now.Sub(s.lastMove) > 80*time.Millisecond {
			// 松手前停住了：不产生惯性。
			cam.spinYaw, cam.spinPitch = 0, 0
		}
		return true
	case ui.InputScroll:
		s.touched = true
		cam.dist = clamp(cam.dist*math.Exp(float64(ev.DY)*0.003), near, far)
		return true
	}
	return false
}

// clamp 把 x 夹到 [lo, hi]。
func clamp(x, lo, hi float64) float64 { return max(lo, min(hi, x)) }
