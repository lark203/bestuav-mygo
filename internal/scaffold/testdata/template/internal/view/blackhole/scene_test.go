package blackhole

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestShadersCompile 保证三份 .kage 都能编译（着色器在包初始化时 MustCompile，
// 编译失败会 panic，因此本测试只要能进入函数体即说明着色器没问题）；
// 再取一次后面要用的 uniform —— 名字写错时 Shader.Uniform 会 panic，
// 这样「着色器改了变量名但 Go 侧没跟上」会在测试里暴露，而不是等运行时崩。
func TestShadersCompile(t *testing.T) {
	s := New()
	if s.values == nil || s.postValues == nil {
		t.Fatal("New 没有创建 uniform 值容器")
	}
}

// TestViewWithoutGPU 验证没有硬件加速时会把操作提示换成说明文案
// （离屏渲染 ui.Tester 就是这种环境），而不是只留一块纯黑让人以为卡住了。
func TestViewWithoutGPU(t *testing.T) {
	s := New()
	tt := ui.NewTester(func(c *ui.Context) { s.View(c) }, 800, 430)
	tt.Frame()
	for _, want := range []string{"Gargantua", "没有硬件加速"} {
		if !tt.HasText(want) {
			t.Errorf("缺少文本 %q，当前文本: %v", want, tt.Texts())
		}
	}
	if tt.HasText("拖拽旋转") {
		t.Error("没有 GPU 时不该再显示拖拽提示")
	}
}

// TestAxesAreOrthonormal 验证相机三轴互相垂直且为单位向量：
// 着色器直接拿这三个向量建坐标系，非正交会导致画面被拉伸。
func TestAxesAreOrthonormal(t *testing.T) {
	for _, pitch := range []float64{-1.4, -0.3, 0, 0.5, 1.4} {
		c := camera{yaw: 0.7, pitch: pitch, dist: rest}
		eye, right, up, fwd := c.axes()
		for _, tc := range []struct {
			name string
			v    vec3
		}{{"right", right}, {"up", up}, {"forward", fwd}} {
			if got := math.Sqrt(tc.v.dot(tc.v)); math.Abs(got-1) > 1e-9 {
				t.Errorf("pitch=%v: %s 不是单位向量，长度 %v", pitch, tc.name, got)
			}
		}
		for _, tc := range []struct {
			name string
			a, b vec3
		}{{"right·up", right, up}, {"right·forward", right, fwd}, {"up·forward", up, fwd}} {
			if got := tc.a.dot(tc.b); math.Abs(got) > 1e-9 {
				t.Errorf("pitch=%v: %s 不正交，点积 %v", pitch, tc.name, got)
			}
		}
		// 相机看向黑洞，因此 forward 与 eye 反向。
		if got := fwd.dot(eye.norm()); math.Abs(got+1) > 1e-9 {
			t.Errorf("pitch=%v: forward 应与 eye 反向，点积 %v", pitch, got)
		}
	}
}

// TestScrollClampsDistance 验证滚轮缩放不会把相机推出 [near, far]。
func TestScrollClampsDistance(t *testing.T) {
	s := New()
	for i := 0; i < 200; i++ {
		s.input(ui.InputEvent{Kind: ui.InputScroll, DY: 500})
	}
	if s.cam.dist != far {
		t.Errorf("一直拉远应停在 far=%v，实际 %v", far, s.cam.dist)
	}
	for i := 0; i < 400; i++ {
		s.input(ui.InputEvent{Kind: ui.InputScroll, DY: -500})
	}
	if s.cam.dist != near {
		t.Errorf("一直拉近应停在 near=%v，实际 %v", near, s.cam.dist)
	}
}

// TestDragSpinsThenSettles 验证拖拽会转动相机，松手后惯性衰减到自转速度。
func TestDragSpinsThenSettles(t *testing.T) {
	s := New()
	yaw0 := s.cam.yaw
	s.input(ui.InputEvent{Kind: ui.InputPointerDown, Button: 0, X: 100, Y: 100})
	s.input(ui.InputEvent{Kind: ui.InputPointerMove, Button: 0, X: 160, Y: 100})
	if s.cam.yaw == yaw0 {
		t.Error("拖拽没有转动相机")
	}
	s.input(ui.InputEvent{Kind: ui.InputPointerUp, Button: 0, X: 160, Y: 100})
	if s.down {
		t.Error("松手后 down 仍为真")
	}
	// 推进若干帧，惯性应收敛到自转速度 orbitSpeed，而不是一直加速。
	for i := 0; i < 600; i++ {
		s.step(1.0/60, false)
	}
	if math.Abs(s.cam.spinYaw-orbitSpeed) > 1e-3 {
		t.Errorf("惯性应衰减到自转速度 %v，实际 %v", orbitSpeed, s.cam.spinYaw)
	}
}

// TestQualityAdapts 验证帧速慢时会降分辨率、持续快时再升回去。
func TestQualityAdapts(t *testing.T) {
	s := New()
	s.frame, s.quality = 1.0/60, 1
	for i := 0; i < 120; i++ {
		s.step(1.0/20, false) // 20fps：太慢
	}
	if s.quality >= 1 {
		t.Errorf("持续慢帧应降低分辨率，实际 quality=%v", s.quality)
	}
	if s.quality < minQuality {
		t.Errorf("分辨率不应低于 minQuality=%v，实际 %v", minQuality, s.quality)
	}
	// 升档要「持续快于 fastFrame 满 3 秒」才动一档，从 0.625 回到 1 需要好几档。
	for i := 0; i < 4000; i++ {
		s.step(1.0/240, false) // 240fps：够快
	}
	if s.quality != 1 {
		t.Errorf("持续快帧应恢复满分辨率，实际 quality=%v", s.quality)
	}
}
