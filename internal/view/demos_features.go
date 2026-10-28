package view

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/ui"
)

// ---- 功能演示：五个页面 ----
//
// 迁移自 mygo 官方示例的思路（examples/confetti、examples/chart-streaming、
// examples/todo、examples/gallery 的万行列表、examples/timeline），按我们的
// 主题与页面结构重写：
//   - 彩带：核心算法在 internal/view/fx（移植自 examples/confetti）；
//   - 滚动图表：用 ui.Box.Draw + ui.Path 自绘多序列曲线（对应 examples/chart-streaming
//     的「滚动波形」，这里用确定性数据 + 动画帧推进，避免引入 15k 采样缓冲）；
//   - 待办清单：对应 examples/todo 的增删改；
//   - 虚拟列表：ui.Table 承载 10000 行（对应 examples/gallery 的万行列表）；
//   - 动画时间线：多元素错峰入场（对应 examples/timeline 的多 tween 同轨播放）。

// ---- 彩带庆祝 ----

// confettiDemoPage 手动触发彩带，并说明它同时被用在任务完成时。
func (a *App) confettiDemoPage(c *ui.Context, t *ui.Theme) {
	card(c, t, func() {
		cardHeader(c, t, "彩带庆祝", "移植自 mygo examples/confetti（canvas-confetti 的三层运动模型）", nil)
		ui.Text(c, "任务完成时会自动撒彩带；这里可以手动触发，观察不同数量与形状。").
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.PrimaryButton(c, "撒一把（3 团）").Clicked() {
				a.celebrate(3)
			}
			if ui.Button(c, "大量（6 团）").Clicked() {
				a.celebrate(6)
			}
			ui.Text(c, "每一片都是方块/圆/星/心随机组合，受重力下落并翻转淡出。").
				FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1)
		})
	})

	ui.Grid(c).Columns(2).Gap(16).Children(func() {
		card(c, t, func() {
			cardHeader(c, t, "在业务里怎么用", "两行代码", nil)
			ui.Text(c, "a.celebrate(3)          // 任务完成时撒 3 团").Selectable().
				FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			ui.Text(c, "a.startLightTask(\"导出\") // 轻量任务跑完会自动庆祝").Selectable().
				FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
		card(c, t, func() {
			cardHeader(c, t, "实现要点", "为什么不用 GIF/PNG", nil)
			ui.Text(c, "彩带是 GPU 自绘的矢量图形：任意分辨率清晰、颜色跟随主题、无需素材文件，"+
				"几百片同时飞也不掉帧（位姿只在重绘帧推进，不重建视图）。").
				FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
	})
}

// ---- 滚动图表 ----

// chartDemoPage 自绘滚动折线图：三条序列，随时间向右推进。
func (a *App) chartDemoPage(c *ui.Context, t *ui.Theme) {
	if len(a.chartSeries[0]) == 0 {
		a.seedChart()
	}
	card(c, t, func() {
		cardHeader(c, t, "实时吞吐（示意）", "每秒追加一个采样点，滚动显示最近 60 点", func() {
			if ui.Button(c, "暂停/继续").Clicked() {
				a.chartPaused = !a.chartPaused
			}
		})
		ui.Box(c).Height(200).Radius(8).Background(withAlpha(t.SurfaceHover, 0.6)).
			Draw(func(p *ui.Painter, r ui.Rect) {
				drawChart(p, r, t, a.chartSeries, []string{"授权校验", "报表导出", "任务调度"})
			})
		ui.Row(c).Gap(16).Children(func() {
			names := []string{"授权校验", "报表导出", "任务调度"}
			cols := []ui.Color{t.Accent, t.Success, t.Warning}
			for i, n := range names {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Column(c).Width(10).Height(3).Radius(2).Background(cols[i])
					ui.Text(c, n).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				})
			}
		})
	})

	if !a.chartPaused {
		// 每次重绘推进一格：用 AnimationFrame 请求下一帧，形成滚动。
		c.AnimationFrame()
		a.stepChart()
	}
}

// seedChart 初始化三条序列。
func (a *App) seedChart() {
	for s := 0; s < 3; s++ {
		a.chartSeries[s] = a.chartSeries[s][:0]
		for i := 0; i < 60; i++ {
			a.chartSeries[s] = append(a.chartSeries[s], chartValue(s, i))
		}
	}
}

// stepChart 追加新点并丢弃最旧的点（保持 60 点窗口）。
func (a *App) stepChart() {
	for s := 0; s < 3; s++ {
		if len(a.chartSeries[s]) >= 60 {
			a.chartSeries[s] = a.chartSeries[s][1:]
		}
		a.chartSeries[s] = append(a.chartSeries[s], chartValue(s, int(a.chartTick)))
	}
	a.chartTick++
}

// chartValue 生成确定性的示意数据（正弦 + 噪声 + 序列偏移）。
func chartValue(series, i int) float32 {
	x := float64(i) / 8
	v := 0.5 + 0.28*math.Sin(x+float64(series)*1.7) + 0.12*math.Sin(x*2.7+float64(series))
	v += (rand.Float64() - 0.5) * 0.04
	return float32(v)
}

// drawChart 画网格 + 三条折线。
func drawChart(p *ui.Painter, r ui.Rect, t *ui.Theme, data [3][]float32, names []string) {
	// 背景网格：4 条水平线
	for i := 0; i <= 4; i++ {
		y := r.Y + r.H*float32(i)/4
		p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: 1}, withAlpha(t.Border, 0.7), 0)
	}
	cols := []ui.Color{t.Accent, t.Success, t.Warning}
	for s := 0; s < 3; s++ {
		pts := data[s]
		if len(pts) < 2 {
			continue
		}
		path := &ui.Path{}
		for i, v := range pts {
			x := r.X + r.W*float32(i)/float32(len(pts)-1)
			y := r.Y + r.H*(1-v)
			if i == 0 {
				path.MoveTo(x, y)
			} else {
				path.LineTo(x, y)
			}
		}
		p.StrokePath(path, 1.8, cols[s])
	}
}

