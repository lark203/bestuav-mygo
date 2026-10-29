package view

import (
	"testing"

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

	// 滚完必须停下来：还在请求帧的话窗口会永远 invalidation。
	if tt.FramePending() {
		t.Error("数字滚完后不应再请求帧")
	}
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
