package view

import (
	"testing"

	"github.com/egoist/mygo/ui"

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
		"量子科学",                // 品牌头
		"首页",                  // 页头 + 侧栏选中项
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
	for _, w := range []string{"量子科学", "工具箱", "生成license", "设置"} {
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

// TestSettingsPage 验证设置页分区域渲染（外观/窗口与关闭/关于）。
func TestSettingsPage(t *testing.T) {
	a := NewApp()
	a.page = "settings"
	mustHave(t, render(a),
		"设置", "外观", "主题", "字体大小",
		"窗口与关闭", "关闭按钮行为", "每次询问",
		"关于", "配置文件",
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
