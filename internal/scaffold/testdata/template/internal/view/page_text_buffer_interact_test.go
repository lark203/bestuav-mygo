package view

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// 这一组用离屏 Tester 真点按钮 / 真打字，验证「点下去之后状态确实变了」。
// 纯函数测试证明不了这一点 —— 按钮回调没接上时前者照样全绿。

// tbOpen 打开大文档页并推进两帧（一帧建、一帧排版完成）。
func tbOpen(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a := newTestApp()
	a.page = "official-text-buffer"
	tt := ui.NewTester(a.View, 1100, 780)
	tt.Frame()
	tt.Frame()
	return a, tt
}

// TestTbClickInsertThenUndo 点「中间插一行」→ 行数 +1；点「撤销」→ 行数回来。
func TestTbClickInsertThenUndo(t *testing.T) {
	a, tt := tbOpen(t)
	before := a.tb.buf.LineCount()

	if err := tt.Click("中间插一行"); err != nil {
		t.Fatalf("点不到「中间插一行」：%v", err)
	}
	tt.Frame()
	if got := a.tb.buf.LineCount(); got != before+1 {
		t.Fatalf("点一次后行数 %d → %d，期望 +1", before, got)
	}
	if len(a.tb.snaps) != 1 {
		t.Errorf("撤销栈深度 = %d，期望 1", len(a.tb.snaps))
	}
	// 状态行要跟着更新（这是用户唯一能看到的反馈）。
	if !tt.HasText("撤销栈 1 步") {
		t.Errorf("状态行没更新，当前文本: %v", tt.Texts())
	}

	if err := tt.Click("撤销"); err != nil {
		t.Fatalf("点不到「撤销」：%v", err)
	}
	tt.Frame()
	if got := a.tb.buf.LineCount(); got != before {
		t.Errorf("撤销后行数 = %d，期望 %d", got, before)
	}
	if !tt.HasText("撤销栈 0 步") {
		t.Errorf("撤销后状态行没更新，当前文本: %v", tt.Texts())
	}
}

// TestTbClickJump 在行号框里打字再点「跳转」，光标要移过去。
func TestTbClickJump(t *testing.T) {
	a, tt := tbOpen(t)

	// 输入框本身没有可见文字，靠 Label 定位。
	if err := tt.Click("跳转到行"); err != nil {
		t.Fatalf("点不到行号输入框：%v", err)
	}
	tt.Type("42")
	tt.Frame()
	if a.tb.jumpTo != "42" {
		t.Fatalf("输入框内容 = %q，期望 42", a.tb.jumpTo)
	}

	if err := tt.Click("跳转"); err != nil {
		t.Fatalf("点不到「跳转」：%v", err)
	}
	tt.Frame()
	if !strings.Contains(a.tb.note, "第 42 行") {
		t.Errorf("操作结果 = %q，期望提到第 42 行", a.tb.note)
	}
	// 待设置的选区在渲染时被消费掉（-1），说明真的下过一次 SetTextSelection。
	if a.tb.pending != -1 {
		t.Errorf("pending 应在渲染时被消费，实际 %d", a.tb.pending)
	}
}

// TestTbClickBenchmark 点「跑一次对比」后要出现结果条。
func TestTbClickBenchmark(t *testing.T) {
	a, tt := tbOpen(t)
	if err := tt.Click("跑一次对比"); err != nil {
		t.Fatalf("点不到「跑一次对比」：%v", err)
	}
	tt.Frame()
	if a.tb.bench == "" {
		t.Fatal("点完没产生结果")
	}
	if !tt.HasText("整串复制") {
		t.Errorf("结果没渲染出来，当前文本: %v", tt.Texts())
	}
}

// TestTbClickDeleteThenRebuild 点「删 100 行」、「重建文档」这两条路径。
func TestTbClickDeleteThenRebuild(t *testing.T) {
	a, tt := tbOpen(t)
	before := a.tb.buf.LineCount()

	if err := tt.Click("删 100 行"); err != nil {
		t.Fatalf("点不到「删 100 行」：%v", err)
	}
	tt.Frame()
	if got := a.tb.buf.LineCount(); got != before-100 {
		t.Fatalf("删除后行数 = %d，期望 %d", got, before-100)
	}

	old := a.tb
	if err := tt.Click("重建文档"); err != nil {
		t.Fatalf("点不到「重建文档」：%v", err)
	}
	tt.Frame()
	if a.tb == old {
		t.Error("点「重建文档」后状态没换新")
	}
	if a.tb.buf.LineCount() != before {
		t.Errorf("重建后行数 = %d，期望 %d", a.tb.buf.LineCount(), before)
	}
}
