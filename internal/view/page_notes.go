package view

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"bestuav-mygo/internal/apppath"
	"bestuav-mygo/internal/logging"
)

// ---- 官方示例：笔记（富文本编辑）----
//
// 移植自 mygo examples/notes：左侧笔记列表、右侧富文本编辑器、格式工具条
// （粗体/斜体/下划线/删除线/代码/链接/列表/引用/代码块），内容存成 JSON。
//
// 与示例的差别：
//   - 存放位置改到本应用的配置目录（`<配置目录>/notes.json`），不另建 "MyGo Notes"；
//   - 去掉了应用级菜单（RoleFileMenu 等），改为页面内的工具条；
//   - 编辑区放进固定高度的容器：页面整体是可滚动的，编辑器需要确定的边界。

// noteItem 一条笔记。
type noteItem struct {
	ID      int           `json:"id"`
	Text    richtext.Text `json:"text"`
	Updated time.Time     `json:"updated"`
	doc     *richtext.Document
}

// document 返回笔记的文档对象（首次访问时创建）。
func (n *noteItem) document() *richtext.Document {
	if n.doc == nil {
		n.doc = richtext.NewDocument(n.Text)
	}
	return n.doc
}

// title 取笔记第一行作为标题（空则 "新笔记"）。
func (n *noteItem) title() string {
	t := n.Text
	if n.doc != nil {
		t = n.doc.Text()
	}
	s := t.PlainText()
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if s = strings.TrimSpace(s); s == "" {
		return "新笔记"
	}
	return s
}

// ensureNotes 首次进入时从配置目录加载笔记（没有则建一条欢迎笔记）。
//
// notePath 若已被外部设置（如单测注入临时文件），就沿用，不再去读真实用户目录，
// 避免测试结果依赖开发机上已有的笔记。
func (a *App) ensureNotes() {
	if a.noteLoaded {
		return
	}
	a.noteLoaded = true
	if a.notePath == "" {
		if dir, err := apppath.ConfigDir(); err == nil {
			a.notePath = filepath.Join(dir, "notes.json")
		}
	}
	if a.notePath != "" {
		if b, err := os.ReadFile(a.notePath); err == nil && json.Unmarshal(b, &a.notes) == nil {
			for _, n := range a.notes {
				a.noteNext = max(a.noteNext, n.ID)
			}
		}
	}
	if len(a.notes) == 0 {
		a.noteNext++
		a.notes = []*noteItem{{ID: a.noteNext, Updated: time.Now(), Text: notesWelcome()}}
	}
}

// currentNote 当前选中的笔记（无效选择时回落到第一条并补齐）。
func (a *App) currentNote() *noteItem {
	for _, n := range a.notes {
		if strconv.Itoa(n.ID) == a.noteSel {
			return n
		}
	}
	if len(a.notes) == 0 {
		a.addNote()
	}
	a.noteSel = strconv.Itoa(a.notes[0].ID)
	return a.notes[0]
}

// addNote 新建一条空笔记并选中。
func (a *App) addNote() *noteItem {
	a.noteNext++
	n := &noteItem{
		ID:      a.noteNext,
		Updated: time.Now(),
		Text: richtext.TextOf(richtext.Paragraph{
			Style: richtext.ParagraphStyle{Heading: 1},
		}),
	}
	a.notes = append([]*noteItem{n}, a.notes...)
	a.noteSel = strconv.Itoa(n.ID)
	return n
}