// ---- 待办清单 ----

// todoItem 一条待办。
type todoItem struct {
	text string
	done bool
}

// todoDemoPage 待办清单：新增、勾选、删除、统计。
func (a *App) todoDemoPage(c *ui.Context, t *ui.Theme) {
	if len(a.todos) == 0 && !a.todoSeeded {
		a.todos = []todoItem{
			{"接入授权服务健康检查", true},
			{"补充 license 过期提醒", false},
			{"把任务进度接到真实后端", false},
			{"整理 README 与截图", false},
		}
		a.todoSeeded = true
	}

	card(c, t, func() {
		cardHeader(c, t, "待办清单", a.todoSummary(), func() {
			if ui.Button(c, "清除已完成").Clicked() {
				kept := a.todos[:0]
				for _, it := range a.todos {
					if !it.done {
						kept = append(kept, it)
					}
				}
				a.todos = kept
			}
		})

		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, &a.todoInput).Placeholder("输入待办，回车或点右侧添加").Grow(1)
			if ui.PrimaryButton(c, "添加").Clicked() {
				a.addTodo()
			}
		})

		ui.Divider(c)
		for i := range a.todos {
			idx := i
			row := ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(6, 6).Radius(t.Radius)
			if row.Hovered() {
				row.Background(t.SurfaceHover)
			}
			row.Children(func() {
				ui.Checkbox(c, &a.todos[idx].done, a.todos[idx].text)
				ui.Row(c).Grow(1)
				if ui.Button(c, "删除").Clicked() {
					a.todos = append(a.todos[:idx], a.todos[idx+1:]...)
				}
			})
		}
		if len(a.todos) == 0 {
			emptyHint(c, t, "全部完成，干净了。")
		}
	})
}

