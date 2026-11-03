package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestTextFieldCaretAtEnd 验证初始光标停在文本末尾（下标按 UTF-16 计，不是字节）。
func TestTextFieldCaretAtEnd(t *testing.T) {
	f := newTextField("abc")
	if got := f.selection.Range; got.Start != 3 || got.End != 3 {
		t.Errorf("光标应在末尾 3，实际 %v", got)
	}
	// 多字节文本：中文一个字 = 1 个 UTF-16 单元，emoji = 2 个。
	f = newTextField("a中😀")
	if want := 1 + 1 + 2; f.selection.Range.Start != want {
		t.Errorf("光标应在 UTF-16 长度 %d，实际 %d", want, f.selection.Range.Start)
	}
}

// TestTextFieldReplaceAtCaret 验证不指定范围时的替换落在光标处，且光标随之后移。
func TestTextFieldReplaceAtCaret(t *testing.T) {
	f := newTextField("abc")
	f.ReplaceText(nil, "X")
	if f.text != "abcX" {
		t.Errorf("文本应为 abcX，实际 %q", f.text)
	}
	if f.selection.Range.Start != 4 || f.selection.Range.End != 4 {
		t.Errorf("光标应移到 4，实际 %v", f.selection.Range)
	}
	// 指定范围时替换该范围。
	f.ReplaceText(&ui.TextInputRange{Start: 0, End: 1}, "Z")
	if f.text != "ZbcX" {
		t.Errorf("文本应为 ZbcX，实际 %q", f.text)
	}
}

// TestTextFieldTextForRange 验证范围取文本：返回的是字节切片，下标却按 UTF-16，
// 二者不一致就会切错（中文 3 字节 / emoji 4 字节但只占 2 个 UTF-16 单元）。
func TestTextFieldTextForRange(t *testing.T) {
	f := newTextField("a中😀b")
	cases := []struct {
		start, end int
		want       string
	}{
		{0, 1, "a"},
		{1, 2, "中"},
		{2, 4, "😀"}, // emoji 占 2 个 UTF-16 单元
		{4, 5, "b"},
		{1, 5, "中😀b"},
	}
	for _, tc := range cases {
		got, r := f.TextForRange(ui.TextInputRange{Start: tc.start, End: tc.end})
		if got != tc.want {
			t.Errorf("范围 %d–%d: 得到 %q，期望 %q", tc.start, tc.end, got, tc.want)
		}
		if r.Start != tc.start || r.End != tc.end {
			t.Errorf("范围 %d–%d: 回传的范围是 %d–%d", tc.start, tc.end, r.Start, r.End)
		}
	}
}

// TestTextFieldMarkedText 验证输入法组字：组字期间替换落在组字区，
// UnmarkText 之后组字区消失、文本留下。
func TestTextFieldMarkedText(t *testing.T) {
	f := newTextField("abc")
	f.SetMarkedText(nil, "中", ui.TextInputRange{})
	if f.text != "abc中" {
		t.Errorf("组字后文本应为 abc中，实际 %q", f.text)
	}
	if _, ok := f.MarkedRange(); !ok {
		t.Error("组字期间应有 marked range")
	}
	// 组字未确认时继续替换，应替换掉正在组的那段而不是往后追加。
	f.ReplaceText(nil, "文")
	if f.text != "abc文" {
		t.Errorf("再次组字应替换掉未确认的部分，实际 %q", f.text)
	}
	f.UnmarkText()
	if _, ok := f.MarkedRange(); ok {
		t.Error("UnmarkText 之后不应还有 marked range")
	}
	if f.text != "abc文" {
		t.Errorf("UnmarkText 不应改动文本，实际 %q", f.text)
	}
}

