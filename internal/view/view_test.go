package view

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/appmeta"
	"bestuav-mygo/internal/settings"
)

// newTestApp 用默认偏好构造 App：不读开发机的 settings.json，保证测试结果
// 与「本机是否收起过侧栏、是否设过背景图」无关。
func newTestApp() *App { return newApp(settings.Default()) }

func render(a *App) *ui.Tester { return ui.NewTester(a.View, 1100, 720) }

func mustHave(t *testing.T, tt *ui.Tester, want ...string) {
	t.Helper()
	for _, w := range want {
		if !tt.HasText(w) {
			t.Errorf("缺少文本 %q，当前文本: %v", w, tt.Texts())
		}
	}
}

// TestHomePage 验证默认进入首页，且首页具备产品级概览的四块内容。
func TestHomePage(t *testing.T) {
	a := newTestApp()
	mustHave(t, render(a),
		appmeta.Title,                   // 侧栏品牌头
		"首页",                            // 页头标题 + 侧栏选中项
		"今日任务", "任务成功率", "有效授权", "平均耗时", // 四个 KPI
		"近期任务", "服务状态", "快捷入口", // 三块主体
		"同步授权数据", "已完成", // 近期任务行
		"异步任务", "生成 license", "基本信息", // 侧栏与快捷入口
	)
}

// TestSidebarGroups 验证注册表里的分组都会被渲染出来，且「个人中心」已并入设置。
func TestSidebarGroups(t *testing.T) {
	a := newTestApp()
	tt := render(a)
	mustHave(t, tt, "工作台", "组件演示", "功能演示")
	if tt.HasText("个人中心") {
		t.Error("「个人中心」分组已移除，不应再出现在侧栏")
	}
}

// TestSidebarCollapsed 验证收缩态：只留图标，分组标题与菜单文字都不显示。
func TestSidebarCollapsed(t *testing.T) {
	a := newTestApp()
	a.sidebarCollapsed = true
	tt := render(a)
	// 注意：首页问候语里含产品名，故品牌文字的断言改用分组标题与菜单项。
	for _, w := range []string{"工作台", "组件演示", "按钮组件", "功能演示"} {
		if tt.HasText(w) {
			t.Errorf("收缩态不应显示文本 %q", w)
		}
	}
}

// TestSidebarCollapsedInitFromSettings 验证收缩状态由已保存偏好初始化。
func TestSidebarCollapsedInitFromSettings(t *testing.T) {
	s := settings.Default()
	s.SidebarCollapsed = true
	a := newApp(s)
	if !a.sidebarCollapsed {
		t.Fatal("应从已保存偏好初始化为收缩态")
	}
}

// TestSettingsPage 验证设置页分区域渲染（含页面缓存项）。
func TestSettingsPage(t *testing.T) {
	a := newTestApp()
	a.page = "settings"
	mustHave(t, render(a),
		"设置", "外观", "主题", "字体大小",
		"背景", "背景图片", "选择图片", "恢复默认", "菜单区", "内容区",
		"窗口与关闭", "关闭按钮行为",
		"启动", "允许多开", "页面缓存",
		"关于", "基本信息", "查看",
	)
}

// TestTaskPage 验证异步任务页的两类任务卡与任务队列表。
func TestTaskPage(t *testing.T) {
	a := newTestApp()
	a.page = "task"
	mustHave(t, render(a),
		"异步任务",
		"轻量级任务", "进度显示在页头下方浅线处",
		"重量级任务", "完成前禁止操作",
		"任务队列", "重建检索索引",
	)
}

// TestBasicInfoDialog 验证基本信息弹窗（原「个人中心」页）的页签与默认页内容。
func TestBasicInfoDialog(t *testing.T) {
	a := newTestApp()
	a.page = "settings"
	a.showBasicInfo = true
	tt := render(a)
	// 标题 + 五个页签 + 底部动作
	mustHave(t, tt,
		"基本信息",
		"应用", "环境", "缓存", "数据与日志", "配置",
		"复制全部信息", "关闭",
	)
	// 默认停在「应用」页：应用标识与构建版本可见
	mustHave(t, tt, "产品名称", "应用标识", appmeta.Identifier, "构建版本")
	// 其它页签的内容此刻不应渲染（分页签就是为了每屏只显示一组）
	if tt.HasText("Go 版本") {
		t.Error("「应用」页签下不应同时渲染「环境」页签的内容")
	}
}

