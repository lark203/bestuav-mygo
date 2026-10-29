package view

import (
	"image"
	"testing"
	"time"

	"bestuav-mygo/internal/imgtest"
	"bestuav-mygo/internal/ui/kit"
)

// TestHomeKPICountsUp 首页四个 KPI 的数字会滚到终值，并且格式正确（千分位、
// 小数位、单位都在）。
func TestHomeKPICountsUp(t *testing.T) {
	a := newTestApp()
	tt := render(a)

	// 刚进页面时数字还在路上：不该已经是终值。
	if tt.HasText("1,286") {
		t.Error("刚进首页就显示终值，滚动没生效")
	}
	tt.Advance(2 * kit.CountUpDur)
	mustHave(t, tt, "37", "99.2%", "1,286", "1.8s")

	// 滚完必须停在终值上（再走 2 秒仍是这几个数）。
	//
	// 注意这里**不能**断言整页 FramePending：首页的服务状态点在呼吸（循环构件，
	// 没有终点），整页会一直请求帧。要断言「一次性动画自己会停」，得用只有它自
	// 己的视图（见 kit 的 TestCountUp*）；页面上只能看它的结果有没有不再变化。
	tt.Advance(2 * time.Second)
	mustHave(t, tt, "37", "99.2%", "1,286", "1.8s")
}

// TestHomeStatusDotBreathes 服务状态点的呼吸：一次性动画结束后，画面仍在动。
//
// 与上面那条是一组：整页会一直请求帧（呼吸），但动的只有那几个点 —— KPI 数字
// 与完成脉冲都停在自己的终态上。
func TestHomeStatusDotBreathes(t *testing.T) {
	a := newTestApp()
	tt := render(a)
	// 先让一次性动画（KPI 滚动）走完，之后画面上只剩呼吸在动。
	tt.Advance(2 * kit.CountUpDur)
	if !tt.FramePending() {
		t.Error("首页有会呼吸的状态点，整页应持续请求帧")
	}
	lo, hi := int64(1<<60), int64(-1<<60)
	for range 16 {
		v := greenBias(tt.Image())
		lo, hi = min(lo, v), max(hi, v)
		tt.Advance(kit.BreathePeriod / 16)
	}
	// 呼吸一轮的振幅：四个点加起来上千（单点约 2000，相位不同会互相抵消一部分）。
	if hi-lo < 500 {
		t.Errorf("服务状态点应在呼吸，实测整幅波动 %d（%d→%d）", hi-lo, lo, hi)
	}
	// 一直在动，也就一直停在「请求帧」上。
	if !tt.FramePending() {
		t.Error("呼吸没有终点，应持续请求帧")
	}
}

// greenBias 整幅图的「偏绿」程度：Σ(G−R)。绿光处为正、白底处为 0，正好把
// 「有绿光」和「没绿光」分开（同 kit/motion_test.go）。
func greenBias(img *image.RGBA) int64 {
	var s int64
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			c := imgtest.At(img, x, y)
			s += int64(c.G) - int64(c.R)
		}
	}
	return s
}

// TestHomeKPIReRollsOnReturn 切走再切回首页，数字重新滚一遍（onEnter 清零起点）。
func TestHomeKPIReRollsOnReturn(t *testing.T) {
	a := newTestApp()
	tt := render(a)
	tt.Advance(2 * kit.CountUpDur)

	// 切页要用 Frame 强制跑一帧：直接改 a.page 不会请求帧，syncPages 就
	// 不会执行 onEnter（表现为「切回首页但起点没清零」）。
	a.page = "task"
	tt.Frame()
	a.page = "home"
	tt.Frame()
	// 起点被 onEnter 清零、又在帧内重新取，于是重新从 0 附近起步。
	if tt.HasText("1,286") {
		t.Error("切回首页应重新滚动，实际直接是终值")
	}
	tt.Advance(2 * kit.CountUpDur)
	mustHave(t, tt, "1,286")
}

// TestGroupDigits 千分位：符号、短数、刚好三位、跨逗号。
func TestGroupDigits(t *testing.T) {
	cases := map[int]string{
		0: "0", 7: "7", 37: "37", 999: "999", 1000: "1,000",
		1286: "1,286", 12345: "12,345", 1234567: "1,234,567",
		-1286: "-1,286",
	}
	for n, want := range cases {
		if got := groupDigits(n); got != want {
			t.Errorf("groupDigits(%d) = %q，期望 %q", n, got, want)
		}
	}
}

// TestPushTaskRowMarksDoneAt 只有「已完成」的记录带完成时刻（状态列据此脉冲）。
func TestPushTaskRowMarksDoneAt(t *testing.T) {
	a := newTestApp()
	a.tasks = nil
	a.pushTaskRow("跑完的任务", "轻量级", "已完成")
	if a.tasks[0].doneAt.IsZero() {
		t.Error("已完成的记录应记下完成时刻，否则状态列不会脉冲")
	}
	a.pushTaskRow("排队的任务", "重量级", "排队")
	if !a.tasks[0].doneAt.IsZero() {
		t.Error("非完成状态不该有完成时刻")
	}
}
