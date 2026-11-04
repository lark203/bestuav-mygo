package view

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
)

// TestNavGroupsSkipHidden 隐藏页不从侧栏出现；整组都是隐藏页时该组也不出现。
func TestNavGroupsSkipHidden(t *testing.T) {
	groups := navGroupsFrom([]navGroup{
		{title: "", items: []navItem{{id: "home", label: "首页"}}},
		{title: "工作台", items: []navItem{
			{id: "task", label: "异步任务"},
			{id: "task-detail", label: "任务详情", hidden: true, back: "task"},
		}},
		{title: "只有隐藏页", items: []navItem{
			{id: "secret-detail", label: "隐藏详情", hidden: true, back: "home"},
		}},
	})

	if len(groups) != 2 {
		t.Fatalf("应只剩 2 组（首页组 + 工作台组），实际 %d：%+v", len(groups), groups)
	}
	if len(groups[1].Items) != 1 || groups[1].Items[0].ID != "task" {
		t.Errorf("工作台组应只剩 task，实际 %+v", groups[1].Items)
	}
}

// TestHiddenPageConventions 隐藏页必须声明 back，且 back 目标必须存在——
// 否则用户进得去出不来（侧栏没有它，页头也没返回键）。
func TestHiddenPageConventions(t *testing.T) {
	eachPage(func(it navItem) {
		if it.hidden && it.back == "" {
			t.Errorf("隐藏页 %q 没有声明 back，进去就出不来了", it.id)
		}
		if it.back != "" {
			if _, ok := pageByID(it.back); !ok {
				t.Errorf("页面 %q 的 back 指向不存在的页面 %q", it.id, it.back)
			}
		}
	})
}

// TestHeaderBackNavigates 页头返回键点了之后真的会跳页。
func TestHeaderBackNavigates(t *testing.T) {
	a := newApp(settings.Default())
	a.page = "task"

	tt := ui.NewTester(func(c *ui.Context) {
		a.headerBack(c, c.Theme(), "home")
	}, 320, 80)
	tt.Frame()
	if err := tt.Click("返回"); err != nil {
		t.Fatalf("点击返回键失败: %v", err)
	}
	tt.Frame()
	if a.page != "home" {
		t.Errorf("点击返回后应跳到 home，实际 %q", a.page)
	}
}

// TestPageLeadingOnlyForBackPages 没声明 back 的页面不该出现返回键。
func TestPageLeadingOnlyForBackPages(t *testing.T) {
	a := newApp(settings.Default())
	a.page = "home"
	tt := ui.NewTester(func(c *ui.Context) {
		if got := a.pageLeading(c, c.Theme()); got != nil {
			t.Error("home 没有 back，不该有返回键")
		}
	}, 240, 80)
	tt.Frame()
}

// TestRegistryIndex 钉住「注册表 → 索引」的一致性：
// 每个注册项都能按 id 查回同一份定义，且 id 不重复（重复会被 map 静默覆盖，
// 表现为「某个菜单项点进去是另一个页面」这种极难查的假 bug）。
func TestRegistryIndex(t *testing.T) {
	seen := map[string]bool{}
	count := 0
	eachPage(func(it navItem) {
		count++
		if seen[it.id] {
			t.Fatalf("页面 id 重复: %q", it.id)
		}
		seen[it.id] = true

		got, ok := pageByID(it.id)
		if !ok {
			t.Fatalf("%q 在索引里找不到", it.id)
		}
		if got.id != it.id || got.label != it.label {
			t.Fatalf("%q 查回来是 %+v", it.id, got)
		}
	})
	if count < 10 {
		t.Fatalf("注册页面只有 %d 个，疑似注册表构建失败", count)
	}
	if _, ok := pageByID("no-such-page"); ok {
		t.Fatal("不存在的 id 不应命中")
	}
}

// TestRegistryLookupAllocations 钉住「每次查找零分配」：
// 侧栏、页头、闲置淘汰每帧都要查十几次，重建注册表/投影曾被实测为单次 9.9µs、6KB。
func TestRegistryLookupAllocations(t *testing.T) {
	if got := testing.AllocsPerRun(200, func() { _, _ = pageByID("task") }); got != 0 {
		t.Errorf("pageByID 每次分配 %.1f 次，应为 0", got)
	}
	if got := testing.AllocsPerRun(200, func() { _ = navGroups() }); got != 0 {
		t.Errorf("navGroups 每次分配 %.1f 次，应为 0", got)
	}
	// navRegistry 本身仍是「返回字面量」的构建函数（生成器要靠 AST 改它），
	// 允许分配；每帧路径必须走 navAll/pageByID/navGroups 这几个缓存入口。
	if got := testing.AllocsPerRun(200, func() { _, _ = navAll() }); got != 0 {
		t.Errorf("navAll 每次分配 %.1f 次，应为 0", got)
	}
}

// TestPageTitleAndTools 检查页头标题与工具条的取数路径。
func TestPageTitleAndTools(t *testing.T) {
	if got := pageTitle("task"); got != "异步任务" {
		t.Errorf("task 的标题应为「异步任务」，实际 %q", got)
	}
	if got := pageTitle("no-such-page"); got != "no-such-page" {
		t.Errorf("未知页面应退回 id，实际 %q", got)
	}
	if len(pageToolsFor("task")) == 0 {
		t.Error("task 页应注册页头工具条")
	}
}
