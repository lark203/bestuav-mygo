package view

import (
	"sync"
	"time"

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
// build 为 nil 时自动落到占位页（hint 作为说明文案），可以先占位再实现；页面需要
// 「页面级设置」时再补 settings: (*App).orderSettings（会出现在「设置 → 页面」里）。
// 更省事的做法是 `go run ./scripts/newpage ...`（见 internal/pagegen）。

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
	// hidden 为 true 时不在侧栏出现，但仍可按 id 分发、显示页头标题、参与生命周期。
	//
	// 用途：只能从别处跳进来的页面（详情页、向导页），不该占用侧栏位置。
	// 配合 back 使用——隐藏页通常需要「从哪来回哪去」。
	hidden bool
	// back 非空时，页头标题左侧自动出现返回键，点击跳到该页面 id。
	//
	// 为什么放在注册表而不是页面里：返回目标是「这一页在导航里的位置」，属于页面
	// 声明；写在这里，骨架（shell）就能统一画返回键，页面函数只写内容。
	back string
	// settings 页面级设置项（可空）：在「设置 → 页面」里渲染成一个区块。
	//
	// 设置项自己用 a.PrefsFor(id) 读写页面私有状态（见 prefs.go），设置页不需要
	// 认识任何具体字段；没有任何页面声明 settings 时，「页面」这一组不会出现。
	settings pageFunc
}

// navGroup 侧栏里的一个可折叠分组。
type navGroup struct {
	title string
	items []navItem
}

