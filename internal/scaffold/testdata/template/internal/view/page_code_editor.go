package view

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"bestuav-mygo/internal/ui/kit"
)

// ---- 官方示例：代码编辑器（语法高亮 / 补全 / 悬停 / 诊断 / diff）----
//
// 移植自 mygo examples/code-editor。示例文件在 code_samples.go，
// 本文件是状态与版面。
//
// 与示例的差别：
//   - 示例把「文件列表」做成窗口级侧栏；本应用已有自己的导航侧栏，
//     故把它收成页面内的一列（与笔记页一致）；
//   - 顶部工具条塞进卡片的标题行（kit.CardHeader 的动作区），省一层高度；
//   - 编辑区放进固定高度的容器：页面整体是可滚动的，编辑器需要确定的边界。

// codeFile 示例里的一个文件：原文（diff 的基线）、正在编辑的缓冲区，
// 以及编辑器与 diff 两处的状态。
type codeFile struct {
	name     string
	original *editor.Buffer
	editor   *editor.State
	diff     *editor.DiffState
}

// codeEditor 代码编辑器页的全部状态。
type codeEditor struct {
	files    []*codeFile
	selected string // 当前文件名
	page     int    // 0 编辑器，1 变更
	layout   int    // diff 布局：0 并排，1 统一
	inline   string // 行内差异粒度：词 / 字符 / 关
	wrap     bool   // 软换行
}

// newCodeEditor 构造示例状态（五个示例文件，默认选中 server.go）。
func newCodeEditor() *codeEditor {
	ce := &codeEditor{inline: "词", selected: "server.go"}
	for _, s := range codeSamples {
		lang := editor.LanguageFor(s.name)
		original := editor.NewBuffer(s.text)
		original.SetLanguage(lang)
		buf := editor.NewBuffer(s.text)
		buf.SetLanguage(lang)
		// WrapGuides：100 列处画一条竖线，提示过长的行。
		ce.files = append(ce.files, &codeFile{
			name:     s.name,
			original: original,
			editor:   &editor.State{Buffer: buf, DiffBase: original, WrapGuides: []int{100}},
			diff:     &editor.DiffState{Old: original, New: buf, Editable: true, HunkActions: true, LineSelection: true},
		})
	}
	return ce
}

// current 当前选中的文件（选择失效时回落到第一个）。
func (ce *codeEditor) current() *codeFile {
	for _, f := range ce.files {
		if f.name == ce.selected {
			return f
		}
	}
	ce.selected = ce.files[0].name
	return ce.files[0]
}

// codeEditorPage 代码编辑器页。
func (a *App) codeEditorPage(c *ui.Context, t *ui.Theme) {
	if a.ce == nil {
		a.ce = newCodeEditor()
	}
	ce := a.ce
	f := ce.current()
	// 软换行是整页共用的开关，两个视图都跟着变。
	f.editor.SoftWrap = ce.wrap
	f.diff.Wrap = ce.wrap

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "代码编辑器",
			"语法高亮 · 括号匹配 · 单词补全 · 悬停 · TODO 诊断 · 行号槽里的变更标记", func() {
				// 只放两个全局开关：diff 专用的（布局、行内粒度）放到差异视图上方的细条里，
				// 否则标题行会被挤窄、说明文字被迫折成两行。
				ui.Segmented(c, &ce.page, "编辑器", "变更").Label("视图")
				ui.Text(c, "软换行").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				ui.Switch(c, &ce.wrap).Label("软换行")
			})

		// 固定高度：编辑器需要确定的上下边界（页面本身是可滚动的）。
		// 注意别加 Fill()——Fill 会把高度交回父容器，而父容器的高度又由内容决定，
		// 结果编辑器被压成「最小可见行数」（实测只剩 8 行）。
		ui.Row(c).Height(470).FillWidth().AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Width(200).Shrink(0).Background(t.Surface).Children(func() {
				ui.Sidebar(c, &ce.selected, func() {
					for _, it := range ce.files {
						label := it.name
						if it.editor.Buffer.Modified() {
							label += " •"
						}
						ui.SidebarItem(c, it.name, nil, label)
					}
				}).Grow(1).Label("文件")
			})
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				if ce.page == 0 {
					a.codeEditorBody(c, t, f)
				} else {
					a.codeDiffBody(c, t, f)
				}
			})
		})
	})
}

