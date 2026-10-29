package view

import (
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/icons"
	"bestuav-mygo/internal/view/shell"
)

// 本文件是「菜单 + 页面 + 页头按钮」的**唯一注册处**。
//
// 旧做法要在三处同步改：sidebar.go 的 navItemDef、app.go 的 pageTitles、
// 以及页面分发 switch——漏改一处就会出现「点了菜单没反应」「页头标题空白」这类
// 假 bug。现在统一成一张表：加一条 = 多一个菜单项 + 多一个可用页面。
//
// 新增一个菜单/页面：
//
//	1) 写页面函数（新建文件或加到已有文件）：
//	     func (a *App) orderPage(c *ui.Context, t *ui.Theme) { ... }
//	2) 在下面 navRegistry() 的对应分组里加一条：
//	     {id: "order", label: "订单管理", icon: icons.Table, build: (*App).orderPage,
//	      tools: []toolAction{{icon: icons.Search, label: "搜索", tip: "搜索订单"}}}
//
// build 为 nil 时自动落到占位页（hint 作为说明文案），可以先占位再实现。

// pageFunc 页面内容构建函数：签名与 App 方法一致，可直接写 (*App).xxx。
type pageFunc func(a *App, c *ui.Context, t *ui.Theme)

// navItem 一个菜单项 = 一个页面。
type navItem struct {
	id     string       // 页面 id，同时是选中态判断依据
	label  string       // 侧栏文字 + 页头标题
	icon   *ui.SVG      // 侧栏图标（16px 线性 SVG）
	tools  []toolAction // 页头右侧图标按钮（可空）
	build  pageFunc     // 页面内容（nil → 占位页）
	hint   string       // 占位页的一句话说明
	policy pagePolicy   // 生命周期与缓存策略（零值 = 缓存 + 默认闲置销毁）
}

// navGroup 侧栏里的一个可折叠分组。
type navGroup struct {
	title string
	items []navItem
}

// navRegistry 是全部菜单与页面的定义。侧栏、页头、页面分发都由它驱动。
func navRegistry() []navGroup {
	return []navGroup{
		// 首页不放进分组（侧栏顶部第一项，无分组标题）。
		{title: "", items: []navItem{
			{id: "home", label: "首页", icon: icons.Home, build: (*App).homePage,
				policy: pagePolicy{keep: keepPinned}, // 首页常驻，不必重建
				tools: []toolAction{
					{icon: icons.Refresh, label: "刷新", tip: "刷新概览数据（演示轻量任务进度条）"},
					{icon: icons.Search, label: "搜索", tip: "全局搜索（演示）"},
					{icon: icons.Download, label: "导出", tip: "导出概览报表（演示）"},
					{icon: icons.More, label: "更多", tip: "更多操作（演示）"},
				}},
		}},

		{title: "工作台", items: []navItem{
			{id: "task", label: "异步任务", icon: icons.Bolt, build: (*App).taskPage,
				// 任务列表要"切回来还在"，用缓存；闲置 30 分钟再由全局规则带走。
				policy: pagePolicy{keep: keepCached},
				tools: []toolAction{
					{icon: icons.Bolt, label: "轻量任务", tip: "运行一个轻量级任务（进度显示在页头浅线处）"},
					{icon: icons.Sparkles, label: "演示彩带", tip: "撒一把彩带看看效果"},
				}},
			{id: "gen-license", label: "生成 license", icon: icons.FileKey, build: (*App).licensePage,
				// 表单类页面：每次进入都要一份干净的表单，故不缓存。
				policy: pagePolicy{keep: keepFresh},
				tools: []toolAction{
					{icon: icons.Refresh, label: "重置表单", tip: "清空当前表单"},
					{icon: icons.Copy, label: "复制结果", tip: "复制最近一次生成的 license"},
				}},
		}},

		{title: "组件演示", items: []navItem{
			{id: "comp-buttons", label: "按钮组件", icon: icons.Square, build: (*App).buttonsDemoPage},
			{id: "comp-inputs", label: "输入框组件", icon: icons.Type, build: (*App).inputsDemoPage},
			{id: "comp-table", label: "表格组件", icon: icons.Table, build: (*App).tableDemoPage},
			{id: "comp-card", label: "卡片组件", icon: icons.IdCard, build: (*App).cardDemoPage},
			{id: "comp-tabs", label: "标签页组件", icon: icons.Panel, build: (*App).tabsDemoPage},
		}},

		{title: "官方示例", items: []navItem{
			// 从 mygo 官方 examples 移植（见 page_loading.go 等文件顶部的说明）。
			{id: "official-blackhole", label: "黑洞（GPU 特效）", icon: icons.Atom,
				// 着色器与两块画布比较重：离开即销毁，顺便释放 GPU 资源。
				policy: pagePolicy{keep: keepFresh, onDestroy: func(a *App) { a.bh = nil }},
				build:  (*App).blackholePage},
			{id: "official-text-input", label: "自绘输入框（文本原语）", icon: icons.Pencil,
				// 输入中的文本与选区要「切回来还在」，缓存即可。
				policy: pagePolicy{keep: keepCached},
				build:  (*App).textInputPage},
			{id: "official-code-editor", label: "代码编辑器", icon: icons.Code,
				// 编辑中的缓冲区要「切回来还在」，缓存即可；示例文件只存在于内存。
				policy: pagePolicy{keep: keepCached},
				build:  (*App).codeEditorPage},
			{id: "official-loading", label: "加载指示器", icon: icons.Sparkles,
				build: (*App).loadingDemoPage},
			{id: "official-notes", label: "笔记（富文本）", icon: icons.Clipboard,
				// 编辑中的内容要「切回来还在」，缓存即可；落盘在 notes.json。
				policy: pagePolicy{keep: keepCached},
				build:  (*App).notesPage,
				tools: []toolAction{
					{icon: icons.Copy, label: "复制笔记", tip: "把当前笔记复制为 Markdown",
						run: func(a *App) { a.copyCurrentNote() }},
				}},
			{id: "official-chat", label: "Markdown 聊天", icon: icons.Chat,
				// 流式输出属于瞬态，离开即销毁，回来是干净的新会话。
				policy: pagePolicy{keep: keepFresh},
				build:  (*App).markdownChatPage},
		}},

		{title: "功能演示", items: []navItem{
			{id: "fn-confetti", label: "彩带庆祝", icon: icons.Sparkles, build: (*App).confettiDemoPage,
				policy: pagePolicy{keep: keepFresh}}, // 离开即销毁：彩带等瞬态效果不残留
			{id: "fn-chart", label: "滚动图表", icon: icons.Chart, build: (*App).chartDemoPage},
			{id: "fn-todo", label: "待办清单", icon: icons.Checklist, build: (*App).todoDemoPage},
			{id: "fn-vlist", label: "虚拟列表", icon: icons.List, build: (*App).vlistDemoPage},
			{id: "fn-timeline", label: "动画时间线", icon: icons.Clock, build: (*App).timelineDemoPage},
		}},
	}
}

