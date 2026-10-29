package kit

import (
	"fmt"
	"image"
	"strconv"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// 本文件只测「动」的部分：数字滚动的中间值与终值、脉冲的出现与退场。
// 断言都用 Tester 的时钟（Advance 推动），不碰 time.Now()。

func testTheme() *ui.Theme { return &ui.Theme{FontSize: 14} }

// number 从渲染出的文本里读回数字（视图里只有一个数字文本）。
func number(t *testing.T, tt *ui.Tester) float64 {
	t.Helper()
	texts := tt.Texts()
	if len(texts) != 1 {
		t.Fatalf("期望只有一个文本，实际 %v", texts)
	}
	v, err := strconv.ParseFloat(texts[0], 64)
	if err != nil {
		t.Fatalf("文本 %q 不是数字", texts[0])
	}
	return v
}

// TestCountUpRollsToTarget 数字从 0 滚到终值：中途严格在两者之间，跑完正好落在终值。
func TestCountUpRollsToTarget(t *testing.T) {
	start := time.Now()
	tt := ui.NewTester(func(c *ui.Context) {
		CountUp(c, 1286, start, CountUpDur, func(v float64) string { return fmt.Sprintf("%.0f", v) })
	}, 200, 60)

	first := number(t, tt)
	if first < 0 || first > 200 {
		t.Errorf("刚起步应接近 0，实际 %v", first)
	}
	tt.Advance(CountUpDur / 2)
	mid := number(t, tt)
	if !(first < mid && mid < 1286) {
		t.Errorf("中途应落在开局与终值之间，实际 %v（开局 %v）", mid, first)
	}
	tt.Advance(CountUpDur)
	if got := number(t, tt); got != 1286 {
		t.Errorf("跑完应正好是终值，实际 %v", got)
	}
}

// TestCountUpZeroStartShowsTarget start 为零值 = 「没安排过动画」，直接显示终值。
//
// 这条的意义：调用方把起点清零（如每次进入页面）时不能先闪一个 0 再滚。
func TestCountUpZeroStartShowsTarget(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		CountUp(c, 37, time.Time{}, CountUpDur, func(v float64) string { return fmt.Sprintf("%.0f", v) })
	}, 200, 60)
	if got := number(t, tt); got != 37 {
		t.Errorf("零值起点应直接显示终值，实际 %v", got)
	}
}

// TestCountUpRespectsReduceMotion 系统要求「减弱动态效果」时不滚动。
func TestCountUpRespectsReduceMotion(t *testing.T) {
	start := time.Now()
	tt := ui.NewTester(func(c *ui.Context) {
		CountUp(c, 1286, start, CountUpDur, func(v float64) string { return fmt.Sprintf("%.0f", v) })
	}, 200, 60)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Advance(time.Millisecond)
	if got := number(t, tt); got != 1286 {
		t.Errorf("减弱动态效果时应直接显示终值，实际 %v", got)
	}
}

// greenBias 整幅图的「偏绿」程度：Σ(G−R)。
//
// 不能用绿色总量：药丸画在白底上，叠加绿光后 G 反而**降低**（#2fbf71 的 G=191
// 比白的 255 小）。而 R 降得更多（47 vs 191），所以 G−R 在绿光处为正、白底处为 0，
// 正好把「绿光」和「白底」分开。
func greenBias(img *image.RGBA) int64 {
	var s int64
	for i := 0; i+3 < len(img.Pix); i += 4 {
		s += int64(img.Pix[i+1]) - int64(img.Pix[i])
	}
	return s
}

// TestPillPulseFadesOutAndStops 脉冲：刚完成时亮一下，1.1 秒后回到常态，且不再请求帧。
func TestPillPulseFadesOutAndStops(t *testing.T) {
	th := testTheme()
	col := ui.Hex("#2fbf71")
	at := time.Now()
	tt := ui.NewTester(func(c *ui.Context) {
		PillPulse(c, th, "已完成", col, at)
	}, 200, 80)

	// 头 12% 是淡入，第一帧还没有光：采样要等它亮起来。
	tt.Advance(PulseDur / 5)
	hot := greenBias(tt.Image())
	tt.Advance(2 * PulseDur)
	cold := greenBias(tt.Image())
	if hot <= cold {
		t.Errorf("刚完成时应该更亮：脉冲 %d，常态 %d", hot, cold)
	}
	// 常态与「从未发生」一致：脉冲退干净，不留残影。
	plain := ui.NewTester(func(c *ui.Context) {
		PillPulse(c, th, "已完成", col, time.Time{})
	}, 200, 80)
	base := greenBias(plain.Image())
	if diff := cold - base; diff > 200 || diff < -200 {
		t.Errorf("脉冲结束后应回到与 kit.Pill 一致的底色，差 %d（脉冲后 %d，常态 %d）", diff, cold, base)
	}
	// 动画必须自己停下来：跑完还在请求帧的话，窗口会永远 invalidation。
	if tt.FramePending() {
		t.Error("脉冲结束后不应再请求帧")
	}
}
