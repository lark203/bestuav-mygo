package view

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/editor"
)

// TestCodeWordAt 验证「光标下的词」的提取：给缓冲区里的一个偏移量，应返回该词
// 文本与它在缓冲区里的起止偏移（悬停提示要用这两个偏移画下划线）。
func TestCodeWordAt(t *testing.T) {
	b := editor.NewBuffer("func main() {\n\tprintln(\"hi\")\n}")
	cases := []struct {
		off        int
		word       string
		start, end int
	}{
		{0, "func", 0, 4},       // 词首
		{2, "func", 0, 4},       // 词中：仍算同一个词
		{5, "main", 5, 9},       // 下一个词
		{9, "main", 5, 9},       // 词尾的括号：归属左边的词
		{11, "", 11, 11},        // 空格与 { 之间：没有词
		{15, "println", 15, 22}, // 第二行
	}
	for _, tc := range cases {
		word, start, end := codeWordAt(b, tc.off)
		if word != tc.word || start != tc.start || end != tc.end {
			t.Errorf("offset %d: 得到 (%q, %d, %d)，期望 (%q, %d, %d)",
				tc.off, word, start, end, tc.word, tc.start, tc.end)
		}
		// 取到的词必须真能从缓冲区里按偏移切出来，否则悬停下划线会画错位置。
		if got := b.Slice(start, end); got != word {
			t.Errorf("offset %d: 切片 %q 与词 %q 不一致", tc.off, got, word)
		}
	}
}

// TestCodeWordsOf 验证补全候选：去重、排除前缀本身、跳过过短的词、按字典序。
func TestCodeWordsOf(t *testing.T) {
	b := editor.NewBuffer("alpha alpha beta gamma delta\nfunc main()")
	out := codeWordsOf(b, "alpha")
	var labels []string
	for _, c := range out {
		labels = append(labels, c.Label)
	}
	for _, want := range []string{"beta", "delta", "gamma", "main"} {
		found := false
		for _, l := range labels {
			if l == want {
				found = true
			}
		}
		if !found {
			t.Errorf("候选里缺少 %q，实际: %v", want, labels)
		}
	}
	for _, l := range labels {
		if l == "alpha" {
			t.Errorf("前缀本身不该出现在候选里: %v", labels)
		}
		if len(l) < 3 {
			t.Errorf("过短的词 %q 不该出现（补全全是噪音）", l)
		}
	}
	for i := 1; i < len(labels); i++ {
		if labels[i-1] > labels[i] {
			t.Fatalf("候选未按字典序: %v", labels)
		}
	}
}

// TestCodeTodos 验证诊断：每行带 TODO 的注释都变成一条告警，范围从 TODO 到行尾。
func TestCodeTodos(t *testing.T) {
	b := editor.NewBuffer("package a\n// TODO: 限流\nfunc a() {}\n// TODO: 缓存\n")
	ds := codeTodos(b)
	if len(ds) != 2 {
		t.Fatalf("应有 2 条诊断，实际 %d", len(ds))
	}
	want := []string{"TODO: 限流", "TODO: 缓存"}
	for i, d := range ds {
		if d.Severity != editor.SeverityWarning {
			t.Errorf("第 %d 条不是告警级别", i)
		}
		if d.Message != want[i] {
			t.Errorf("第 %d 条文案 %q，期望 %q", i, d.Message, want[i])
		}
		if got := b.Slice(d.Start, d.End); !strings.HasPrefix(got, "TODO") || got != want[i] {
			t.Errorf("第 %d 条范围不对: %q", i, got)
		}
	}
}

// TestCodeEditorDiffStats 验证改动后 diff 统计为正，且缓冲区进入「已修改」状态
// （文件列表要靠它显示 •）。
func TestCodeEditorDiffStats(t *testing.T) {
	ce := newCodeEditor()
	f := ce.current()
	if f.name != "server.go" {
		t.Fatalf("默认应选中 server.go，实际 %q", f.name)
	}
	if added, deleted := f.diff.Stats(); added != 0 || deleted != 0 {
		t.Fatalf("初始状态不应有改动，实际 +%d −%d", added, deleted)
	}
	// diff 的基线是 original，故往缓冲区里插一行会产生一处改动。
	f.editor.Buffer.Edit(editor.Edit{Start: 0, End: 0, Text: "// 新增一行\n"})
	if added, _ := f.diff.Stats(); added != 1 {
		t.Errorf("插入一行后应 +1，实际 +%d", added)
	}
	if !f.editor.Buffer.Modified() {
		t.Error("缓冲区应处于已修改状态")
	}
}

// TestCodeEditorCurrentFallsBack 验证选中项失效时回落到第一个文件，不会取到 nil
// （页面每帧都调 current()，取到 nil 就是空指针崩溃）。
func TestCodeEditorCurrentFallsBack(t *testing.T) {
	ce := newCodeEditor()
	ce.selected = "不存在的文件.go"
	if got := ce.current(); got == nil || got.name != "server.go" {
		t.Fatalf("应回落到 server.go，实际 %v", got)
	}
	if ce.selected != "server.go" {
		t.Errorf("回落时应把选中项写回，实际 %q", ce.selected)
	}
}
