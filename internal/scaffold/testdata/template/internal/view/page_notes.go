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
// notesState 笔记页的私有状态（列表、选中项、id 递增、落盘状态、链接弹层）。
//
// 存在 pageState.data 里：切走再回来还是这份笔记；页面被回收时丢弃内存副本，
// 下次进入从 notes.json 重新加载（落盘由 saveNotes 负责）。
type notesState struct {
	rows     []*noteItem
	sel      string
	next     int
	linkOpen bool
	linkURL  string
	path     string // notes.json 路径（空 = 不落盘）
	loaded   bool   // 是否已尝试加载
}

// notesStateOf 取本页状态。
func (a *App) notesStateOf() *notesState {
	return pageStateOf(a, "official-notes", "state", func() *notesState { return &notesState{} })
}

// ---- 官方示例：笔记（富文本）----
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
	ns := a.notesStateOf()
	if ns.loaded {
		return
	}
	ns.loaded = true
	if ns.path == "" {
		if dir, err := apppath.ConfigDir(); err == nil {
			ns.path = filepath.Join(dir, "notes.json")
		}
	}
	if ns.path != "" {
		if b, err := os.ReadFile(ns.path); err == nil && json.Unmarshal(b, &ns.rows) == nil {
			for _, n := range ns.rows {
				ns.next = max(ns.next, n.ID)
			}
		}
	}
	if len(ns.rows) == 0 {
		ns.next++
		ns.rows = []*noteItem{{ID: ns.next, Updated: time.Now(), Text: notesWelcome()}}
	}
}

// currentNote 当前选中的笔记（无效选择时回落到第一条并补齐）。
func (a *App) currentNote() *noteItem {
	ns := a.notesStateOf()
	for _, n := range ns.rows {
		if strconv.Itoa(n.ID) == ns.sel {
			return n
		}
	}
	if len(ns.rows) == 0 {
		a.addNote()
	}
	ns.sel = strconv.Itoa(ns.rows[0].ID)
	return ns.rows[0]
}

// addNote 新建一条空笔记并选中。
func (a *App) addNote() *noteItem {
	ns := a.notesStateOf()
	ns.next++
	n := &noteItem{
		ID:      ns.next,
		Updated: time.Now(),
		Text: richtext.TextOf(richtext.Paragraph{
			Style: richtext.ParagraphStyle{Heading: 1},
		}),
	}
	ns.rows = append([]*noteItem{n}, ns.rows...)
	ns.sel = strconv.Itoa(n.ID)
	return n
}

