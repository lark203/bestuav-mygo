package view

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 浮出面板（Flyout）是**独立子窗口**，离屏渲染不到它 —— 真机点开才能看见。
// 这里能测的只有两件事：页面骨架能渲染出来，以及那些「不碰窗口」的纯状态逻辑。

// TestFlyoutHoverWraps 验证方向键在列表两端循环（取模不能出负数）。
func TestFlyoutHoverWraps(t *testing.T) {
	f := &flyoutDemo{color: 2, hover: 2}
	n := len(flyoutColors)

	// 从任意起点走 n 步必须回到原地。
	for start := 0; start < n; start++ {
		f.hover = start
		for i := 0; i < n; i++ {
			f.move(1)
		}
		if f.hover != start {
			t.Errorf("从 %d 向下走 %d 步回到 %d，期望 %d", start, n, f.hover, start)
		}
	}
	// 越界：0 往上走一步应是最后一项，且不能是负数。
	f.hover = 0
	f.move(-1)
	if f.hover != n-1 {
		t.Errorf("0 往上一步 = %d，期望 %d（不能为负）", f.hover, n-1)
	}
	// 多步负增量同样要落回合法区间。
	f.hover = 1
	f.move(-3)
	if f.hover != (n-2)%n {
		t.Errorf("1 往上三步 = %d，期望 %d", f.hover, (n-2)%n)
	}
	for i := -2 * n; i <= 2*n; i++ {
		f.hover = 0
		f.move(i)
		if f.hover < 0 || f.hover >= n {
			t.Fatalf("增量 %d 让高亮越界：%d", i, f.hover)
		}
	}
}

// TestFlyoutChoose 验证选中写入 color，且越界下标被忽略（防御回调传入坏值）。
func TestFlyoutChoose(t *testing.T) {
	f := &flyoutDemo{color: 0, place: flyoutPlacements[0].label}
	const pick = 4
	f.choose(newTestApp(), pick)
	if f.color != pick {
		t.Errorf("选中 %d 后 color = %d", pick, f.color)
	}
	if want := "已选颜色 " + flyoutColors[pick].name; f.last != want {
		t.Errorf("状态行 = %q，期望 %q", f.last, want)
	}
	for _, bad := range []int{-1, len(flyoutColors), 99} {
		f.choose(newTestApp(), bad)
		if f.color != pick {
			t.Errorf("越界下标 %d 不该改变 color，现在 = %d", bad, f.color)
		}
	}
}

// TestFlyoutPlacements 验证下拉文案能换回 Placement，且未知文案退回默认。
func TestFlyoutPlacements(t *testing.T) {
	labels := flyoutPlaceLabels()
	if len(labels) != len(flyoutPlacements) {
		t.Fatalf("选项数 %d 与方位表 %d 不一致", len(labels), len(flyoutPlacements))
	}
	for i, want := range flyoutPlacements {
		if labels[i] != want.label {
			t.Errorf("第 %d 项文案 = %q，期望 %q", i, labels[i], want.label)
		}
		if got := flyoutPlacement(labels[i]); got != want.place {
			t.Errorf("文案 %q → %q，期望 %q", labels[i], got, want.place)
		}
		if got := flyoutPlacementLabel(labels[i]); got != want.label {
			t.Errorf("展示名 %q → %q", labels[i], got)
		}
	}
	if got := flyoutPlacement("不存在的方位"); got != mygo.PlacementBottomStart {
		t.Errorf("未知文案应退回 PlacementBottomStart，实际 %q", got)
	}
	if got := flyoutPlacementLabel("不存在的方位"); got != "默认" {
		t.Errorf("未知文案的展示名应退回「默认」，实际 %q", got)
	}
}

// TestAnchorRect 验证元素矩形到锚点的换算（Flyout 的锚点是整数 DIP）。
func TestAnchorRect(t *testing.T) {
	r := anchorRect(ui.Rect{X: 12.7, Y: 30.2, W: 88.9, H: 24.4})
	want := mygo.Rectangle{X: 12, Y: 30, Width: 88, Height: 24}
	if r != want {
		t.Errorf("anchorRect = %+v，期望 %+v（向下取整到整数 DIP）", r, want)
	}
	if anchorRect(ui.Rect{}) != (mygo.Rectangle{}) {
		t.Error("零矩形应换成零锚点（可表示一个点，如指针位置）")
	}
}

// TestFlyoutNoteText 验证空便签给占位文案（浮层里不能显示一片空白）。
func TestFlyoutNoteText(t *testing.T) {
	if got := (&flyoutDemo{}).noteText(); got != "（便签为空）" {
		t.Errorf("空便签 = %q，期望占位文案", got)
	}
	f := &flyoutDemo{note: "记得充电"}
	if got := f.noteText(); got != "记得充电" {
		t.Errorf("有内容时应原样显示，实际 %q", got)
	}
}

// TestFlyoutWithoutWindow 验证没有窗口句柄时（离屏渲染 / 单测）点按钮不会崩，
// 而是把原因写进状态行 —— 否则真机之外一跑就 panic。
func TestFlyoutWithoutWindow(t *testing.T) {
	a := newTestApp()
	if a.win != nil {
		t.Fatal("newTestApp 不应持有窗口句柄")
	}
	// 借离屏渲染跑一帧页面：既验证了骨架，也完成 fo 的初始化。
	ui.NewTester(func(c *ui.Context) { a.flyoutPage(c, c.Theme()) }, 900, 600)
	if a.fo == nil {
		t.Fatal("进入页面应初始化 fo")
	}
	a.fo.toggleList(a, ui.Rect{X: 10, Y: 10, W: 80, H: 24})
	if a.fo.list != nil {
		t.Error("没有父窗口时不应创建浮层窗口")
	}
	if want := "未注入窗口句柄"; !strings.Contains(a.fo.last, want) {
		t.Errorf("状态行 = %q，应包含 %q", a.fo.last, want)
	}
	// 另外两个入口在无句柄时也必须静默返回，不能 panic。
	a.fo.openEditor(a, ui.Rect{})
	a.fo.openAbout(a, ui.Rect{})
	if a.fo.editor != nil || a.fo.about != nil {
		t.Error("无句柄时便签 / 关于都不该被创建")
	}
	// closeFlyouts 在无句柄时也要安全（页面离开会调）。
	a.closeFlyouts()
	if a.fo != nil {
		t.Error("closeFlyouts 后 fo 应为 nil")
	}
}

// TestFlyoutPageRenders 验证页面骨架能在离屏环境渲染出来（浮层本身测不到）。
func TestFlyoutPageRenders(t *testing.T) {
	a := newTestApp()
	a.page = "official-flyout"
	mustHave(t, render(a),
		"浮出面板（Flyout）", // 卡片标题
		"编辑便签…", "关于",  // 三个触发按钮
		"（便签为空）",         // 便签占位
		"与对话框（Modal）的区别", // 说明卡
		"越界",             // 说明卡里的一行
	)
}
