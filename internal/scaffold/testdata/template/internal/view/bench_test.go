package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 性能基线：这两条不是「必须多快」的断言，而是给优化前后一个可复现的对照。
//
//	go test ./internal/view -run '^$' -bench 'BenchmarkFrame|BenchmarkNavLookup' -benchmem
//
// 注意：BenchmarkFrame 走的是 CPU 光栅化（离屏），真机 GPU 下渲染部分会低不少；
// 它的价值是「同一个基准上比较前后差异」，而不是绝对值。参考量级（i5-12600KF,
// 1100×720）：约 11ms/帧、4.4MB/帧，其中输出位图本身就占 3.2MB。
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

// BenchmarkNavLookup 菜单/页头每帧要查十几次注册表：这里必须是零分配。
func BenchmarkNavLookup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := pageByID("task"); !ok {
			b.Fatal("task 页面未注册")
		}
	}
}