// navRegistry 是全部菜单与页面的定义。侧栏、页头、页面分发都由它驱动。
//
// 注意：这里必须保持「直接 return 一个 []navGroup 字面量」的写法——生成器
// （internal/scaffold/regcut.go）靠 AST 在这个字面量上做增删，改成先算再 return
// 会让「新建项目」的菜单裁剪失效（scaffold 的 TestCutRegistry 会拦住）。
// 每帧的查询走 navAll() 里一次性建好的缓存，不要直接调本函数。
func navRegistry() []navGroup {
	return []navGroup{
		// 首页不放进分组（侧栏顶部第一项，无分组标题）。
		// build 指向欢迎页：新项目刚生成时没有业务数据，首页承担「下一步做什么」的
		// 引导；原来的运行概览降级为最末分组里的一页（见下方「概览示例」）。
		{title: "", items: []navItem{
			{id: "home", label: "首页", icon: icons.Home, build: (*App).welcomePage,
				policy: pagePolicy{keep: keepPinned}}, // 首页常驻，不必重建
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
			// 新建项目：以本工程为基调生成新项目。生成后该菜单会从产物里消失
			// （scaffold 按 id 摘掉本项并删除本页文件），不会自我复制到下一代。
			{id: "new-project", label: "新建项目", icon: icons.Package,
				build:  (*App).newProjectPage,
				policy: pagePolicy{keep: keepCached}}, // 生成结果要留着看，故缓存
		}},

		{title: "组件演示", items: []navItem{
			{id: "comp-buttons", label: "按钮组件", icon: icons.Square, build: (*App).buttonsDemoPage},
			{id: "comp-inputs", label: "输入框组件", icon: icons.Type, build: (*App).inputsDemoPage},
			{id: "comp-table", label: "表格组件", icon: icons.Table, build: (*App).tableDemoPage},
			{id: "comp-card", label: "卡片组件", icon: icons.IdCard, build: (*App).cardDemoPage},
			{id: "comp-tabs", label: "标签页组件", icon: icons.Panel, build: (*App).tabsDemoPage},
		}},

		{title: "布局示例", items: []navItem{
			// 内容区三种公共样式（页头扩展 / tab / 左右两栏）的活文档。
			{id: "layout-tabs", label: "标签页布局", icon: icons.Panel,
				build: (*App).layoutTabsPage, policy: pagePolicy{keep: keepCached}},
			{id: "layout-split", label: "主从布局", icon: icons.List,
				build: (*App).layoutSplitPage, policy: pagePolicy{keep: keepCached}},
		}},

		{title: "CrychicUI 组件", items: []navItem{
			{id: "crychic-datetime", label: "日期时间组件", icon: icons.Clock, build: (*App).crychicDatetimePage},
			{id: "crychic-code", label: "代码组件", icon: icons.Code, build: (*App).crychicCodePage},
		}},

		{title: "官方示例", items: []navItem{
			// 从 mygo 官方 examples 移植（见 page_loading.go 等文件顶部的说明）。
			{id: effortPageID, label: "投入滑块（自绘动画）", icon: icons.Sliders,
				// 整张卡片是自绘的：档位与动画时钟留在内存里，切回来接着动。
				// 用 keepPinned 而不是 keepCached：滑块构造要跑一次离屏渲染（帧内
				// 不能做），只能在启动时建好，故不能被闲置回收掉。
				policy: pagePolicy{keep: keepPinned},
				build:  (*App).effortPage},
			{id: "official-blackhole", label: "黑洞（GPU 特效）", icon: icons.Atom,
				// 着色器与两块画布比较重：离开即销毁。场景对象存在页面私有状态里，
				// destroyPage 丢弃 data 时 GPU 资源随之释放，不需要额外的 onDestroy。
				policy: pagePolicy{keep: keepFresh},
				build:  (*App).blackholePage},
			{id: "official-piano", label: "钢琴（合成音）", icon: icons.Music,
				// 离开页面先停曲子（否则后台还在弹）；键盘状态本身可以留着，
				// 故缓存而非每次重建。
				policy: pagePolicy{
					keep:      keepCached,
					onLeave:   func(a *App) { a.stopPiano() },
					onDestroy: func(a *App) { a.closePiano() },
				},
				build: (*App).pianoPage},
			{id: "official-clipboard", label: "多格式剪贴板", icon: icons.Copy,
				// 编辑中的笔记要留着；剪贴板内容归系统，页面销毁也不必清理。
				policy: pagePolicy{keep: keepCached},
				build:  (*App).clipboardPage},
			{id: "official-text-buffer", label: "大文档编辑（索引缓冲）", icon: icons.FileText,
				// 文档本体有 4 MB：缓存让「切回来还在编辑」，销毁时才真正释放。
				policy: pagePolicy{keep: keepCached},
				build:  (*App).textBufferPage},
			{id: "official-text-input", label: "自绘输入框（文本原语）", icon: icons.Pencil,
				// 输入中的文本与选区要「切回来还在」，缓存即可。
				policy: pagePolicy{keep: keepCached},
				build:  (*App).textInputPage},
			{id: "official-flyout", label: "浮出面板", icon: icons.Popover,
				// 浮层是独立子窗口：离开页面必须先关掉，否则会留下孤儿窗口。
				// 颜色/便签本身无所谓，但句柄必须清，故整页销毁。
				policy: pagePolicy{keep: keepFresh, onDestroy: func(a *App) { a.closeFlyouts() }},
				build:  (*App).flyoutPage},
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
			{id: asyncDemoPageID, label: "异步数据（加载/重试）", icon: icons.Refresh,
				build: (*App).asyncDemoPage, settings: (*App).asyncDemoSettings},
		}},

		// 概览示例：首页原本的内容（KPI + 趋势图 + 近期任务表）。放到最后、
		// 且默认在「新建项目」里不勾选——它是一份可以照着写的布局参考，
		// 不是新项目必须的页面。
		{title: "概览示例", items: []navItem{
			{id: overviewPageID, label: "运行概览", icon: icons.Chart, build: (*App).homePage,
				policy: pagePolicy{keep: keepCached,
					// 每次进入都重新滚一遍 KPI 数字：onEnter 里只能把起点清零，
					// 真正的起点由 homePage 在帧内取（要用帧时钟，见 kit.CountUp）。
					onEnter: func(a *App) { a.kpiStart = time.Time{} },
				},
				tools: []toolAction{
					{icon: icons.Refresh, label: "刷新", tip: "刷新概览数据（演示轻量任务进度条）"},
					{icon: icons.Search, label: "搜索", tip: "全局搜索（演示）"},
					{icon: icons.Download, label: "导出", tip: "导出概览报表（演示）"},
					{icon: icons.More, label: "更多", tip: "更多操作（演示）"},
				}},
		}},
	}
}

// overviewPageID 「概览示例」分组的运行概览页 id。
//
// 这一页属于可裁剪分组：新项目勾掉「概览示例」后它就不存在了，所以欢迎页里那个
// 入口按钮要先按注册表判断再决定画不画（见 page_welcome.go）。
const overviewPageID = "home-overview"

// settingsPageID 底部固定「设置」页的 id（页面文件、欢迎页、单实例转发共用）。
const settingsPageID = "settings"

// settingsPageLabel 「设置」项的侧栏文字与页头标题。
const settingsPageLabel = "设置"

// settingsNavIcon 「设置」项的图标（侧栏底部与注册表共用）。
//
// 单独提出来是为了让 app.go 的侧栏底部不必调用 settingsPageItem()（那会每帧构造
// 一个 navItem 与它的策略闭包）。
var settingsNavIcon = icons.Gear

// settingsOnEnter 进入设置页时后台检查更新（见 settingsPageItem 的说明）。
func settingsOnEnter(a *App) { a.checkForUpdates(false) }

