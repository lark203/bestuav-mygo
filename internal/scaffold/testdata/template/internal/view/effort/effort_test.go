package effort

import (
	"image"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/imgtest"
)

// 离屏画布：宽度给宽一点（验证居中），高度就是 View 认的画布高。
const (
	testW = 640
	testH = CanvasH
)

// cardAt 卡片在离屏画布里的左上角：水平居中、垂直 cardTop。
func cardAt() (float32, float32) {
	return float32(math.Round(float64(testW-cardW) / 2)), cardTop
}

// TestCardCentered 验证卡片在画布里水平居中、竖直也基本居中（cardTop 取整）。
func TestCardCentered(t *testing.T) {
	ox, oy := cardAt()
	if want := float32(testW-cardW) / 2; ox != want {
		t.Errorf("卡片左边应在 %v，实际 %v", want, ox)
	}
	if d := math.Abs(float64(oy) + cardH/2 - CanvasH/2); d > 0.5 {
		t.Errorf("卡片竖直偏离中线 %.2f DIP（cardTop=%v）", d, oy)
	}
}

// TestSliderDrag 验证拖动：拇指跟着指针走、档位取最近的一档，松手后缓动到那一档。
func TestSliderDrag(t *testing.T) {
	s := New()
	tt := ui.NewTester(s.View, testW, testH)
	now := tt.Now()
	ox, oy := cardAt()
	at := func(pos float32) (float32, float32) { return ox + stopX + stopGap*pos, oy + brainY }

	tt.Press(at(0))
	tt.Move(at(1.3))
	if pos, _ := s.thumb(now); math.Abs(float64(pos)-1.3) > 0.01 || s.level != 1 {
		t.Errorf("拖到 1.3：拇指在 %.2f，档位 %d", pos, s.level)
	}
	tt.Move(at(2.7))
	if pos, _ := s.thumb(now); math.Abs(float64(pos)-2.7) > 0.01 || s.level != 3 || Levels[s.level] != "Extra" {
		t.Errorf("拖到 2.7：拇指在 %.2f，档位 %d", pos, s.level)
	}
	tt.Release(at(2.7))
	if pos, moving := s.thumb(now); math.Abs(float64(pos)-2.7) > 0.01 || !moving {
		t.Errorf("在 2.7 松手：拇指在 %.2f，仍在动 %v", pos, moving)
	}
	now = now.Add(moveTime)
	if pos, moving := s.thumb(now); pos != 3 || moving {
		t.Errorf("缓动结束后拇指应在第 3 档，实际 %.2f，仍在动 %v", pos, moving)
	}

	tt.Key(0, ui.KeyRight)
	if s.level != 4 {
		t.Errorf("右方向键走到第 %d 档，应为 4", s.level)
	}
	tt.Key(0, ui.KeyHome)
	if s.level != 0 {
		t.Errorf("Home 走到第 %d 档，应为 0", s.level)
	}
}

// TestGalaxy 验证 End 进 Galaxy，特效随之进来、离开时再退回去。
func TestGalaxy(t *testing.T) {
	s := New()
	tt := ui.NewTester(s.View, testW, testH)
	ox, oy := cardAt()
	tt.ClickAt(ox+stopX, oy+brainY)
	tt.Key(0, ui.KeyEnd)
	if g := s.galaxyLevel(tt.Now()); s.level != Galaxy || g != 0 {
		t.Fatalf("End 之后档位 %d，Galaxy 进度 %.2f", s.level, g)
	}
	tt.Advance(time.Second)
	if g := s.galaxyLevel(tt.Now()); g != 1 {
		t.Errorf("进 Galaxy 一秒后进度 %.2f，应为 1", g)
	}
	tt.Key(0, ui.KeyLeft)
	tt.Advance(galaxyOut)
	if g := s.galaxyLevel(tt.Now()); s.level != 4 || g != 0 {
		t.Errorf("离开 Galaxy 后档位 %d，进度 %.2f；应为第 4 档、0", s.level, g)
	}
}