// saveNotes 把笔记写回配置目录（失败只记日志，不影响编辑）。
func (a *App) saveNotes() {
	if a.notePath == "" {
		return
	}
	for _, n := range a.notes {
		if n.doc != nil {
			n.Text = n.doc.Text()
		}
	}
	b, err := json.MarshalIndent(a.notes, "", "  ")
	if err != nil {
		logging.Errv("notes", "序列化笔记失败", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(a.notePath), 0o755); err != nil {
		logging.Errv("notes", "创建笔记目录失败", err)
		return
	}
	if err := os.WriteFile(a.notePath, b, 0o644); err != nil {
		logging.Errv("notes", "写入笔记失败", err)
	}
}

// notesPage 笔记页：左列表 + 右编辑器 + 工具条。
func (a *App) notesPage(c *ui.Context, t *ui.Theme) {
	a.ensureNotes()
	n := a.currentNote()
	doc := n.document()
	if c.Shortcut(ui.Cmd, ui.KeyN) {
		n = a.addNote()
		doc = n.document()
	}

	// 固定高度：编辑器需要确定的上下边界（页面本身是可滚动的）。
	ui.Column(c).Height(520).Children(func() {
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			// 左：笔记列表
			ui.Column(c).Width(230).Shrink(0).Gap(8).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "笔记").Bold().Grow(1)
					if ui.Button(c, "新建").Tooltip("新建笔记（⌘/Ctrl+N）").Clicked() {
						a.addNote()
					}
				})
				ui.Sidebar(c, &a.noteSel, func() {
					for _, it := range a.notes {
						ui.SidebarItem(c.Key(it.ID), strconv.Itoa(it.ID), nil, it.title())
					}
				}).Grow(1).Label("笔记列表")
			})

			// 右：编辑器
			ui.Column(c).Grow(1).MinWidth(0).Gap(10).Children(func() {
				a.noteToolbar(c, t, doc)
				changed := richtext.Editor(c.Key(n.ID), doc, richtext.EditorOptions{
					Label:         "笔记",
					Placeholder:   "标题",
					OnLinkCommand: func() { a.noteLinkURL = doc.Formatting().Link; a.noteLinkOpen = true },
				}).Grow(1).Changed()
				if changed {
					n.Text, n.Updated = doc.Text(), time.Now()
					a.saveNotes()
				}
				words := len(strings.FieldsFunc(doc.Text().PlainText(), func(r rune) bool {
					return unicode.IsSpace(r)
				}))
				ui.Text(c, itoa(words)+" 词 · 修改于 "+n.Updated.Format("01-02 15:04")).
					FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			})
		})
	})
}

// noteToolbar 格式工具条：显示当前选区的格式，点击切换。
func (a *App) noteToolbar(c *ui.Context, t *ui.Theme, doc *richtext.Document) {
	f := doc.Formatting()
	button := func(label, tip string, on bool, run func()) ui.Element {
		b := ui.Button(c, label).KeepFocus().Padding(4, 9).MinWidth(30).Tooltip(tip)
		if on {
			b.Background(t.Accent.Alpha(0.18))
		}
		if b.Clicked() {
			run()
		}
		return b
	}
	divider := func() { ui.Box(c).Width(1).Height(20).Background(t.Border) }

	ui.Row(c).Gap(4).Wrap().AlignItems(ui.Center).Children(func() {
		style := map[int]string{0: "正文", 1: "标题", 2: "一级标题", 3: "二级标题"}[f.Paragraph.Heading]
		if f.ParagraphsMixed {
			style = ""
		}
		if ui.Select(c, &style, []string{"标题", "一级标题", "二级标题", "正文"}).
			KeepFocus().Width(130).Label("段落样式").Changed() {
			doc.SetHeading(map[string]int{"标题": 1, "一级标题": 2, "二级标题": 3}[style])
		}
		divider()
		button("B", "粗体", f.Bold == richtext.FormatOn, doc.ToggleBold).Bold()
		button("I", "斜体", f.Italic == richtext.FormatOn, doc.ToggleItalic).Italic()
		button("U", "下划线", f.Underline == richtext.FormatOn, doc.ToggleUnderline).Underline()
		button("S", "删除线", f.Strikethrough == richtext.FormatOn, doc.ToggleStrikethrough).Strikethrough()
		button("</>", "代码", f.Code == richtext.FormatOn, doc.ToggleCode)

		link := button("链接", "插入链接", f.Link != "", func() {
			a.noteLinkURL = doc.Formatting().Link
			a.noteLinkOpen = true
		})
		ui.Popover(c, link, &a.noteLinkOpen, func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				input := ui.TextInput(c, &a.noteLinkURL).Placeholder("https://").Width(260).AutoFocus().Label("URL")
				if input.Submitted() || ui.PrimaryButton(c, "确定").Clicked() {
					doc.SetLink(strings.TrimSpace(a.noteLinkURL))
					a.noteLinkOpen = false
				}
				if f.Link != "" && ui.Button(c, "移除").Clicked() {
					doc.SetLink("")
					a.noteLinkOpen = false
				}
			})
		})

		divider()
		button("•", "无序列表", f.Paragraph.List == richtext.Bullets, func() {
			doc.ToggleList(richtext.Bullets)
		})
		button("1.", "有序列表", f.Paragraph.List == richtext.Numbers, func() {
			doc.ToggleList(richtext.Numbers)
		})
		button("☐", "任务清单", f.Paragraph.List == richtext.Tasks, func() {
			doc.ToggleList(richtext.Tasks)
		})
		button("❝", "引用", f.Paragraph.Quote, doc.ToggleQuote)
		button("{}", "代码块", f.Paragraph.CodeBlock, doc.ToggleCodeBlock)

		divider()
		if ui.Button(c, "复制 Markdown").KeepFocus().Clicked() {
			a.copyText(doc.Text().Markdown(), "Markdown 已复制")
		}
	})
}

