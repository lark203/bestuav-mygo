package view

import "github.com/egoist/mygo/ui"

// ---- 首页（对应截图主体） ----

func (a *App) homePage(c *ui.Context, t *ui.Theme) {
	// 说明：页面标题已由内容区固定页头统一展示（见 content.go），此处不再重复。
	// 项目概览 区块标题行
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "项目概览").FontSize(20).Bold().Grow(1)
		ui.Text(c, a.welcome).TextColor(t.TextMuted)
		// 紫框按钮（outline）
		if ui.Button(c, "点击交互").
			Background(t.Background).
			Border(1, t.Accent).
			TextColor(t.Accent).
			Clicked() {
			a.showDlg = true
		}
	})

	ui.Text(c, "概览卡片").FontSize(15).Bold().Padding(20, 0, 10)

	// 三张概览卡
	ui.Row(c).Gap(16).Children(func() {
		overviewCard(c, t, iconPackage, "总销售额", "¥ 12,345")
		overviewCard(c, t, iconBell, "新订单", "120 笔")
		overviewCard(c, t, iconStar, "活跃用户", "5000+")
	})

	ui.Text(c, "最近产品").FontSize(15).Bold().Padding(24, 0, 10)
	a.productTable(c, t)
}

func overviewCard(c *ui.Context, t *ui.Theme, svg *ui.SVG, label, value string) {
	ui.Column(c).Gap(8).Padding(18).Width(220).Radius(10).
		Background(t.Surface).Border(1, t.Border).
		Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.18)).Children(func() {
		ui.Image(c, svg).Size(26, 26).TextColor(t.Accent)
		ui.Text(c, label).FontSize(13).TextColor(t.TextMuted)
		ui.Text(c, value).FontSize(26).Bold()
	})
}

func (a *App) productTable(c *ui.Context, t *ui.Theme) {
	cols := []ui.TableColumn{
		{Title: "名称", Sortable: true},
		{Title: "分类", Sortable: true},
		{Title: "价格", Width: 120, Align: ui.End, Sortable: true},
	}
	a.plist.Key = func(i int) any { return a.products[i].name }
	a.plist.Label = func(i int) string { return a.products[i].name }
	a.plist.Selected = &a.pSel
	a.plist.Sort = &a.pSort
	ui.Table(c, &a.plist, cols, len(a.products), func(row, col int) {
		p := a.products[row]
		switch col {
		case 0:
			ui.Text(c, p.name).SingleLine()
		case 1:
			ui.Text(c, p.category).SingleLine()
		case 2:
			ui.Text(c, p.price).SingleLine()
		}
	}).Grow(1)
}

// ---- 占位页（其余菜单项） ----

func placeholderPage(c *ui.Context, t *ui.Theme, title, id string) {
	if title == "" {
		title = id
	}
	// 页面标题由内容区固定页头展示，这里只留页面自身内容。
	ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(0, 0, 16).Children(func() {
		ui.Column(c).Width(8).Height(8).Radius(999).Background(t.Accent)
		ui.Text(c, "（演示占位页面）").TextColor(t.TextMuted)
	})
	ui.Text(c, "该页面为整体框架占位，后续可接入对应功能模块。").TextColor(t.TextMuted)
}

// ---- 弹窗（点击交互后弹出，对应 JavaFX 的 FXCustomDialog） ----

func (a *App) dialog(c *ui.Context, t *ui.Theme) {
	ui.Modal(c, &a.showDlg, func() {
		ui.Text(c, "系统通知").FontSize(18).Bold()
		ui.Text(c, "您保存的配置已成功应用到本地 config.json 文件。").
			TextColor(t.TextMuted).Padding(0, 12, 12)
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.PrimaryButton(c, "我知道了").Clicked() {
				a.showDlg = false
			}
		})
	})
}