// TestBasicInfoDialogEnvTab 验证切到「环境」页后显示运行时信息。
func TestBasicInfoDialogEnvTab(t *testing.T) {
	a := newTestApp()
	a.showBasicInfo = true
	a.basicTab = 1
	mustHave(t, render(a), "操作系统", "Go 版本", "逻辑核心", "堆内存")
}

// TestBasicInfoCacheTabListsAll 验证缓存页签把**所有**缓存页都列出来（不截断行数），
// 且不再出现「另有 N 个未列出」这类提示——多了应该在表体里滚动。
func TestBasicInfoCacheTabListsAll(t *testing.T) {
	a := newTestApp()
	a.showBasicInfo = true
	a.basicTab = 2

	// 把所有页面都标记为「已缓存」，模拟长时间使用后的状态。
	var labels []string
	eachPage(func(it navItem) {
		a.pageStat(it.id)
		labels = append(labels, it.label)
	})

	tt := render(a)
	for _, l := range labels {
		if !tt.HasText(l) {
			t.Errorf("缓存表应列出全部缓存页，缺少 %q", l)
		}
	}
	if tt.HasText("未列出") {
		t.Error("不应再出现「另有 N 个未列出」的截断提示")
	}
	mustHave(t, tt, "页面", "策略", "进入", "闲置", "滚动位置")
}

// TestCacheTabHeightFitsAnyFont 验证缓存页签的高度在任意字号下都成立：
// 表体高度必须等于「整行数 × 行高 + 行距」（不会多出半行），且总高不超上限
// （否则会在最小窗口里把底部按钮挤出弹窗）。
func TestCacheTabHeightFitsAnyFont(t *testing.T) {
	for fs := float32(11); fs <= 22; fs++ {
		th := &ui.Theme{FontSize: fs}
		n := cacheRowsFit(th)
		if n < 3 || n > cacheRowsPreferred {
			t.Errorf("字号 %.0f：整行数 %d 超出 3..%d", fs, n, cacheRowsPreferred)
		}
		body := cacheBodyHeight(th)
		want := float32(n)*cacheRowHeight(th) + float32(n-1)*2
		if body != want {
			t.Errorf("字号 %.0f：表体高度 %.1f 与整行算式 %.1f 不一致", fs, body, want)
		}
		if tab := cacheTabChrome + body; tab > cacheTabMaxHeight {
			t.Errorf("字号 %.0f：页签高度 %.1f 超过上限 %d", fs, tab, cacheTabMaxHeight)
		}
	}
}

// TestSettingsHasBasicInfoButton 验证设置页「关于」区提供打开基本信息弹窗的入口。
func TestSettingsHasBasicInfoButton(t *testing.T) {
	a := newTestApp()
	a.page = "settings"
	// 「关于」区在设置页最下方，默认 720 高的视口里点不到，用更高的视口渲染。
	tt := ui.NewTester(a.View, 1100, 1200)
	mustHave(t, tt, "关于", "基本信息", "查看")
	if err := tt.Click("查看"); err != nil {
		t.Fatalf("点击「查看」失败: %v", err)
	}
	tt.Frame()
	if !a.showBasicInfo {
		t.Fatal("点击「查看」应打开基本信息弹窗")
	}
	mustHave(t, tt, "数据与日志")
}

// TestLicensePage 验证生成 license 页的表单与结果区。
func TestLicensePage(t *testing.T) {
	a := newTestApp()
	a.page = "gen-license"
	mustHave(t, render(a),
		"授权信息", "授权对象", "版本", "有效期", "功能位",
		"生成凭证", "生成结果", "生成历史",
	)
}

// TestLicenseGenerate 验证生成动作会产出可自校验的凭证并写入历史。
func TestLicenseGenerate(t *testing.T) {
	a := newTestApp()
	a.page = "gen-license"
	tt := render(a)

	// 未填授权对象时应给出提示而不是生成
	if err := tt.Click("生成凭证"); err != nil {
		t.Fatalf("点击生成失败: %v", err)
	}
	tt.Frame()
	if a.lic.result != "" {
		t.Fatal("未填授权对象时不应生成")
	}

	a.lic.subject = "北京必优航空科技"
	tt.Frame()
	if err := tt.Click("生成凭证"); err != nil {
		t.Fatalf("点击生成失败: %v", err)
	}
	tt.Frame()
	if a.lic.result == "" {
		t.Fatal("填写授权对象后应生成凭证")
	}
	if len(a.lic.history) != 1 {
		t.Fatalf("应写入 1 条历史，实际 %d", len(a.lic.history))
	}
}