// copyCurrentNote 把当前笔记复制为 Markdown（页头工具条用）。
func (a *App) copyCurrentNote() {
	a.ensureNotes()
	a.copyText(a.currentNote().document().Text().Markdown(), "笔记已复制为 Markdown")
}

// notesWelcome 首条欢迎笔记：展示支持的格式。
func notesWelcome() richtext.Text {
	code := richtext.Attributes{Code: true}
	return richtext.TextOf(
		richtext.Paragraph{
			Style: richtext.ParagraphStyle{Heading: 1},
			Runs:  []richtext.Run{{Text: "欢迎使用笔记"}},
		},
		richtext.Paragraph{Runs: []richtext.Run{
			{Text: "用工具条或快捷键写 "},
			{Text: "粗体", Attributes: richtext.Attributes{Weight: 700}},
			{Text: "、"},
			{Text: "斜体", Attributes: richtext.Attributes{Italic: true}},
			{Text: "、"},
			{Text: "链接", Attributes: richtext.Attributes{Link: "https://github.com/egoist/mygo"}},
			{Text: " 和列表。"},
		}},
		richtext.Paragraph{
			Style: richtext.ParagraphStyle{Heading: 2},
			Runs:  []richtext.Run{{Text: "试试看"}},
		},
		richtext.Paragraph{
			Style: richtext.ParagraphStyle{List: richtext.Tasks, Checked: true},
			Runs:  []richtext.Run{{Text: "打开这条笔记"}},
		},
		richtext.Paragraph{
			Style: richtext.ParagraphStyle{List: richtext.Tasks},
			Runs: []richtext.Run{
				{Text: "输入 "}, {Text: "- ", Attributes: code},
				{Text: " 开始列表，"}, {Text: "# ", Attributes: code},
				{Text: " 变成标题"},
			},
		},
		richtext.Paragraph{
			Style: richtext.ParagraphStyle{List: richtext.Tasks},
			Runs:  []richtext.Run{{Text: "点「复制 Markdown」把笔记转成 Markdown"}},
		},
		richtext.Paragraph{Runs: []richtext.Run{
			{Text: "内容会保存到 "},
			{Text: "配置目录/notes.json", Attributes: code},
			{Text: "，重启后还在。"},
		}},
	)
}
