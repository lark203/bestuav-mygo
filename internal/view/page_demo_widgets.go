package view

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/icons"
	"bestuav-mygo/internal/ui/kit"
)

// ---- 组件演示：五个页面，每个聚焦一类控件 ----
//
// 这些页面替代了原来的占位页。写法与真实页面一致（卡片 + 说明 + 可交互控件），
// 既是控件目录，也是「怎么用 mygo 控件」的可运行示例。

// demoRow 一行「控件 + 说明」。行距由外层的 kit.Card 统一给出，这里不再加内边距。
func demoRow(c *ui.Context, t *ui.Theme, label, desc string, fn func()) {
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, label).FontSize(t.FontSize - 1).Bold()
		if desc != "" {
			ui.Text(c, desc).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		}
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(fn)
	})
}

// buttonsDemoPage 按钮类控件。
func (a *App) buttonsDemoPage(c *ui.Context, t *ui.Theme) {
	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "按钮", "主按钮只用于页面主操作，次要动作用普通按钮", nil)
		demoRow(c, t, "基础样式", "ui.PrimaryButton / ui.Button", func() {
			ui.PrimaryButton(c, "主要操作")
			ui.Button(c, "次要操作")
			ui.Button(c, "禁用").Disabled(true)
		})
		demoRow(c, t, "带图标", "用 Children 放图标，label 留空并加 Label", func() {
			ui.PrimaryButton(c, "").Label("刷新").Children(func() {
				ui.Icon(c, icons.Refresh).FontSize(14)
			})
			ui.Button(c, "").Label("导出").Children(func() {
				ui.Icon(c, icons.Download).FontSize(14)
			})
		})
		demoRow(c, t, "开关按钮", "ui.Toggle：按下态保持", func() {
			ui.Toggle(c, &a.demoToggle, "加粗")
			ui.Toggle(c, &a.demoToggle2, "自动刷新")
		})
		demoRow(c, t, "危险操作", "用 TextColor(t.Danger) 表达破坏性", func() {
			ui.Button(c, "删除授权").TextColor(t.Danger)
		})
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "评分与徽标", "ui.Rating / kit.Pill", nil)
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Rating(c, &a.demoRating, 5)
			kit.Pill(c, t, "推荐", t.Success)
			kit.Pill(c, t, "试用中", t.Warning)
			kit.Pill(c, t, "已停用", t.TextMuted)
		})
	})
}

// inputsDemoPage 输入类控件。
func (a *App) inputsDemoPage(c *ui.Context, t *ui.Theme) {
	kit.RowGrid(c, 2, func() {
		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "文本输入", "ui.TextInput / ui.TextArea", nil)
			ui.Field(c, "名称", func() {
				ui.TextInput(c, &a.demoText).Placeholder("请输入名称")
			})
			ui.Field(c, "备注", func() {
				ui.TextArea(c, &a.demoNote).Placeholder("多行文本…")
			})
			ui.Field(c, "日期", func() {
				ui.DateInput(c, &a.demoDateT)
			})
		})

		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "选择与数值", "下拉、开关、滑块、步进器", nil)
			ui.Field(c, "版本", func() {
				ui.Select(c, &a.demoSelect, []string{"标准版", "专业版", "企业版"})
			})
			ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
				ui.Checkbox(c, &a.demoCheck, "启用缓存")
				ui.Switch(c, &a.demoSwitch)
			})
			ui.Field(c, "阈值", func() {
				ui.Slider(c, &a.demoSlider, 0, 100).Width(200)
				ui.Text(c, fmt.Sprintf("%.0f", a.demoSlider)).
					TextColor(t.TextMuted).Padding(8, 0, 0, 0)
			})
			ui.Field(c, "并发数", func() {
				ui.Stepper(c, &a.demoStep, 1, 32, 1)
			})
		})
	})
}