// TestDemoPages 验证组件演示与功能演示的页面都能渲染出内容（不 panic、不空白）。
func TestDemoPages(t *testing.T) {
	pages := []struct {
		id   string
		want []string
	}{
		{"comp-buttons", []string{"按钮", "基础样式", "开关按钮"}},
		{"comp-inputs", []string{"文本输入", "选择与数值", "并发数"}},
		{"comp-table", []string{"表格", "授权对象", "L-1001"}},
		{"comp-card", []string{"基础卡片", "带头像", "指标卡"}},
		{"comp-tabs", []string{"标签页", "分段控件", "折叠面板"}},
		{"fn-confetti", []string{"彩带庆祝", "撒一把"}},
		{"fn-chart", []string{"实时吞吐", "授权校验"}},
		{"fn-todo", []string{"待办清单", "添加"}},
		{"fn-vlist", []string{"虚拟列表", "10000 行"}},
		{"fn-timeline", []string{"动画时间线", "重播"}},
	}
	for _, p := range pages {
		a := newTestApp()
		a.page = p.id
		mustHave(t, render(a), p.want...)
	}
}

// TestNavigationFromHome 验证首页的快捷按钮能跳到对应页面。
func TestNavigationFromHome(t *testing.T) {
	a := newTestApp()
	tt := render(a)
	if err := tt.Click("发起任务"); err != nil {
		t.Fatalf("点击 发起任务 失败: %v", err)
	}
	tt.Frame()
	if a.page != "task" {
		t.Fatalf("应跳转到 task，实际 %q", a.page)
	}
	mustHave(t, tt, "任务队列")
}

// TestDialogRender 直接打开弹窗，验证其内容与关闭按钮渲染。
func TestDialogRender(t *testing.T) {
	a := newTestApp()
	a.showDlg = true
	mustHave(t, render(a), "系统通知", "config.json 文件", "我知道了")
}

// TestOfficialDemoPages 验证从 mygo 官方 examples 移植的页面都能渲染出内容。
func TestOfficialDemoPages(t *testing.T) {
	cases := []struct {
		id   string
		want []string
	}{
		// 离屏渲染没有 GPU，黑洞页会走「无硬件加速」的说明分支（见 blackhole.Scene.draw）。
		{"official-blackhole", []string{"黑洞（GPU 着色器）", "Gargantua", "没有硬件加速"}},
		{"official-piano", []string{"钢琴（合成音）", "静音", "小星星", "欢乐颂", "停止"}},
		{"official-code-editor", []string{"代码编辑器", "软换行", "server.go", "App.tsx", "行 1，列 1"}},
		{"official-text-input", []string{"自绘输入框（文本原语）", "选区", "UTF-16 长度"}},
		{"official-text-buffer", []string{"大文档编辑（索引缓冲）", "中间插一行", "撤销", "跳转到行", "为什么不用 string"}},
		{"official-flyout", []string{"浮出面板（Flyout）", "编辑便签…", "关于", "（便签为空）"}},
		{"official-loading", []string{"加载指示器", "全部指示器", "暂停动画", "尺寸", "缓动"}},
		{"official-notes", []string{"笔记", "新建", "复制 Markdown", "欢迎使用笔记"}},
		{"official-chat", []string{"Markdown 聊天", "发送", "随便问点什么", "流式单位"}},
	}
	for _, c := range cases {
		a := newTestApp()
		// 笔记页会把内容落到 notes.json：测试注入临时路径，别读开发机上的真实笔记。
		a.notePath = filepath.Join(t.TempDir(), "notes.json")
		a.page = c.id
		tt := render(a)
		// 黑洞页的「没有 GPU」说明要靠第二帧才切换过来，故统一多渲一帧。
		tt.Frame()
		mustHave(t, tt, c.want...)
	}
}

