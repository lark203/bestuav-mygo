package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 产品数据（mock，对应截图“最近产品”表）。
type product struct {
	name     string
	category string
	price    string
}

// app 持有窗口状态；view 在主线程按帧从它构建界面。
type app struct {
	page string // 当前选中页 id

	// 侧栏分组开合状态
	openTools, openProfile, openComponents, openFunctions bool

	// 首页
	welcome  string
	products []product
	showDlg  bool

	// 最近产品表状态
	plist ui.ListState
	pSel  int
	pSort ui.SortOrder
}

func newApp() *app {
	return &app{
		page:            "home",
		openTools:       true,
		openProfile:     true,
		openComponents:  true,
		openFunctions:   true,
		welcome:         "欢迎来到AtlantaFX 世界！",
		products: []product{
			{"笔记本电脑", "电子产品", "¥ 8999"},
			{"机械键盘", "外设", "¥ 699"},
			{"无线鼠标", "外设", "¥ 299"},
			{"显示器", "电子产品", "¥ 2499"},
		},
	}
}

// pageTitles 各页面的标题（用于右侧页头与占位页）。
var pageTitles = map[string]string{
	"home":             "首页",
	"gen-license":      "生成license",
	"gen-pwd":          "生成用户密码",
	"profile-basic":    "基本信息",
	"profile-security": "安全设置",
	"debug":            "调试中心",
	"keypair":          "公钥私钥",
	"async":            "异步任务",
	"comp-buttons":     "按钮组件",
	"comp-inputs":      "输入框组件",
	"comp-table":       "表格组件",
	"comp-card":        "卡片组件",
	"comp-tabs":        "标签页组件",
	"fn-chart":         "图表功能",
	"fn-form":          "表单功能",
	"fn-dialog":        "弹窗功能",
	"task":             "Task演示",
	"virtuallist":      "虚拟列表演示",
	"grand":            "GRAND_DEMO",
	"async-dev":        "异步开发",
	"settings":         "设置",
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	// 不覆盖主题色：强调色、背景等均使用 mygo 的系统默认（跟随系统明暗）。

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Fill().Children(func() {
			a.sidebar(c, t)
			ui.Scroll(c).Fill().Padding(24, 28).Children(func() {
				switch a.page {
				case "home":
					a.homePage(c, t)
				default:
					placeholderPage(c, t, pageTitles[a.page], a.page)
				}
			})
		})
	})
	a.dialog(c, t)
}

func main() {
	a := newApp()
	// 跟随系统明暗（不再强制深色，也不自定义强调色）。
	mygo.Theme.SetSource(mygo.ThemeSystem)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:     "量子科学 (mygo)",
			Width:     1100,
			Height:    720,
			MinWidth:  820,
			MinHeight: 520,
			StateKey:  "bestuav-mygo",
			Content:   ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
