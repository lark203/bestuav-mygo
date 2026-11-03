package view

import (
	"time"

	"github.com/ELY-Open-Source/CrychicUI/code"
	"github.com/ELY-Open-Source/CrychicUI/datetime"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"bestuav-mygo/internal/ui/kit"
)

// ---- CrychicUI 对照页 ----
//
// 这两个页面专门用来评估 CrychicUI 的 datetime / code 包（自研成本最高的两类组件），
// 与「组件演示」分组里用本项目 kit 手搓的同类控件做并排对比。原页面一律未改动。
//
// 组件读的是宿主 *ui.Theme，故自动继承「量子科学」主题；CrychicUI 的默认点击/悬停
// 音效已在 ApplyTheme 里统一关闭。

// crySampleGo / crySampleJSON 是代码页的演示内容。
const crySampleGo = `package main

import "fmt"

// Task 是无人机作业任务的最小单元。
type Task struct {
	ID   string
	Name string
	Done bool
}

func main() {
	t := Task{ID: "T-001", Name: "航拍测绘", Done: false}
	fmt.Printf("task %s: %s\n", t.ID, t.Name)
}`

// cryCodeFiles 代码标签页的多个文件。
var cryCodeFiles = []code.CodeFile{
	{Name: "task.go", Lang: "go", Code: crySampleGo},
	{Name: "task_test.go", Lang: "go", Code: "func TestTask(t *testing.T) {\n\tt := Task{ID: \"T-001\"}\n\tif t.ID == \"\" {\n\t\tt.Fatal(\"empty id\")\n\t}\n}"},
	{Name: "README.md", Lang: "markdown", Code: "# bestuav\n\n无人机业务管理系统。\n"},
}

// cryLogEntries 控制台演示日志。
var cryLogEntries = []code.LogEntry{
	{Level: code.LevelLog, Text: "bestuav-mygo listening on :8080", Source: "main.go:31"},
	{Level: code.LevelLog, Text: "closed 214 架次任务", Source: "task.go:42"},
	{Level: code.LevelWarn, Text: "电量阈值偏低，已切换保守航线", Source: "drone.go:88"},
	{Level: code.LevelLog, Text: "推送 214 条任务回执", Source: "sync.go:17"},
	{Level: code.LevelError, Text: "stripe: card_declined for ws_8f2k", Source: "charge.go:63"},
}

// cryDiffState 差异视图的演示数据。
var cryDiffState = &editor.DiffState{
	Old: editor.NewBuffer("func Plan(name string) string {\n\treturn \"plan: \" + name\n}\n"),
	New: editor.NewBuffer("func Plan(name string, seats int) string {\n\treturn fmt.Sprintf(\"plan %s x %d\", name, seats)\n}\n"),
}

// crychicDatetimePage 日期时间组件对照页。
func (a *App) crychicDatetimePage(c *ui.Context, t *ui.Theme) {
	ui.Text(c, "对照：组件演示 → 输入框组件（手搓的日期输入 vs 这里的 CrychicUI datetime 包）").
		FontSize(t.FontSize - 2).TextColor(t.TextMuted)

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "日期 / 时间选择器", "DatePicker · TimePicker · DateTimePicker", nil)
		ui.Row(c).Gap(16).Children(func() {
			datetime.DatePicker(c, "到期日", &a.cryPick)
			datetime.TimePicker(c, "提醒时间", &a.cryPick, datetime.TimePickerOptions{})
			datetime.DateTimePicker(c, "会议时间", &a.cryMini, datetime.DateTimePickerOptions{})
		})
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "日历与时钟", "MiniCalendar · DigitalClock · AnalogClock", nil)
		ui.Row(c).Gap(24).Children(func() {
			datetime.MiniCalendar(c, &a.cryMini,
				[]time.Time{a.cryMini.AddDate(0, 0, 2), a.cryMini.AddDate(0, 0, 5), a.cryMini.AddDate(0, 0, -4)},
				datetime.MiniCalendarOptions{})
			ui.Column(c).Gap(16).Children(func() {
				datetime.DigitalClock(c, datetime.DigitalClockOptions{Seconds: true})
				datetime.AnalogClock(c, datetime.AnalogClockOptions{})
			})
		})
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "热度图 / 徽标 / 相对时间", "CalendarHeat · DateBadge · DateNav · TimeAgo", nil)
		ui.Column(c).Gap(16).Children(func() {
			// 12 周活跃度（确定性数据，便于复现）。
			counts := make([]int, 84)
			for i := range counts {
				counts[i] = (i*3 + i%4) % 6
			}
			datetime.CalendarHeat(c, "提交活跃度", a.cryHeatFrom, counts, datetime.CalendarHeatOptions{})
			ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
				datetime.DateBadge(c, a.cryPick)
				datetime.DateNav(c, &a.cryNav, datetime.DateNavOptions{})
				datetime.TimeAgo(c, a.cryHeatFrom, datetime.TimeAgoOptions{})
			})
		})
	})
}

// crychicCodePage 代码组件对照页。
func (a *App) crychicCodePage(c *ui.Context, t *ui.Theme) {
	ui.Text(c, "对照：官方示例 → 代码编辑器（自研代码编辑器 vs 这里的 CrychicUI code 包）").
		FontSize(t.FontSize - 2).TextColor(t.TextMuted)

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "代码块 / JSON 格式化", "CodeBlock · JSONFormat", nil)
		ui.Column(c).Gap(16).Children(func() {
			code.CodeBlock(c, "Go", crySampleGo, code.CodeBlockOptions{Title: "bestuav/task.go"})
			code.JSONFormat(c, "config.json", &a.cryJSON, code.JSONFormatOptions{})
		})
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "代码标签页", "CodeTabs（多文件切换）", nil)
		code.CodeTabs(c, cryCodeFiles, &a.cryCodeTab)
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "控制台", "Console（按级别过滤）", nil)
		code.Console(c, "运行日志", cryLogEntries, &a.cryConsoleFilter, code.ConsoleOptions{})
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "命令行", "CommandLine（历史 + 输出）", nil)
		_, _ = code.CommandLine(c, "zsh — bestuav", a.cryCmdHistory, &a.cryCmdDraft,
			func(s string) {
				a.cryCmdHistory = append(a.cryCmdHistory, code.Command{Text: s, Output: "ok: " + s})
			}, code.CommandLineOptions{Height: 200})
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "差异视图", "DiffView（旧 → 新）", nil)
		code.DiffView(c, cryDiffState, &a.crySplit, code.DiffViewOptions{Height: 280, Name: "plan.go"})
	})
}
