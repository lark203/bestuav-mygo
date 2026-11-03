package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 性能基线：这些不是「必须多快」的断言，而是给优化前后一个可复现的对照。
//
//	go test ./internal/view -run '^$' -bench 'Benchmark' -benchmem
//
// 两种口径务必分清，混用会把结论带偏：
//
//	BenchmarkFrame        冷帧：每轮都新建 Tester（含 3.2MB 输出位图分配）
//	BenchmarkSteadyFrame  稳态帧：复用同一个 Tester，只算「UI 重建 + 绘制」
//
// 参考量级（i5-12600KF 同级，1100×720，本机实测）：冷帧约 3.4ms / 4.1MB，
// 稳态帧约 130µs / 3.5KB。也就是说冷帧的耗时绝大部分是位图分配，**不是** UI 本身；
// 「一帧要 3.4ms」这种说法只对冷启动成立，别拿它评价日常渲染。
//
// BenchmarkFrame 走的是 CPU 光栅化（离屏），真机 GPU 下渲染部分会低不少；
// 它的价值是「同一个基准上比较前后差异」，而不是绝对值。
func BenchmarkFrame(b *testing.B) {
	a := newTestApp()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if img := ui.NewTester(a.View, 1100, 720).Image(); img == nil {
			b.Fatal("no image")
		}
	}
}

// BenchmarkSteadyFrame 稳态帧（首页）：这才是用户平时感受到的那条路径。
//
// 首帧先跑一次让布局/字体度量/页面状态就绪，再重置计时——否则测进去的是初始化。
func BenchmarkSteadyFrame(b *testing.B) {
	a := newTestApp()
	tt := ui.NewTester(a.View, 1100, 720)
	tt.Frame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}

// BenchmarkSteadyFrameSettings 稳态帧（设置页）。
//
// 单独留一条是因为它是全应用最重的一页：实测约 2.6ms/帧（首页 130µs 的 20 倍），
// 且热点全在框架的 commitElement / 文字 shaping —— 也就是「元素数量」驱动。
// 想给设置页做「分区折叠」之类的优化时，用它当基线。
func BenchmarkSteadyFrameSettings(b *testing.B) {
	a := newTestApp()
	a.page = "settings"
	tt := ui.NewTester(a.View, 1100, 720)
	tt.Frame()
	tt.Frame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}

// BenchmarkNavLookup 菜单/页头每帧要查十几次注册表：这里必须是零分配。
func BenchmarkNavLookup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := pageByID("task"); !ok {
			b.Fatal("task 页面未注册")
		}
	}
}