// settingsNav 底部固定的「设置」项：不放在任何分组里（侧栏单独渲染在底部），
// 但它同样需要参与「页头标题 / 页面分发 / 生命周期」的查找，故并进 eachPage。
var settingsNav = navItem{
	id: "settings", label: "设置", icon: icons.Gear, build: (*App).settingsPage,
	policy: pagePolicy{keep: keepPinned},
}

// eachPage 遍历注册表里的所有页面（含无分组标题的首页与底部设置项）。
func eachPage(fn func(navItem)) {
	for _, g := range navRegistry() {
		for _, it := range g.items {
			fn(it)
		}
	}
	fn(settingsNav)
}

// pageByID 按 id 找页面定义。
func pageByID(id string) (navItem, bool) {
	var found navItem
	ok := false
	eachPage(func(it navItem) {
		if it.id == id && !ok {
			found, ok = it, true
		}
	})
	return found, ok
}

// pageTitle 页头标题：注册表里的 label，找不到时退回 id。
func pageTitle(id string) string {
	if it, ok := pageByID(id); ok {
		return it.label
	}
	return id
}

// pageToolsFor 页头右侧图标按钮：来自注册表，避免与菜单定义分家。
func pageToolsFor(id string) []toolAction {
	it, ok := pageByID(id)
	if !ok {
		return nil
	}
	return it.tools
}

// ---- 与架构骨架（internal/view/shell）的桥接 ----

// navGroups 把菜单注册表投影成侧栏骨架需要的分组数据。
//
// 注册表里的 navItem 还带着页面构建函数、工具条、生命周期策略等业务信息，
// 而侧栏只需要「id + 文字 + 图标」。做这次投影后，shell 包不必 import 业务类型，
// 也就不会出现包循环依赖。
func navGroups() []shell.Group {
	reg := navRegistry()
	out := make([]shell.Group, 0, len(reg))
	for _, g := range reg {
		items := make([]shell.Item, 0, len(g.items))
		for _, it := range g.items {
			items = append(items, shell.Item{ID: it.id, Label: it.label, Icon: it.icon})
		}
		out = append(out, shell.Group{Title: g.title, Items: items})
	}
	return out
}

// pageBody 渲染当前页面：注册表里有 build 就用它，否则落到占位页。
//
// 这是「菜单 → 页面」唯一的映射点：新增页面只需改本文件一处。
func (a *App) pageBody(c *ui.Context, t *ui.Theme) {
	if it, ok := pageByID(a.page); ok && it.build != nil {
		it.build(a, c, t)
		return
	}
	placeholderPage(c, t, pageTitle(a.page), a.page)
}