// saveNotes 把笔记写回配置目录（失败只记日志，不影响编辑）。
func (a *App) saveNotes() {
	ns := a.notesStateOf()
	if ns.path == "" {
		return
	}
	for _, n := range ns.rows {
		if n.doc != nil {
			n.Text = n.doc.Text()
		}
	}
	b, err := json.MarshalIndent(ns.rows, "", "  ")
	if err != nil {
		logging.Errv("notes", "序列化笔记失败", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(ns.path), 0o755); err != nil {
		logging.Errv("notes", "创建笔记目录失败", err)
		return
	}
	if err := os.WriteFile(ns.path, b, 0o644); err != nil {
		logging.Errv("notes", "写入笔记失败", err)
	}
}

// notesPage 笔记页：左列表 + 右编辑器 + 工具条。
func (a *App) notesPage(c *ui.Context, t *ui.Theme) {
	ns := a.notesStateOf()
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
				ui.Sidebar(c, &ns.sel, func() {
					for _, it := range ns.rows {
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
					OnLinkCommand: func() { ns.linkURL = doc.Formatting().Link; ns.linkOpen = true },
				}).Grow(1).Changed()
				if changed {
					n.Text, n.Updated = doc.Text(), time.Now()
					a.saveNotes()
				}
				// 底部状态行：词数与修改时间。
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
//
// 这里只放「格式」，不放动作 —— 导出类的按钮在编辑区底部（见 notesPage）：它
// 挂在末尾时，13 个按钮挤在一行，稍窄一点的窗口就会把最后一项挤到第二行。
func (a *App) noteToolbar(c *ui.Context, t *ui.Theme, doc *richtext.Document) {
	ns := a.notesStateOf()
	f := doc.Formatting()
	button := func(label, tip string, on bool, run func()) ui.Element {
		b := ui.Button(c, label).KeepFocus().Padding(5, 10).MinWidth(32).Tooltip(tip)
		if on {
			b.Background(t.Accent.Alpha(0.18))
		}
		if b.Clicked() {
			run()
		}
		return b
	}
	divider := func() { ui.Box(c).Width(1).Height(22).Background(t.Border) }

	// 两行而不是一行：13 个按钮塞一行，窗口稍窄就把末尾的挤到第二行（孤零零
	// 一个）。分成「行内格式」与「块级格式」两行后，每行都宽松，也不再依赖
	// 窗口宽度 —— 两行都是左对齐，看起来是分组而不是换行。
	ui.Column(c).Gap(8).Children(func() {
		ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			style := map[int]string{0: "正文", 1: "标题", 2: "一级标题", 3: "二级标题"}[f.Paragraph.Heading]
			if f.ParagraphsMixed {
				style = ""
			}
			if ui.Select(c, &style, []string{"标题", "一级标题", "二级标题", "正文"}).
				KeepFocus().Label("段落样式").Changed() {
				doc.SetHeading(map[string]int{"标题": 1, "一级标题": 2, "二级标题": 3}[style])
			}
			divider()
			button("B", "粗体", f.Bold == richtext.FormatOn, doc.ToggleBold).Bold()
			button("I", "斜体", f.Italic == richtext.FormatOn, doc.ToggleItalic).Italic()
			button("U", "下划线", f.Underline == richtext.FormatOn, doc.ToggleUnderline).Underline()
			button("S", "删除线", f.Strikethrough == richtext.FormatOn, doc.ToggleStrikethrough).Strikethrough()
			button("</>", "代码", f.Code == richtext.FormatOn, doc.ToggleCode)

			link := button("链接", "插入链接", f.Link != "", func() {
				ns.linkURL = doc.Formatting().Link
				ns.linkOpen = true
			})
			ui.Popover(c, link, &ns.linkOpen, func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					input := ui.TextInput(c, &ns.linkURL).Placeholder("https://").Width(260).AutoFocus().Label("URL")
					if input.Submitted() || ui.PrimaryButton(c, "确定").Clicked() {
						doc.SetLink(strings.TrimSpace(ns.linkURL))
						ns.linkOpen = false
					}
					if f.Link != "" && ui.Button(c, "移除").Clicked() {
						doc.SetLink("")
						ns.linkOpen = false
					}
				})
			})
		})

		ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			button("•", "无序列表", f.Paragraph.List == richtext.Bullets, func() {
				doc.ToggleList(richtext.Bullets)
			})
			button("1.", "有序列表", f.Paragraph.List == richtext.Numbers, func() {
				doc.ToggleList(richtext.Numbers)
			})
			button("☐", "任务清单", f.Paragraph.List == richtext.Tasks, func() {
				doc.ToggleList(richtext.Tasks)
			})
			divider()
			button("❝", "引用", f.Paragraph.Quote, doc.ToggleQuote)
			button("{}", "代码块", f.Paragraph.CodeBlock, doc.ToggleCodeBlock)

			// 导出动作靠右：它是对「整条笔记」的动作，不是选区格式，所以与
			// 格式按钮拉开距离、顶到行尾（原先挂在工具条末尾，窗口稍窄就会被
			// 挤到第二行的行首，孤零零一个）。
			ui.Row(c).Grow(1)
			if ui.Button(c, "复制 Markdown").KeepFocus().Padding(5, 10).
				Tooltip("把这条笔记复制为 Markdown").Clicked() {
				a.copyText(doc.Text().Markdown(), "Markdown 已复制")
			}
		})
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