// TestWidensWithCanvas 画布变宽后卡片仍居中：换个尺寸再拖大脑，档位要跟着走。
//
// 居中靠两侧的 Spacer，不靠编译期算好的偏移 —— 这条测试就是防止哪天有人把
// 它改回「按 c.Size() 算」。位置错了的话，Press 根本落不到滑块上，档位不动。
func TestWidensWithCanvas(t *testing.T) {
	s := New()
	tt := ui.NewTester(s.View, 480, CanvasH)
	tt.SetSize(900, CanvasH)
	tt.Frame()
	ox := float32(900-cardW) / 2
	tt.Press(ox+stopX, cardTop+brainY)
	tt.Move(ox+stopX+stopGap*3, cardTop+brainY)
	if s.level != 3 {
		t.Errorf("宽度 900 时拖到第 3 档位置，档位却是 %d —— 卡片大概没居中", s.level)
	}
	tt.Release(ox+stopX+stopGap*3, cardTop+brainY)
}

// TestDrawsCardAndBrain 直接看像素：卡片之外是示例的暗底、之内是卡片底色，
// 大脑在 Low 档是粉的，进 Galaxy 换成蓝紫渐变。
//
// 卡片是自绘的，页面上没有任何文本节点可供断言，只能这么验 —— 否则「画歪了」
// 或「压根没画」在离屏测试里完全看不出来。
func TestDrawsCardAndBrain(t *testing.T) {
	s := New()
	tt := ui.NewTester(s.View, testW, testH)
	ox, oy := cardAt()
	// 大脑跟着拇指走：第 0 档在 stopX，Galaxy 在 stopX + stopGap*5。
	at := func(level float32) image.Rectangle {
		cx, cy := int(ox+stopX+stopGap*level), int(oy+brainY)
		return image.Rect(cx-18, cy-18, cx+18, cy+18)
	}
	brain := at(0)

	img := tt.Image()
	if got := imgtest.At(img, 2, 2); !imgtest.Near(got, pageColor, 6) {
		t.Errorf("画布角落应是示例的暗底 %v，实际 %v", pageColor, got)
	}
	if got := imgtest.At(img, int(ox)+150, int(oy)+8); !imgtest.Near(got, cardColor, 8) {
		t.Errorf("卡片内（标题右侧的空白处）应是卡片底色 %v，实际 %v", cardColor, got)
	}
	if n := imgtest.CountNear(img, brain, brainFill, 24); n < 20 {
		t.Errorf("第 0 档时大脑区域应有成片的粉色，实际只有 %d 个像素", n)
	}

	// 推到 Galaxy：大脑炸开、换成蓝紫渐变（galaxyFrom → galaxyTo）。
	tt.ClickAt(ox+stopX, oy+brainY)
	tt.Key(0, ui.KeyEnd)
	tt.Advance(time.Second)
	img = tt.Image()
	brain = at(galaxy)
	if n := imgtest.CountNear(img, brain, brainFill, 24); n > 40 {
		t.Errorf("进 Galaxy 后大脑不该还是粉的，实际还有 %d 个像素", n)
	}
	if n := imgtest.CountNear(img, brain, galaxyFrom, 40) + imgtest.CountNear(img, brain, galaxyTo, 40); n < 20 {
		t.Errorf("进 Galaxy 后大脑应是蓝紫渐变，实际只有 %d 个接近的像素", n)
	}
}

// TestSetLevelFromPage 验证页面按钮那条路径：直接从外部跳档（不走指针）。
func TestSetLevelFromPage(t *testing.T) {
	s := New()
	now := time.Now()
	s.SetLevel(1, now)
	if s.Level() != "Medium" || s.value != 1 {
		t.Fatalf("跳到第 1 档后：%s，value %.1f", s.Level(), s.value)
	}
	// 跳到 Galaxy 再跳回来：galaxyLeft 要记上，特效才会退场而不是立刻消失。
	s.SetLevel(Galaxy, now.Add(time.Second))
	if s.galaxyLeft.IsZero() == false || s.galaxyAt.IsZero() {
		t.Fatalf("进 Galaxy 后 galaxyAt/galaxyLeft 不对：%v / %v", s.galaxyAt, s.galaxyLeft)
	}
	s.SetLevel(4, now.Add(2*time.Second))
	if s.galaxyLeft.IsZero() {
		t.Error("离开 Galaxy 应记下 galaxyLeft，否则特效不淡出")
	}
	if g := s.galaxyLevel(now.Add(2 * time.Second)); g != 1 {
		t.Errorf("刚离开的一瞬间进度应还是 1，实际 %.2f", g)
	}
}
