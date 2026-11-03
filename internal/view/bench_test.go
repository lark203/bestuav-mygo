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

// BenchmarkSteadyFrameSettings 稳态帧（设置页，默认「常规」分组）。
//
// ⚠️ 量这一页必须先关掉「更新检查中」状态，否则测到的是**动画帧**：
//
//	updateChecking=true  → 2.16ms / 1060 allocs
//	updateChecking=false → 104µs  /   46 allocs
//
// 原因是检查中时标题栏会画一个呼吸点（kit.BreathingDot），动画会让框架每帧全量
// 重绘；而静态帧走的是增量渲染，几乎不花时间。更要命的是单测里没有事件循环，
// 检查结果的回调（mygo.RunOnMain）会被丢弃，`updateChecking` 就永远停在 true——
// 于是「设置页很慢」这个结论完全是测量假象（这一条踩过两次，见
// docs/20261104_复测发现与修复.md）。
func BenchmarkSteadyFrameSettings(b *testing.B) {
	a := newTestApp()
	a.page = "settings"
	tt := ui.NewTester(a.View, 1100, 720)
	// 第一帧会触发 onEnter（进设置页即后台检查更新），之后把状态压回「静态」。
	tt.Frame()
	a.updateChecking = false
	tt.Frame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 每轮都挡住「检查中」：一旦为真，这一帧就变成动画帧（差 20 倍），
		// 平均值会随首帧残留与否乱跳。
		a.updateChecking = false
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
