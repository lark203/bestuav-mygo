package view

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
)

// 本文件测笔记页的**工具条排版**：按钮多、窗口一窄就容易挤坏，而挤坏在离屏
// 测试里不会报错（只是悄悄换行），所以要把几何断言写下来。

// noteTester 渲染笔记页（笔记落到临时目录，不读开发机上的真实笔记）。
func noteTester(t *testing.T, w, h int) *ui.Tester {
	t.Helper()
	a := newTestApp()
	a.notePath = filepath.Join(t.TempDir(), "notes.json")
	a.page = "official-notes"
	tt := ui.NewTester(a.View, w, h)
	tt.Frame()
	return tt
}

// boxOf 取元素的位置；找不到直接失败（说明工具条改没了）。
func boxOf(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("工具条上找不到 %q", label)
	}
	return r
}

// sameRow 一组元素是否都在同一水平行（文本顶对齐即可，行内元素高度略有差异）。
func sameRow(t *testing.T, tt *ui.Tester, labels []string) float32 {
	t.Helper()
	y := boxOf(t, tt, labels[0]).Y
	for _, l := range labels[1:] {
		if got := boxOf(t, tt, l).Y; got-y > 1 || y-got > 1 {
			t.Errorf("%q 与 %q 不在同一行：y=%.1f / %.1f", l, labels[0], got, y)
		}
	}
	return y
}

// TestNoteToolbarTwoRows 工具条分两行：行内格式一行、块级格式一行。
//
// 13 个按钮塞进一行时，末尾的「{} / 复制 Markdown」会被挤到第二行的**行首**
// （孤零零一个，看起来像排版错误）；分两行后每行都宽松，也不再依赖窗口宽度。
func TestNoteToolbarTwoRows(t *testing.T) {
	// 多种窗口宽度都成立：窄窗口正是原来挤坏的场景（右列随窗口变窄）。
	for _, w := range []int{1000, 1100, 1280, 1600} {
		t.Run(itoa(w), func(t *testing.T) { noteToolbarRows(t, w) })
	}
}

func noteToolbarRows(t *testing.T, w int) {
	tt := noteTester(t, w, 720)

	inline := sameRow(t, tt, []string{"B", "I", "U", "S", "</>", "链接"})
	block := sameRow(t, tt, []string{"•", "1.", "☐", "❝", "{}"})
	if block-inline < 10 {
		t.Errorf("两行工具条的行距太小：%.1f → %.1f", inline, block)
	}
	// 导出动作在块级那行，且顶到行尾（不是跟在「{}」后面）。
	export := boxOf(t, tt, "复制 Markdown")
	if export.Y-boxOf(t, tt, "{}").Y > 1 {
		t.Errorf("「复制 Markdown」应与块级按钮同一行，实际 y=%.1f vs %.1f",
			export.Y, boxOf(t, tt, "{}").Y)
	}
	last := boxOf(t, tt, "{}")
	if export.X-last.X < 100 {
		t.Errorf("「复制 Markdown」应靠右，实际与「{}」只差 %.1f", export.X-last.X)
	}
	// 右边界贴着右列的内边（内容区右内缩 PagePadX），不能溢出窗口。
	if right := export.X + export.W; right > float32(w)-kit.PagePadX {
		t.Errorf("「复制 Markdown」的右边界 %.1f 溢出内容区（%.1f）", right, float32(w)-kit.PagePadX)
	}
}
