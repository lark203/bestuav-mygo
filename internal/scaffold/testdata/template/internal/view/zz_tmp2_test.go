package view

import (
	"testing"

	"bestuav-mygo/internal/settings"
	"github.com/egoist/mygo/ui"
)

// 稳态帧：复用同一个 Tester，排除 Tester 构造与首帧成本。
func BenchmarkTmpSteadyFrame(b *testing.B) {
	a := newApp(settings.Default())
	tt := ui.NewTester(a.View, 1100, 720)
	tt.Frame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}

// 切页帧：每 10 帧换一次页，覆盖 syncPages + 不同页面构建。
func BenchmarkTmpSwitchFrame(b *testing.B) {
	a := newApp(settings.Default())
	tt := ui.NewTester(a.View, 1100, 720)
	ids := []string{"home", "task", "official-blackhole", "fn-todo", "settings"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.page = ids[i%len(ids)]
		tt.Frame()
	}
}

func benchPage(b *testing.B, id string) {
	a := newApp(settings.Default())
	a.page = id
	tt := ui.NewTester(a.View, 1100, 720)
	tt.Frame()
	tt.Frame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}

func BenchmarkTmpPageHome(b *testing.B)     { benchPage(b, "home") }
func BenchmarkTmpPageTask(b *testing.B)     { benchPage(b, "task") }
func BenchmarkTmpPageSettings(b *testing.B) { benchPage(b, "settings") }
func BenchmarkTmpPageCrychic(b *testing.B)  { benchPage(b, "crychic-datetime") }
func BenchmarkTmpPageVlist(b *testing.B)    { benchPage(b, "fn-vlist") }
func BenchmarkTmpPageTimeline(b *testing.B) { benchPage(b, "fn-timeline") }
func BenchmarkTmpPageBlackhole(b *testing.B) { benchPage(b, "official-blackhole") }
