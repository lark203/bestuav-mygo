package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func render(a *app) *ui.Tester { return ui.NewTester(a.view, 1100, 720) }

func mustHave(t *testing.T, tt *ui.Tester, want ...string) {
	t.Helper()
	for _, w := range want {
		if !tt.HasText(w) {
			t.Errorf("缺少文本 %q，当前文本: %v", w, tt.Texts())
		}
	}
}

// TestHomePage 验证默认进入首页，标题栏/侧栏/概览卡/产品表全部渲染到位（无窗口）。
func TestHomePage(t *testing.T) {
	a := newApp()
	mustHave(t, render(a),
		"量子科学",      // 自定义标题栏
		"首页",          // 页头 + 侧栏选中项
		"项目概览",      // 区块标题
		"概览卡片",      // 卡片区标题
		"总销售额", "新订单", "活跃用户", // 三张概览卡
		"¥ 12,345", "120 笔", "5000+",
		"最近产品",    // 表格区标题
		"笔记本电脑", "¥ 8999", "机械键盘", "无线鼠标", "显示器", // 产品表
		// 侧栏菜单项
		"生成license", "调试中心", "公钥私钥", "异步任务",
		"组件演示", "功能演示", "GRAND_DEMO", "设置",
	)
}

// TestSidebarFolders 验证文件夹（带箭头可折叠）作为分组呈现。
func TestSidebarFolders(t *testing.T) {
	a := newApp()
	mustHave(t, render(a), "图标", "个人资料", "组件演示", "功能演示")
}

// TestPlaceholderPage 验证非首页菜单项渲染占位页框架。
func TestPlaceholderPage(t *testing.T) {
	a := newApp()
	a.page = "settings"
	mustHave(t, render(a), "设置", "（演示占位页面）")
}

// TestClickInteraction 验证点击“点击交互”紫框按钮弹出系统通知弹窗。
func TestClickInteraction(t *testing.T) {
	a := newApp()
	tt := render(a)
	if err := tt.Click("点击交互"); err != nil {
		t.Fatalf("点击 点击交互 失败: %v", err)
	}
	tt.Frame()
	mustHave(t, tt, "系统通知", "config.json", "我知道了")
}

// TestDialogRender 直接打开弹窗，验证其内容与关闭按钮渲染。
func TestDialogRender(t *testing.T) {
	a := newApp()
	a.showDlg = true
	mustHave(t, render(a), "系统通知", "config.json 文件", "我知道了")
}
