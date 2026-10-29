package view

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// tbTestDemo 造一个小文档用于断言（真页面是 10 万行，测试里没必要建那么大的）。
func tbTestDemo(lines int) *tbDemo {
	var b strings.Builder
	for i := range lines {
		b.WriteString("Line " + itoa(i+1) + ": editable text, 😀, العربية\n")
	}
	return &tbDemo{buf: ui.NewTextBuffer(b.String()), pending: -1}
}

// LineCount 的语义是「换行数 + 1」：每行末尾都有换行，文档最后那个空行也算一行。
// tbTestDemo(n) 写出 n 行文本，因此 LineCount 是 n+1。
func TestTbLineOps(t *testing.T) {
	d := tbTestDemo(5)
	s := d.buf.Snapshot()
	if got := s.LineCount(); got != 6 {
		t.Fatalf("LineCount = %d，期望 6（5 行文本 + 末尾空行）", got)
	}
	// 第 3 行：LineRange 给出左闭右开的 rune 区间，Line 取出来不含换行符。
	a, z := s.LineRange(2)
	if line := s.Line(2); line != "Line 3: editable text, 😀, العربية" {
		t.Errorf("Line(2) = %q", line)
	}
	if z-a != len([]rune(s.Line(2))) {
		t.Errorf("区间长度 %d 与行的 rune 数 %d 不一致", z-a, len([]rune(s.Line(2))))
	}
	// 这一行含 emoji（2 个 UTF-16 单元 / 4 字节 / 1 rune），三种单位必须不同。
	runes := z - a
	bytes := len(s.Line(2))
	units := s.UTF16Offset(z) - s.UTF16Offset(a)
	if !(runes < units && units < bytes) {
		t.Errorf("期望 runes < utf16 < bytes，实际 %d / %d / %d", runes, units, bytes)
	}
	// 越界行号被钳到两端，不 panic。
	if _, zz := s.LineRange(999); zz != s.Len() {
		t.Errorf("越界行号应钳到末行，z = %d", zz)
	}
}

// TestTbEditAndUndo 验证插入 / 删行 / 撤销这条链路，顺带验版本号推进。
func TestTbEditAndUndo(t *testing.T) {
	d := tbTestDemo(100)
	before := d.buf.Snapshot()
	v0 := before.Version()

	lines0 := d.buf.LineCount()
	line := lines0 / 2 // insertMiddle 插在「第 line 行之前」，新行占据这个下标
	d.insertMiddle()
	if got := d.buf.LineCount(); got != lines0+1 {
		t.Fatalf("插入后行数 = %d，期望 %d", got, lines0+1)
	}
	s := d.buf.Snapshot()
	a, _ := s.LineRange(line)
	if got := s.Line(line); !strings.Contains(got, "inserted in the middle") {
		t.Errorf("第 %d 行 = %q，期望是插入的那一行", line, got)
	}
	if d.pending != a {
		t.Errorf("pending = %d，期望指向插入处 %d", d.pending, a)
	}
	if s.Version() <= v0 {
		t.Errorf("版本号没推进：%d → %d", v0, s.Version())
	}

	d.undo()
	after := d.buf.Snapshot()
	if after.LineCount() != lines0 {
		t.Errorf("撤销后行数 = %d，期望 %d", after.LineCount(), lines0)
	}
	if after.String() != before.String() {
		t.Errorf("撤销后内容与原文不一致")
	}

	// 空栈撤销不该改内容，只给提示。
	d2 := tbTestDemo(3)
	orig := d2.buf.String()
	d2.undo()
	if d2.buf.String() != orig {
		t.Errorf("空栈撤销改了内容")
	}
	if !strings.Contains(d2.note, "没有可撤销") {
		t.Errorf("空栈撤销的提示 = %q", d2.note)
	}
}

// TestTbDeleteRange 验证删 100 行后行数与内容都对。
func TestTbDeleteRange(t *testing.T) {
	d := tbTestDemo(1000)
	mid := d.buf.Snapshot().Line(500)
	n0 := d.buf.Snapshot().LineCount()
	d.deleteHundred()
	s := d.buf.Snapshot()
	if got := s.LineCount(); got != n0-100 {
		t.Fatalf("删除后行数 = %d，期望 %d", got, n0-100)
	}
	// 被删的第 501 行不该还在原位置。
	if s.Line(500) == mid {
		t.Errorf("第 501 行没被删掉")
	}
	d.undo()
	if d.buf.Snapshot().Line(500) != mid {
		t.Errorf("撤销后第 501 行没回来")
	}
}

// TestTbJump 验证行号输入的解析：合法行号定位到行首，非法输入只给提示。
func TestTbJump(t *testing.T) {
	d := tbTestDemo(50)
	d.jumpTo = " 42 "
	d.jump()
	s := d.buf.Snapshot()
	want, _ := s.LineRange(41)
	if d.pending != want {
		t.Errorf("跳转到第 42 行：pending = %d，期望 %d", d.pending, want)
	}

	for _, bad := range []string{"", "abc", "0", "-3", "3.5"} {
		d.pending = -1
		d.jumpTo = bad
		d.jump()
		if d.pending != -1 {
			t.Errorf("行号 %q 不该定位，pending = %d", bad, d.pending)
		}
		if !strings.Contains(d.note, "行号") {
			t.Errorf("行号 %q 的提示 = %q", bad, d.note)
		}
	}
	// 超出行数要钳到末行，而不是越界。
	d.jumpTo = "999999"
	d.jump()
	if d.pending < 0 {
		t.Errorf("超大行号应钳到末行，pending = %d", d.pending)
	}
}