// TestNotesPersist 验证笔记改动会写到注入的 notes.json，并在重新进入后读回。
func TestNotesPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.json")

	a := newTestApp()
	a.notePath = path
	a.page = "official-notes"
	render(a) // 触发 ensureNotes：没有文件时建一条欢迎笔记

	if len(a.notes) != 1 {
		t.Fatalf("首次进入应只有 1 条欢迎笔记，实际 %d", len(a.notes))
	}
	a.saveNotes()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("保存后应写出 notes.json: %v", err)
	}

	// 新建一条并保存，再用新实例读回。
	a.addNote()
	a.saveNotes()

	b := newTestApp()
	b.notePath = path
	b.page = "official-notes"
	render(b)
	if len(b.notes) < 2 {
		t.Errorf("重新进入后应读回 2 条笔记，实际 %d", len(b.notes))
	}
}

// TestLightTaskShowsLabel 验证轻量任务执行时页头标题右侧出现任务名。
func TestLightTaskShowsLabel(t *testing.T) {
	a := newTestApp()
	a.taskRunning = true
	a.taskProgress = 0.4
	a.taskLabel = "刷新"
	mustHave(t, render(a), "刷新")
}

// TestDividersOnSameRow 是布局的核心几何断言：侧栏「品牌头下的分割线」与
// 内容区「页头下的分割线」必须落在同一像素行。
func TestDividersOnSameRow(t *testing.T) {
	tt := render(newTestApp())
	img := tt.Image()

	const (
		winW       = 1100
		sidebarCol = 200       // 侧栏内、右侧空白处
		contentCol = winW - 40 // 内容区内、右侧空白处
		bandTop    = 50
		bandBottom = 60
	)
	sbY := dividerRow(img, sidebarCol, bandTop, bandBottom)
	ctY := dividerRow(img, contentCol, bandTop, bandBottom)
	if sbY < 0 || ctY < 0 {
		t.Fatalf("未在带内找到分割线: sidebar=%d content=%d", sbY, ctY)
	}
	if sbY != ctY {
		t.Errorf("两条分割线不在同一行: 侧栏 y=%d, 内容区 y=%d", sbY, ctY)
	}
}

// dividerRow 在给定列的 [top,bottom) 区间内返回最暗的一行。
func dividerRow(img *image.RGBA, x, top, bottom int) int {
	bestY, bestL := -1, 1e9
	for y := top; y < bottom && y < img.Rect.Dy(); y++ {
		c := img.RGBAAt(x, y)
		l := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
		if l < bestL {
			bestL, bestY = l, y
		}
	}
	return bestY
}

// TestBackdropThemeAlphas 锁定「背景图下的透明度层次」：
// 浮层 < 悬停底 < 控件面 < 按下态 ≤ 1。
func TestBackdropThemeAlphas(t *testing.T) {
	a := newTestApp()
	a.bgBitmap = &ui.Bitmap{}
	a.menuOpacity = 0.50
	a.contentOpacity = 0.60

	th := &ui.Theme{
		Background:     ui.RGB(255, 255, 255),
		Surface:        ui.RGB(244, 244, 245),
		SurfaceHover:   ui.RGB(233, 233, 236),
		SurfacePressed: ui.RGB(221, 221, 225),
	}
	a.applyBackdropTheme(th)

	panel := a.contentPanelAlpha
	hover := float64(th.SurfaceHover.A) / 255
	surface := float64(th.Surface.A) / 255
	pressed := float64(th.SurfacePressed.A) / 255
	if !(panel < hover && hover < surface && surface < pressed && pressed <= 1) {
		t.Errorf("透明度层次不成立: panel=%.2f hover=%.2f surface=%.2f pressed=%.2f",
			panel, hover, surface, pressed)
	}
	if surface-panel < 0.05 {
		t.Errorf("控件面与浮层过于接近（%.2f vs %.2f）", surface, panel)
	}
}

// TestNoBackdropKeepsOpaque 验证没有背景图时所有面完全不透明。
func TestNoBackdropKeepsOpaque(t *testing.T) {
	a := newTestApp()
	a.bgBitmap = nil
	th := &ui.Theme{
		Background:     ui.RGB(255, 255, 255),
		Surface:        ui.RGB(244, 244, 245),
		SurfaceHover:   ui.RGB(233, 233, 236),
		SurfacePressed: ui.RGB(221, 221, 225),
	}
	a.applyBackdropTheme(th)
	for name, c := range map[string]ui.Color{
		"Background": th.Background, "Surface": th.Surface,
		"SurfaceHover": th.SurfaceHover, "SurfacePressed": th.SurfacePressed,
	} {
		if c.A != 255 {
			t.Errorf("无背景图时 %s 应完全不透明，实际 alpha=%d", name, c.A)
		}
	}
}
