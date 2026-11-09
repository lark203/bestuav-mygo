package view

import (
	"github.com/egoist/mygo/ui"
)

// 本文件把「页头要展示什么」收拢到一处：
//
//   - 副标题：任务名 / 主从示例里当前选中的项；
//   - Extras：主从示例页头的说明文字（页头固定高度，组件不会把标题栏撑高）；
//   - Tabs：设置页的四个分组、布局示例页的三个 tab（有 tab 时副标题让位）；
//   - Fill：三段式布局（左右两栏）不套统一滚动，由页面自己管理栏内滚动。

// pageSubtitle 页头副标题。
func (a *App) pageSubtitle() string {
	// 轻量任务执行时优先显示任务名（与页头下方的进度条呼应）。
	if a.taskRunning && a.taskLabel != "" {
		return a.taskLabel
	}
	if a.page == "layout-split" {
		if a.demoSel > 0 && a.demoSel <= len(demoLayoutItems) {
			return demoLayoutItems[a.demoSel-1].name
		}
		return "未选择"
	}
	return ""
}

// pageExtras 页头自定义组件：主从示例展示一行说明文字。
//
// 返回 nil 表示这一页没有额外组件；容器高度由 shell 锁定为页头高度，
// 文本/下拉框/小控件都只占一行，不会改变标题栏高度。
func (a *App) pageExtras(c *ui.Context, t *ui.Theme) func() {
	if a.page != "layout-split" {
		return nil
	}
	return func() {
		ui.Text(c, "右侧随左侧选择变化").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	}
}

// pageTabs 页头 tab：返回 (标签, 当前下标, 切换回调)；nil 表示这一页不用 tab。
func (a *App) pageTabs() ([]string, int, func(int)) {
	switch a.page {
	case settingsPageID:
		return settingsGroupLabels(), a.settingsGroup(), a.setSettingsGroup
	case "layout-tabs":
		return []string{"概览", "明细", "设置"}, a.demoTab, func(i int) {
			a.demoTab = i
			a.requestFrame()
		}
	}
	return nil, 0, nil
}

// pageFill 是否需要「内容区不套统一滚动」：三段式布局自己管左右两栏的滚动。
func (a *App) pageFill() bool {
	return a.page == "layout-split"
}

// demoLayoutItems 布局示例用的静态数据（不碰数据库，纯展示）。
var demoLayoutItems = []struct {
	name string
	desc string
	meta string
}{
	{"星图", "把待办、项目与多个仓库摊在一张图上的桌面工作台。", "研发 · 3 个仓库 · 活跃"},
	{"量子科学桌面端", "内部工具链的桌面入口，负责配置与诊断。", "工具 · 1 个仓库 · 维护中"},
	{"数据同步", "跨机器同步配置与数据的辅助项目。", "基础设施 · 2 个仓库 · 暂停"},
	{"文档站", "使用说明与发布手册的静态站点。", "文档 · 0 个仓库 · 活跃"},
	{"实验沙盒", "临时验证新想法的空间。", "实验 · 1 个仓库 · 归档"},
}
