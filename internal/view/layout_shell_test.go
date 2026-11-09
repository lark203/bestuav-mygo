package view

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
	"bestuav-mygo/internal/view/shell"
)

// TestHeaderExtrasKeepHeaderHeight 页头 Extras 再高也不能改变页头高度：
// 正文起始位置应仍在 52px 页头 + 3px 分割线附近。
func TestHeaderExtrasKeepHeaderHeight(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		shell.Content(c, c.Theme(), shell.ContentOptions{
			Title: "标题",
			Extras: func() {
				ui.Box(c).Width(40).Height(200) // 故意比页头高
			},
			Fill: true,
			Body: func() {
				ui.Text(c, "正文起点").Padding(0, 0)
			},
		})
	}, 800, 400)
	tt.Frame()
	r, ok := tt.Find("正文起点")
	if !ok {
		t.Fatal("找不到正文")
	}
	if r.Y > shell.HeaderHeight+12 {
		t.Errorf("正文被 Extras 推下去了：y=%.0f（页头应固定 %d）", r.Y, shell.HeaderHeight)
	}
}

// TestHeaderTabsHideSubtitle 有 tab 时不显示副标题，但 Extras 仍保留。
func TestHeaderTabsHideSubtitle(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		shell.Content(c, c.Theme(), shell.ContentOptions{
			Title:    "标题",
			Subtitle: "副标题不该出现",
			Extras:   func() { ui.Text(c, "页头组件") },
			Tabs:     []string{"A", "B"},
			Tab:      0,
			Fill:     true,
			Body:     func() { ui.Text(c, "正文") },
		})
	}, 900, 400)
	tt.Frame()
	if tt.HasText("副标题不该出现") {
		t.Error("有 tab 时不应渲染副标题")
	}
	if !tt.HasText("页头组件") {
		t.Error("有 tab 时仍应保留页头组件")
	}
}

// TestLayoutTabsPageSwitches 页头 tab 点击切换正文（tab 由 shell 渲染）。
func TestLayoutTabsPageSwitches(t *testing.T) {
	a := newApp(settings.Default())
	a.page = "layout-tabs"
	a.demoTab = 0
	tt := ui.NewTester(a.View, 1000, 700)
	tt.Frame()
	if err := tt.Click("明细"); err != nil {
		t.Fatalf("点击 tab 失败: %v", err)
	}
	if a.demoTab != 1 {
		t.Fatalf("点击「明细」后 demoTab 应为 1，实际 %d", a.demoTab)
	}
	tt.Frame()
	if !tt.HasText("当前 tab") {
		t.Errorf("切换后正文没有变成明细；文本：%v", tt.Texts())
	}
}

// TestHeaderSubtitleBottomAligned 副标题与标题底部对齐（不是垂直居中）。
func TestHeaderSubtitleBottomAligned(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		shell.Content(c, c.Theme(), shell.ContentOptions{
			Title:    "标题",
			Subtitle: "副标题",
			Fill:     true,
			Body:     func() {},
		})
	}, 600, 200)
	tt.Frame()
	tr, ok1 := tt.Find("标题")
	sr, ok2 := tt.Find("副标题")
	if !ok1 || !ok2 {
		t.Fatalf("找不到标题/副标题：%v %v", ok1, ok2)
	}
	if d := (tr.Y + tr.H) - (sr.Y + sr.H); d > 1.5 || d < -1.5 {
		t.Errorf("标题与副标题底部相差 %.1fpx（应底部对齐）", d)
	}
}

// TestSplitDragResizes 拖动分隔线调整左右宽度，并受最小/最大范围约束。
func TestSplitDragResizes(t *testing.T) {
	a := newApp(settings.Default())
	a.page = "layout-split"
	a.demoSel = 0
	a.split.Width = 0
	tt := ui.NewTester(a.View, 1200, 800)
	tt.Frame()
	base := a.split.Width
	if base != 320 {
		t.Fatalf("默认左栏宽度应为 320，实际 %.0f", base)
	}
	x := float32(220) + base + 3 // 侧栏展开宽 + 左栏宽 + 手柄一半
	tt.Press(x, 400)
	tt.Frame()
	tt.Move(x+120, 400)
	tt.Frame()
	tt.Release(x+120, 400)
	tt.Frame()
	if a.split.Width < base+80 {
		t.Fatalf("向右拖动后左栏应变宽，实际 %.0f", a.split.Width)
	}
	// 拖到最右：不超过 520。
	tt.Press(float32(220)+a.split.Width+3, 400)
	tt.Frame()
	tt.Move(2000, 400)
	tt.Frame()
	tt.Release(2000, 400)
	tt.Frame()
	if a.split.Width > 520.5 {
		t.Fatalf("左栏超过最大宽度：%.0f", a.split.Width)
	}
	// 拖到最左：不小于 240。
	tt.Press(float32(220)+a.split.Width+3, 400)
	tt.Frame()
	tt.Move(0, 400)
	tt.Frame()
	tt.Release(0, 400)
	tt.Frame()
	if a.split.Width < 239.5 {
		t.Fatalf("左栏小于最小宽度：%.0f", a.split.Width)
	}
}

// TestLayoutSplitPage 三段式：未选中时右栏为空态，选中后展示详情。
func TestLayoutSplitPage(t *testing.T) {
	a := newApp(settings.Default())
	a.page = "layout-split"
	a.demoSel = 0
	tt := ui.NewTester(a.View, 1100, 720)
	tt.Frame()
	if !tt.HasText("从左侧选择一个项目") {
		t.Fatalf("未选中时应展示空态提示；文本：%v", tt.Texts())
	}
	if got := a.pageSubtitle(); got != "未选择" {
		t.Errorf("未选中时副标题应为「未选择」，实际 %q", got)
	}
	if err := tt.Click("量子科学桌面端"); err != nil {
		t.Fatalf("点击左侧卡片失败: %v", err)
	}
	if a.demoSel != 2 {
		t.Fatalf("点击第二项后 demoSel 应为 2，实际 %d", a.demoSel)
	}
	tt.Frame()
	if !tt.HasText("数据来源") {
		t.Errorf("选中后右栏应展示详情；文本：%v", tt.Texts())
	}
	if got := a.pageSubtitle(); got != "量子科学桌面端" {
		t.Errorf("选中后副标题应为选中项名称，实际 %q", got)
	}
}

// TestSettingsTabsInHeader 设置页分组 tab 在页头而不是正文里。
func TestSettingsTabsInHeader(t *testing.T) {
	a := newApp(settings.Default())
	a.page = settingsPageID
	labels, idx, onTab := a.pageTabs()
	if len(labels) == 0 || onTab == nil {
		t.Fatal("设置页应提供页头 tab")
	}
	if idx != a.settingsGroup() {
		t.Errorf("tab 下标应与设置分组一致")
	}
	tt := ui.NewTester(a.View, 1200, 800)
	tt.Frame()
	if !tt.HasText(labels[0]) {
		t.Errorf("页头应显示 tab %q", labels[0])
	}
}