// tableDemoPage 表格控件：可排序 + 可选中。
func (a *App) tableDemoPage(c *ui.Context, t *ui.Theme) {
	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "表格", "点击表头排序、点击行选中（ui.Table + ui.ListState）", func() {
			if ui.Button(c, "清空选择").Clicked() {
				a.demoSel = -1
			}
		})
		cols := []ui.TableColumn{
			{Title: "编号", Width: 90, Sortable: true},
			{Title: "授权对象", Sortable: true},
			{Title: "版本", Width: 110, Sortable: true},
			{Title: "到期", Width: 120, Align: ui.End},
		}
		a.demoTable.Key = func(i int) any { return demoLicenses[i].id }
		a.demoTable.Label = func(i int) string { return demoLicenses[i].subject }
		a.demoTable.Selected = &a.demoSel
		ui.Table(c, &a.demoTable, cols, len(demoLicenses), func(row, col int) {
			r := demoLicenses[row]
			switch col {
			case 0:
				ui.Text(c, r.id).SingleLine()
			case 1:
				ui.Text(c, r.subject).SingleLine()
			case 2:
				ui.Text(c, r.edition).SingleLine()
			case 3:
				ui.Text(c, r.expiry).SingleLine()
			}
		}).FillWidth()
		if a.demoSel >= 0 && a.demoSel < len(demoLicenses) {
			sel := demoLicenses[a.demoSel]
			ui.Text(c, "已选中："+sel.id+" · "+sel.subject).
				FontSize(t.FontSize - 2).TextColor(t.Accent)
		}
	})
}

// demoLicense 表格演示数据。
type demoLicense struct {
	id, subject, edition, expiry string
}

var demoLicenses = []demoLicense{
	{"L-1001", "北京必优航空科技", "企业版", "2027-06-30"},
	{"L-1002", "上海云图信息", "专业版", "2026-12-31"},
	{"L-1003", "深圳前海智造", "标准版", "2026-11-15"},
	{"L-1004", "杭州灵犀数据", "专业版", "2028-01-31"},
	{"L-1005", "成都天工测控", "企业版", "永久"},
	{"L-1006", "武汉长江光电", "标准版", "2026-12-01"},
}

// cardDemoPage 卡片类布局：三张卡等宽栅格。
func (a *App) cardDemoPage(c *ui.Context, t *ui.Theme) {
	kit.RowGrid(c, 3, func() {
		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "基础卡片", "圆角 10 + 1px 边框 + 表面色", nil)
			ui.Text(c, "所有页面的区块都用这套：背景随主题走，设置背景图后自动半透明，不遮挡壁纸。").
				FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "带头像", "ui.Avatar", nil)
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Avatar(c, "王工", nil)
				ui.Column(c).Gap(1).Children(func() {
					ui.Text(c, "王工").Bold()
					ui.Text(c, "系统管理员").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				})
			})
		})
		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "指标卡", "首页 KPI 用的就是这个", nil)
			ui.Text(c, "1,286").FontSize(24).Bold()
			ui.Text(c, "有效授权").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			kit.Pill(c, t, "↑ 24 份", t.Success)
		})
	})
}

// tabsDemoPage 标签页与折叠面板。
func (a *App) tabsDemoPage(c *ui.Context, t *ui.Theme) {
	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "标签页", "ui.Tabs：同一区域切换内容", nil)
		ui.Tabs(c, &a.demoTab, "概览", "明细", "日志")
		switch a.demoTab {
		case 1:
			ui.Text(c, "明细页：放表格或图表。").TextColor(t.TextMuted)
		case 2:
			ui.Text(c, "日志页："+time.Now().Format("15:04:05")+" 任务执行成功。").
				TextColor(t.TextMuted)
		default:
			ui.Text(c, "概览页：放摘要信息。选中状态会保持在 App.demoTab 里，切页不丢。").
				TextColor(t.TextMuted)
		}
	})

	kit.RowGrid(c, 2, func() {
		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "分段控件", "ui.Segmented：三选一", nil)
			ui.Segmented(c, &a.demoSeg, "全部", "运行中", "已归档")
		})
		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "折叠面板", "ui.Collapsible / Accordion", nil)
			ui.Collapsible(c, "高级设置", &a.demoCollapse, func() {
				switchFeatures(c, t, &a.demoCheck, &a.demoSwitch)
			})
		})
	})
}

// switchFeatures 折叠面板里的内容。
func switchFeatures(c *ui.Context, t *ui.Theme, check *bool, sw *bool) {
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, "折叠区里的内容不占位，展开后才参与布局。").
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		ui.Checkbox(c, check, "记录详细日志")
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Switch(c, sw)
			ui.Text(c, "开启实验特性").FontSize(t.FontSize - 2)
		})
	})
}