// codeEditorBody 编辑器：视图 + 底部状态栏。
func (a *App) codeEditorBody(c *ui.Context, t *ui.Theme, f *codeFile) {
	st := f.editor
	// 补全：把缓冲区里的词按前缀过滤（没有语言服务器时编辑器都这么干）。
	st.OnCompletion = func(r editor.CompletionRequest) {
		st.ShowCompletions(r.Offset-len(r.Prefix), codeWordsOf(st.Buffer, r.Prefix))
	}
	// 悬停：报告光标下的词在文件里出现了几次。
	st.OnHover = func(r editor.HoverRequest) {
		word, start, end := codeWordAt(st.Buffer, r.Offset)
		if word == "" {
			return
		}
		n := strings.Count(st.Buffer.String(), word)
		st.ShowHover(start, end, fmt.Sprintf("%s：在 %s 里出现 %d 次", word, f.name, n), nil)
	}
	// 诊断：把 TODO 注释标成告警（版本号用来让编辑器判断诊断是否过期）。
	st.SetDiagnostics(st.Buffer.Version(), codeTodos(st.Buffer))

	editor.View(c, st).Grow(1).FillWidth()

	pos := st.CursorPos()
	added, deleted := f.diff.Stats()
	status := fmt.Sprintf("行 %d，列 %d", pos.Line+1, pos.Col+1)
	if n := len(st.Selections()); n > 1 {
		status += fmt.Sprintf(" · %d 个选区", n)
	}
	lang := "纯文本"
	if l := st.Buffer.Language(); l != nil {
		lang = l.Name
	}
	ui.Row(c).Gap(16).Padding(4, 12).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Text(c, status).FontSize(12).TextColor(t.TextMuted)
		ui.Text(c, fmt.Sprintf("+%d −%d", added, deleted)).FontSize(12).TextColor(t.TextMuted)
		ui.Box(c).Grow(1)
		ui.Text(c, fmt.Sprintf("%d 行 · %s", st.Buffer.LineCount(), lang)).FontSize(12).TextColor(t.TextMuted)
	})
}

// codeDiffBody 变更：与原文的 diff（并排或统一），可在里面继续编辑。
func (a *App) codeDiffBody(c *ui.Context, t *ui.Theme, f *codeFile) {
	d := f.diff
	d.Layout = editor.DiffLayout(a.ce.layout)
	switch a.ce.inline {
	case "字符":
		d.Inline = editor.DiffInlineChar
	case "关":
		d.Inline = editor.DiffInlineNone
	default:
		d.Inline = editor.DiffInlineWordAlt
	}
	if len(d.Hunks()) == 0 {
		ui.Column(c).Grow(1).Center().Children(func() {
			ui.Text(c, "没有改动").FontSize(18).Bold()
			ui.Text(c, "编辑 "+f.name+" 后这里会出现差异。").TextColor(t.TextMuted)
		})
		return
	}
	// 差异视图专用的两个开关，放在视图正上方的细条里（与编辑器页的差别只在这里）。
	ui.Row(c).Gap(12).Padding(4, 12).AlignItems(ui.Center).Children(func() {
		ui.Segmented(c, &a.ce.layout, "并排", "统一").Label("差异布局")
		ui.Select(c, &a.ce.inline, []string{"词", "字符", "关"}).Label("行内差异")
	})
	editor.DiffView(c, d).Grow(1).FillWidth()
}

// codeWordsOf 用缓冲区里的词补全 prefix（长度 <3 的短词太吵，跳过）。
func codeWordsOf(b *editor.Buffer, prefix string) []editor.Completion {
	seen := map[string]bool{}
	var out []editor.Completion
	isWord := func(r rune) bool {
		return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	for _, w := range strings.FieldsFunc(b.String(), func(r rune) bool { return !isWord(r) }) {
		if len(w) < 3 || seen[w] || w == prefix {
			continue
		}
		seen[w] = true
		out = append(out, editor.Completion{Label: w, Kind: editor.CompletionText, Detail: "词"})
	}
	if b.Language() != nil {
		for _, k := range []string{"func", "return", "import", "package", "const", "struct", "interface", "range"} {
			if !seen[k] {
				seen[k] = true
				out = append(out, editor.Completion{Label: k, Kind: editor.CompletionKeyword, Detail: "关键字"})
			}
		}
	}
	slices.SortFunc(out, func(x, y editor.Completion) int { return strings.Compare(x.Label, y.Label) })
	return out
}

// codeWordAt 返回 offset 处那个词的文本与它在缓冲区里的起止偏移。
func codeWordAt(b *editor.Buffer, off int) (word string, start, end int) {
	p := b.Pos(off)
	line := b.Line(p.Line)
	isWord := func(r byte) bool {
		return r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
	}
	a, z := p.Col, p.Col
	for a > 0 && isWord(line[a-1]) {
		a--
	}
	for z < len(line) && isWord(line[z]) {
		z++
	}
	start = off - (p.Col - a)
	return line[a:z], start, start + z - a
}

// codeTodos 把每个 TODO 注释标成一条告警诊断。
func codeTodos(b *editor.Buffer) []editor.Diagnostic {
	var out []editor.Diagnostic
	for i := range b.LineCount() {
		line := b.Line(i)
		if at := strings.Index(line, "TODO"); at >= 0 {
			start := b.Offset(editor.Pos{Line: i, Col: at})
			out = append(out, editor.Diagnostic{
				Start: start, End: start + len(line) - at,
				Severity: editor.SeverityWarning,
				Message:  strings.TrimSpace(line[at:]),
				Source:   "todo",
			})
		}
	}
	return out
}