// settingsPageItem 底部固定的「设置」项：不放在任何分组里（侧栏单独渲染在底部），
// 但它同样需要参与「页头标题 / 页面分发 / 生命周期」的查找，故并进 eachPage。
//
// 为什么用函数而不是包级 var：设置页会经 pageSettingItems → navAll 间接引用本项，
// 包级 var 的初始化器会形成「初始化环」（Go 会静态检查这种间接引用）。
func settingsPageItem() navItem {
	return navItem{
		id: settingsPageID, label: settingsPageLabel, icon: settingsNavIcon,
		build: (*App).settingsPage,
		// 进入设置即后台检查更新（onEnter 只在切到本页那一帧触发，启动默认页是 home
		// 故不会误触发）。检查由 goroutine 异步完成，UI 显示「正在检查」再到结果。
		//
		// 非阻塞 goroutine，10 分钟内不重复查；用户「忽略」过的版本记在
		// settings.IgnoredVersion 里，检查结果回来时会被重新标记为已忽略；
		// 手动点标题栏图标（force）才重新显示提示。
		policy: pagePolicy{keep: keepPinned, onEnter: settingsOnEnter},
	}
}

// navAll 返回一次性构建的注册表切片与 id → 页面 的索引。
//
// 侧栏、页头标题、页头工具条和闲置淘汰每帧都要查这张表；每次重建五个分组
// 三十来个 item 属于纯浪费（实测单次 9.9µs / 6KB，已访问十个页面后每帧十几次）。
func navAll() ([]navGroup, map[string]navItem) {
	navOnce.Do(func() {
		navGroupsCache = navRegistry()
		navIndex = make(map[string]navItem, 24)
		for _, g := range navGroupsCache {
			for _, it := range g.items {
				navIndex[it.id] = it
			}
		}
		// 设置项不放进任何分组（侧栏单独画在底部），但同样要能被按 id 找到。
		it := settingsPageItem()
		navIndex[it.id] = it
	})
	return navGroupsCache, navIndex
}

var (
	navOnce        sync.Once
	navGroupsCache []navGroup
	navIndex       map[string]navItem
)

// eachPage 遍历注册表里的所有页面（含无分组标题的首页与底部设置项）。
func eachPage(fn func(navItem)) {
	groups, _ := navAll()
	for _, g := range groups {
		for _, it := range g.items {
			fn(it)
		}
	}
	fn(settingsPageItem())
}

// pageByID 按 id 找页面定义（O(1)，走一次性建好的索引）。
func pageByID(id string) (navItem, bool) {
	_, idx := navAll()
	it, ok := idx[id]
	return it, ok
}

// pageSettingItems 返回声明了「页面级设置」的菜单项（顺序即注册表顺序）。
//
// 「设置 → 页面」与 settingsGroupLabels() 都靠它决定是否显示那一组。数量很少
// （页面数级别），每次重建切片没有性能问题；只有设置页会调用。
func pageSettingItems() []navItem {
	groups, _ := navAll()
	var out []navItem
	for _, g := range groups {
		for _, it := range g.items {
			if it.settings != nil {
				out = append(out, it)
			}
		}
	}
	if it := settingsPageItem(); it.settings != nil {
		out = append(out, it)
	}
	return out
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
//
// 结果是只读的常量数据，构建一次即可：侧栏每帧都要用它，没必要每帧重建一遍
// 分组与 item 切片（那又是十几个分配）。
func navGroups() []shell.Group {
	if navShell != nil {
		return navShell
	}
	navShell = navGroupsFrom(navRegistry())
	return navShell
}

// navGroupsFrom 把注册表投影成侧栏数据：跳过 hidden，整组都是隐藏页时该组不出现。
//
// 单独成函数是为了可测试：navGroups 只能读真实注册表，而「隐藏项不从侧栏出现」
// 这条规则需要能用手造的分组来验证（不需要为了测试往注册表里塞假页面）。
func navGroupsFrom(reg []navGroup) []shell.Group {
	out := make([]shell.Group, 0, len(reg))
	for _, g := range reg {
		items := make([]shell.Item, 0, len(g.items))
		for _, it := range g.items {
			if it.hidden {
				continue
			}
			items = append(items, shell.Item{ID: it.id, Label: it.label, Icon: it.icon})
		}
		// 全隐藏的分组没有可显示项：留着只会出现一个空标题。
		if len(items) == 0 && g.title != "" {
			continue
		}
		out = append(out, shell.Group{Title: g.title, Items: items})
	}
	return out
}

// navShell 是 navGroups 的缓存（只由主线程在首帧构建）。
var navShell []shell.Group

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