// TestTbSmallEditDoesNotCopyWholeDocument 验证「小改不复制整篇」这个卖点：
// 在同一份文档上，整串拼接的耗时应显著高于索引缓冲的插入。
func TestTbSmallEditDoesNotCopyWholeDocument(t *testing.T) {
	d := tbTestDemo(tbLines) // 与页面同规模，约 4 MB
	s := d.buf.Snapshot()
	a, _ := s.LineRange(s.LineCount() / 2)
	const ins = ">> benchmark <<\n"

	const runs = 50
	start := time.Now()
	for range runs {
		d.buf.Replace(a, a, ins)
	}
	ropeCost := time.Since(start) / runs

	raw := d.buf.String()
	cut := strings.LastIndexByte(raw[:len(raw)/2], '\n') + 1
	dst := make([]byte, 0, len(raw)+len(ins))
	start = time.Now()
	for range 20 {
		dst = dst[:0]
		dst = append(dst, raw[:cut]...)
		dst = append(dst, ins...)
		dst = append(dst, raw[cut:]...)
	}
	strCost := time.Since(start) / 20
	if len(dst) != len(raw)+len(ins) {
		t.Fatal("复制长度不对")
	}
	// 不断言精确数值（机器差异太大），只断言两件事：插入远没到整篇复制的量级，
	// 以及它确实比整串拼接快。二者在同一份文档上跑，量级差应该在百倍以上。
	if ropeCost > 2*time.Millisecond {
		t.Errorf("索引缓冲插入耗时 %s，超过 2ms —— 退化成整篇复制了？", ropeCost)
	}
	if ropeCost*10 > strCost {
		t.Errorf("索引缓冲 %s 与整串拼接 %s 差距不足 10 倍，卖点不成立", ropeCost, strCost)
	}
	t.Logf("文档 %d 行：索引缓冲 %s，整串拼接 %s", s.LineCount(), tbDur(ropeCost), tbDur(strCost))
}

// TestTbFormatting 验证展示用的三个格式化函数。
func TestTbFormatting(t *testing.T) {
	cases := []struct{ in, want string }{
		{"0", "0"}, {"999", "999"}, {"1000", "1,000"},
		{"100000", "100,000"}, {"1234567", "1,234,567"}, {"-4200", "-4,200"},
	}
	for _, c := range cases {
		n := 0
		neg := strings.HasPrefix(c.in, "-")
		digits := strings.TrimPrefix(c.in, "-")
		for _, ch := range digits {
			n = n*10 + int(ch-'0')
		}
		if neg {
			n = -n
		}
		if got := tbNum(n); got != c.want {
			t.Errorf("tbNum(%d) = %q，期望 %q", n, got, c.want)
		}
	}
	if got := tbBytes(512); got != "512 B" {
		t.Errorf("tbBytes(512) = %q", got)
	}
	if got := tbBytes(2048); got != "2.0 KB" {
		t.Errorf("tbBytes(2048) = %q", got)
	}
	if got := tbBytes(4 * 1024 * 1024); got != "4.0 MB" {
		t.Errorf("tbBytes(4MB) = %q", got)
	}
	if got := tbDur(900 * time.Nanosecond); !strings.HasSuffix(got, "ns") {
		t.Errorf("tbDur(900ns) = %q", got)
	}
	if got := tbDur(12 * time.Microsecond); !strings.HasSuffix(got, "µs") {
		t.Errorf("tbDur(12µs) = %q", got)
	}
	if got := tbDur(3 * time.Millisecond); !strings.HasSuffix(got, "ms") {
		t.Errorf("tbDur(3ms) = %q", got)
	}
}

// TestTbBenchmarkRuns 验证对比流程能跑完并把结果写进 bench。
func TestTbBenchmarkRuns(t *testing.T) {
	// 用与页面同规模的文档：整串复制的成本随篇幅线性增长，小文档上差距不明显。
	d := tbTestDemo(tbLines)
	n0 := d.buf.Snapshot().LineCount()
	d.benchmark()
	if len(d.snaps) != 1 {
		t.Fatalf("对比应压入一个快照，实际 %d", len(d.snaps))
	}
	if d.bench == "" {
		t.Fatal("对比没写出结果")
	}
	for _, want := range []string{"索引缓冲", "整串复制", "倍"} {
		if !strings.Contains(d.bench, want) {
			t.Errorf("结果 %q 缺 %q", d.bench, want)
		}
	}
	if got := d.buf.Snapshot().LineCount(); got <= n0 {
		t.Errorf("对比里的插入没生效，行数 %d → %d", n0, got)
	}
	d.undo()
	if d.buf.Snapshot().LineCount() != n0 {
		t.Errorf("对比的插入应一次撤销干净，行数 = %d，期望 %d", d.buf.Snapshot().LineCount(), n0)
	}
	// 卖点成立的前提：在 10 万行文档上，索引缓冲至少快一个数量级。
	if d.lastRope*10 > d.lastStr {
		t.Errorf("差距不足 10 倍：索引缓冲 %s vs 整串复制 %s", tbNs(d.lastRope), tbNs(d.lastStr))
	}
	t.Logf("%s（%d 行）", d.bench, n0)
}

// TestTbUndoStackDepth 验证撤销栈有上限，不会无限增长。
func TestTbUndoStackDepth(t *testing.T) {
	d := tbTestDemo(10)
	for i := 0; i < tbUndoDepth+8; i++ {
		d.pushUndo()
	}
	if len(d.snaps) != tbUndoDepth {
		t.Errorf("撤销栈深度 = %d，期望钳在 %d", len(d.snaps), tbUndoDepth)
	}
}
