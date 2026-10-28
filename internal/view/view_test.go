package view

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/appmeta"
	"bestuav-mygo/internal/settings"
)

func render(a *App) *ui.Tester { return ui.NewTester(a.View, 1100, 720) }

func mustHave(t *testing.T, tt *ui.Tester, want ...string) {
	t.Helper()
	for _, w := range want {
		if !tt.HasText(w) {
			t.Errorf("缺少文本 %q，当前文本: %v", w, tt.Texts())
		}
	}
}

// TestHomePage 验证默认进入首页，品牌头/侧栏/概览卡/产品表全部渲染到位（无窗口）。
func TestHomePage(t *testing.T) {
	a := NewApp()
	mustHave(t, render(a),
		appmeta.Title,         // 品牌头（取自 appmeta，改名只需改一处）
		"首页",                  // 页头标题 + 侧栏选中项
		"项目概览",                // 区块标题
		"概览卡片",                // 卡片区标题
		"总销售额", "新订单", "活跃用户", // 三张概览卡
		"¥ 12,345", "120 笔", "5000+",
		"最近产品",                                   // 表格区标题
		"笔记本电脑", "¥ 8999", "机械键盘", "无线鼠标", "显示器", // 产品表
		// 侧栏菜单项
		"生成license", "调试中心", "公钥私钥", "异步任务",
		"组件演示", "功能演示", "GRAND_DEMO", "设置",
	)
}

// TestSidebarFolders 验证分组（带箭头可折叠）作为区域呈现。
func TestSidebarFolders(t *testing.T) {
	a := NewApp()
	mustHave(t, render(a), "工具箱", "个人中心", "组件演示", "功能演示")
}

// TestSidebarCollapsed 验证收缩态：仅图标、隐藏品牌文字/分组标题/菜单文字标签。
func TestSidebarCollapsed(t *testing.T) {
	a := NewApp()
	a.sidebarCollapsed = true
	tt := render(a)
	// 注意：页头标题仍会显示（内容区不受侧栏收缩影响），故只检查侧栏专属文本。
	for _, w := range []string{appmeta.Title, "工具箱", "个人中心", "生成license", "调试中心"} {
		if tt.HasText(w) {
			t.Errorf("收缩态不应显示文本 %q，当前文本: %v", w, tt.Texts())
		}
	}
}

// TestSidebarCollapsedInitFromSettings 验证收缩状态由已保存偏好初始化（持久化）。
func TestSidebarCollapsedInitFromSettings(t *testing.T) {
	s := settings.Default()
	s.SidebarCollapsed = true
	a := &App{settings: s, sidebarCollapsed: s.SidebarCollapsed, page: "home"}
	if !a.sidebarCollapsed {
		t.Fatal("应从已保存偏好初始化为收缩态")
	}
}

// TestSettingsPage 验证设置页分区域渲染（外观/背景/窗口与关闭/启动/关于）。
func TestSettingsPage(t *testing.T) {
	a := NewApp()
	a.page = "settings"
	mustHave(t, render(a),
		"设置",
		// 外观
		"外观", "主题", "字体大小",
		// 背景（背景图 + 两个独立不透明度 + 恢复默认）
		"背景", "背景图片", "选择图片", "恢复默认", "菜单区", "内容区",
		// 窗口与关闭
		"窗口与关闭", "关闭按钮行为", "每次询问",
		// 启动（多实例开关）
		"启动", "允许多开",
		// 关于
		"关于", "配置文件", appmeta.Identifier,
	)
}

// TestCloseChoiceDialog 验证首次关闭询问弹窗内容。
func TestCloseChoiceDialog(t *testing.T) {
	a := NewApp()
	a.showCloseChoice = true
	mustHave(t, render(a),
		"您希望点击关闭按钮时执行什么操作？",
		"最小化到托盘", "直接退出", "记住我的选择",
	)
}

// TestClickInteraction 验证点击“点击交互”紫框按钮弹出系统通知弹窗。
func TestClickInteraction(t *testing.T) {
	a := NewApp()
	tt := render(a)
	if err := tt.Click("点击交互"); err != nil {
		t.Fatalf("点击 点击交互 失败: %v", err)
	}
	tt.Frame()
	mustHave(t, tt, "系统通知", "config.json", "我知道了")
}

// TestDialogRender 直接打开弹窗，验证其内容与关闭按钮渲染。
func TestDialogRender(t *testing.T) {
	a := NewApp()
	a.showDlg = true
	mustHave(t, render(a), "系统通知", "config.json 文件", "我知道了")
}

// TestLightTaskShowsLabel 验证轻量任务执行时，页头标题右侧出现任务名
// （分割线槽位此时改画进度条，见 contentDivider）。
func TestLightTaskShowsLabel(t *testing.T) {
	a := NewApp()
	a.taskRunning = true
	a.taskProgress = 0.4
	a.taskLabel = "刷新"
	mustHave(t, render(a), "刷新")
}

// TestDividersOnSameRow 是本次布局重构的核心几何断言：侧栏「品牌头下的分割线」
// 与内容区「页头下的分割线」必须落在同一像素行（用户明确要求「浅线位置和品牌名
// 的浅线持平」）。
//
// 用离屏渲染的真实像素判断，而不是只比对代码里的常量：先各取一列（避开文字与
// 菜单高亮），在页头下方那条窄带里找出唯一的浅线行，再断言两列结果相同。
func TestDividersOnSameRow(t *testing.T) {
	tt := render(NewApp())
	img := tt.Image()

	const (
		winW       = 1100
		sidebarCol = 200       // 侧栏内、右侧空白处（避开菜单文字与高亮药丸左缘）
		contentCol = winW - 40 // 内容区内、右侧空白处（避开标题与工具条）
		bandTop    = 50        // 只在页头下沿附近的窄带里找线
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

// dividerRow 在给定列的 [top,bottom) 区间内返回“最暗的一行”——页头下沿只有
// 那一条 1px 浅线，故它必然比上下相邻的背景/表面色更深。
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