// todoSummary 待办统计文案。
func (a *App) todoSummary() string {
	if len(a.todos) == 0 {
		return "0 条"
	}
	done := 0
	for _, it := range a.todos {
		if it.done {
			done++
		}
	}
	return fmt.Sprintf("%d / %d 已完成", done, len(a.todos))
}

// addTodo 添加一条待办（忽略空白）。回车提交也走这里。
func (a *App) addTodo() {
	text := a.todoInput
	if text == "" {
		a.toast("请输入内容")
		return
	}
	a.todos = append(a.todos, todoItem{text: text})
	a.todoInput = ""
}

// ---- 虚拟列表 ----

// vlistDemoPage 1 万行列表：ui.Table 只渲染可见行，滚动依然顺滑。
func (a *App) vlistDemoPage(c *ui.Context, t *ui.Theme) {
	const rows = 10000
	card(c, t, func() {
		cardHeader(c, t, "虚拟列表", "10000 行，只渲染可见部分（ui.Table + ui.ListState）", func() {
			ui.Text(c, "第 "+itoa(a.vlistSel+1)+" 行").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
		cols := []ui.TableColumn{
			{Title: "序号", Width: 90, Align: ui.End},
			{Title: "事件", Sortable: false},
			{Title: "结果", Width: 110},
		}
		a.vlistState.Key = func(i int) any { return i }
		a.vlistState.Label = func(i int) string { return "第 " + itoa(i+1) + " 行" }
		a.vlistState.Selected = &a.vlistSel
		ui.Table(c, &a.vlistState, cols, rows, func(row, col int) {
			switch col {
			case 0:
				ui.Text(c, itoa(row+1)).SingleLine()
			case 1:
				ui.Text(c, "授权校验请求 #"+itoa(100000+row)).SingleLine()
			case 2:
				if row%17 == 0 {
					pill(c, t, "重试", t.Warning)
				} else {
					pill(c, t, "成功", t.Success)
				}
			}
		}).FillWidth().Height(360)
	})
}

// ---- 动画时间线 ----

// timelineDemoPage 多元素错峰入场：一个按钮重播。
func (a *App) timelineDemoPage(c *ui.Context, t *ui.Theme) {
	card(c, t, func() {
		cardHeader(c, t, "动画时间线", "错峰入场：每张卡延迟 80ms，位姿动画只在重绘帧推进", func() {
			if ui.Button(c, "重播").Clicked() {
				a.timelinePlay = !a.timelinePlay
			}
		})
		ui.Column(c).Gap(12).Children(func() {
			steps := []struct {
				title, desc string
				icon        *ui.SVG
			}{
				{"提交请求", "客户端发起授权校验", iconBolt},
				{"服务校验", "校验签名与有效期", iconShield},
				{"签发结果", "写入审计日志并返回", iconFileKey},
				{"完成", "本地缓存 5 分钟", iconChecklist},
			}
			for i, s := range steps {
				idx := i
				from := ui.Pose{X: -18, Fade: 0}
				if a.timelinePlay {
					from = ui.Pose{}
				}
				row := ui.Row(c).Gap(12).AlignItems(ui.Center).Padding(12).Radius(8).
					Background(withAlpha(t.SurfaceHover, 0.7)).Border(1, t.Border)
				row.AnimatePose(from, ui.Animation{
					Duration: 260 * time.Millisecond,
					Delay:    time.Duration(idx) * 80 * time.Millisecond,
					Ease:     ui.EaseOut,
				})
				row.Children(func() {
					ui.Icon(c, s.icon).FontSize(16).TextColor(t.Accent)
					ui.Column(c).Gap(2).Grow(1).Children(func() {
						ui.Text(c, s.title).Bold().SingleLine()
						ui.Text(c, s.desc).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
					})
					pill(c, t, "步骤 "+itoa(idx+1), t.Accent)
				})
			}
		})
	})
}