// TestTextFieldMoveExtend 验证 Shift 选择：保留锚点，向左选时标记 Reversed。
func TestTextFieldMoveExtend(t *testing.T) {
	f := newTextField("abcdef")
	f.move(2, false)
	if f.selection.Range.Start != 2 || f.selection.Reversed {
		t.Fatalf("单纯移动不该产生选区: %+v", f.selection)
	}
	f.move(5, true) // 向右扩选
	if got := f.selection.Range; got.Start != 2 || got.End != 5 {
		t.Errorf("向右扩选应为 2–5，实际 %d–%d", got.Start, got.End)
	}
	if f.selection.Reversed {
		t.Error("向右扩选时 Reversed 应为 false")
	}
	f.move(1, true) // 从锚点 2 往左扩选
	if got := f.selection.Range; got.Start != 1 || got.End != 2 {
		t.Errorf("向左扩选应为 1–2，实际 %d–%d", got.Start, got.End)
	}
	if !f.selection.Reversed {
		t.Error("向左扩选时 Reversed 应为 true（光标在锚点左边）")
	}
}

// TestTextFieldSelectAll 验证全选用的是 UTF-16 长度（不是字节长度）。
func TestTextFieldSelectAll(t *testing.T) {
	f := newTextField("a中😀")
	f.selectAll()
	want := ui.UTF16Len(f.text) // 1 + 1 + 2
	if got := f.selection.Range; got.Start != 0 || got.End != want {
		t.Errorf("全选应为 0–%d，实际 %d–%d", want, got.Start, got.End)
	}
}

// TestTextFieldBackspace 验证退格删掉光标前一个「字」（按排版边界，不是按字节）。
func TestTextFieldBackspace(t *testing.T) {
	f := newTextField("ab 中😀")
	end := ui.UTF16Len(f.text)
	f.move(end, false)
	f.input(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyBackspace})
	if f.text != "ab 中" {
		t.Fatalf("退格应删掉最后一个字（emoji 整个删掉），实际 %q", f.text)
	}
	// 连续退格不会panic，且最终能删空。
	for i := 0; i < 10 && f.text != ""; i++ {
		f.input(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyBackspace})
	}
	if f.text != "" {
		t.Errorf("连续退格应删空，实际 %q", f.text)
	}
	if f.selection.Range.Start != 0 {
		t.Errorf("删空后光标应在 0，实际 %d", f.selection.Range.Start)
	}
}

// TestTextFieldHomeEnd 验证 Home / End 把光标移到文本首尾。
func TestTextFieldHomeEnd(t *testing.T) {
	f := newTextField("abc")
	f.input(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyHome})
	if f.selection.Range.Start != 0 {
		t.Errorf("Home 后光标应在 0，实际 %d", f.selection.Range.Start)
	}
	f.input(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnd})
	if got := f.selection.Range.Start; got != ui.UTF16Len(f.text) {
		t.Errorf("End 后光标应在 %d，实际 %d", ui.UTF16Len(f.text), got)
	}
}

// TestTextFieldIndexAndBounds 验证「点哪落到哪个字」与「某个字在哪」互为逆运算：
// 两个方法用的内边距必须一致，否则输入法候选窗会飘。
func TestTextFieldIndexAndBounds(t *testing.T) {
	f := newTextField("abcdef")
	for _, x := range []float32{12, 20, 40, 60} {
		idx, ok := f.IndexForPoint(ui.Point{X: x, Y: fieldPadY + 4})
		if !ok {
			t.Fatalf("x=%v: 命中测试失败", x)
		}
		b, _, ok := f.BoundsForRange(ui.TextInputRange{Start: idx, End: idx})
		if !ok {
			t.Fatalf("x=%v: 取不到下标 %d 的位置", x, idx)
		}
		// 该下标的位置应当就在点击点的附近（同一行、横向不超过一个字宽）。
		if b.Y > fieldPadY+4+20 || b.X > x+20 {
			t.Errorf("x=%v: 下标 %d 的位置 (%v,%v) 与点击点不匹配", x, idx, b.X, b.Y)
		}
	}
}
