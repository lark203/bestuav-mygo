package view

import (
	"runtime"
	"runtime/debug"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/appmeta"
	"bestuav-mygo/internal/settings"
)

// ---- 基本信息页 ----
//
// 展示四类信息：应用标识、运行环境、页面缓存状态、配置文件位置。
// 「页面缓存」那块不是装饰：它实时反映 pagelife.go 的缓存/销毁结果，
// 可以直接看到「哪些页面还活着、被进入过几次、闲置多久、滚动到哪」。
// 这也是验证页面生命周期最直观的方式——比看日志快。

// envSnapshot 一次运行环境采集的结果。
type envSnapshot struct {
	goVersion  string
	platform   string
	arch       string
	cpus       int
	goroutines int
	heapMB     float64
	sysMB      float64
	buildVer   string
	buildRev   string
	collected  time.Time
}

// collectEnv 采集运行环境信息。force=false 时 5 秒内复用上次结果，避免每帧读
// MemStats（它会短暂停顿 GC）。
func (a *App) collectEnv(force bool) {
	if !force && !a.env.collected.IsZero() && time.Since(a.env.collected) < 5*time.Second {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	snap := envSnapshot{
		goVersion:  runtime.Version(),
		platform:   runtime.GOOS,
		arch:       runtime.GOARCH,
		cpus:       runtime.NumCPU(),
		goroutines: runtime.NumGoroutine(),
		heapMB:     float64(m.HeapAlloc) / (1 << 20),
		sysMB:      float64(m.Sys) / (1 << 20),
		collected:  time.Now(),
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		snap.buildVer = bi.Main.Version
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" {
				snap.buildRev = s.Value
			}
		}
	}
	a.env = snap
}

// basicInfoPage 基本信息页。
func (a *App) basicInfoPage(c *ui.Context, t *ui.Theme) {
	a.collectEnv(false)

	ui.Grid(c).Columns(2).Gap(16).Children(func() {
		// 应用信息
		ui.Column(c).Children(func() {
			card(c, t, func() {
				cardHeader(c, t, "应用信息", "产品标识统一取自 internal/appmeta", nil)
				ui.Column(c).Gap(10).Children(func() {
					kv(c, t, "产品名称", appmeta.Title)
					kv(c, t, "内部标识", appmeta.Name)
					kv(c, t, "应用标识", appmeta.Identifier)
					kv(c, t, "构建版本", a.buildVersionText())
				})
			})
		})

		// 运行环境
		ui.Column(c).Children(func() {
			card(c, t, func() {
				cardHeader(c, t, "运行环境", "数据来自 Go 运行时，5 秒缓存", func() {
					if ui.Button(c, "刷新").Clicked() {
						a.collectEnv(true)
					}
				})
				ui.Column(c).Gap(10).Children(func() {
					kv(c, t, "操作系统", a.env.platform+" / "+a.env.arch)
					kv(c, t, "Go 版本", a.env.goVersion)
					kv(c, t, "逻辑核心", itoa(a.env.cpus)+" 核")
					kv(c, t, "协程数", itoa(a.env.goroutines))
					kv(c, t, "堆内存", formatMB(a.env.heapMB))
					kv(c, t, "系统内存", formatMB(a.env.sysMB))
				})
			})
		})
	})

	// 页面缓存与生命周期
	ui.Row(c).Gap(16).Padding(16, 0, 0).Children(func() {
		card(c, t, func() {
			cardHeader(c, t, "页面缓存与生命周期",
				"缓存页面切回来保留状态；闲置超时或「不缓存」页面离开即销毁。", func() {
					if ui.Button(c, "销毁非常驻缓存").Clicked() {
						a.destroyCachedPages()
					}
				})
			if rows := a.pageCacheRows(); len(rows) == 0 {
				emptyHint(c, t, "当前没有缓存的页面")
			} else {
				a.pageCacheTable(c, t, rows)
			}
			ui.Text(c, "提示：滚动内容区后切走再切回，滚动位置仍在；点「销毁非常驻缓存」后归零。").
				FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
	})

	// 配置文件
	ui.Row(c).Gap(16).Padding(16, 0, 0).Children(func() {
		card(c, t, func() {
			cardHeader(c, t, "配置文件", "用户偏好以 JSON 原子写入", func() {
				if ui.Button(c, "复制路径").Clicked() {
					if p, err := settings.Path(); err == nil {
						a.copyText(p, "配置路径已复制")
					}
				}
			})
			p, err := settings.Path()
			if err != nil {
				ui.Text(c, "无法定位配置目录："+err.Error()).TextColor(t.Danger)
				return
			}
			ui.Text(c, p).Selectable().FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
	})
}

// buildVersionText 拼装版本信息：模块版本 + 提交号（本地 go run 时是 dev）。
func (a *App) buildVersionText() string {
	v := a.env.buildVer
	if v == "" || v == "(devel)" {
		v = "dev"
	}
	if a.env.buildRev != "" {
		rev := a.env.buildRev
		if len(rev) > 7 {
			rev = rev[:7]
		}
		return v + " · " + rev
	}
	return v
}

// pageCacheTable 页面缓存明细（页面 / 策略 / 进入次数 / 闲置 / 滚动位置）。
func (a *App) pageCacheTable(c *ui.Context, t *ui.Theme, rows []pageCacheRow) {
	ui.Column(c).Gap(2).Children(func() {
		// 表头
		ui.Row(c).Gap(12).Padding(6, 8).Children(func() {
			ui.Text(c, "页面").Width(120).Shrink(0).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			ui.Text(c, "策略").Width(64).Shrink(0).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			ui.Text(c, "进入").Width(56).Shrink(0).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			ui.Text(c, "闲置").Width(72).Shrink(0).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			ui.Text(c, "滚动位置").Grow(1).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})
		ui.Divider(c)
		for _, r := range rows {
			ui.Row(c).Gap(12).Padding(6, 8).Radius(t.Radius).Children(func() {
				ui.Text(c, r.label).Width(120).Shrink(0).SingleLine()
				ui.Text(c, r.policy).Width(64).Shrink(0).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				ui.Text(c, itoa(r.enters)+" 次").Width(56).Shrink(0).FontSize(t.FontSize - 2)
				ui.Text(c, r.idle).Width(72).Shrink(0).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				ui.Text(c, formatPX(r.scrollY)).Grow(1).FontSize(t.FontSize - 2)
			})
		}
	})
}

// destroyCachedPages 销毁所有「可缓存」页面（常驻页面保留）。
func (a *App) destroyCachedPages() {
	n := 0
	for id := range a.pages {
		if it, ok := pageByID(id); ok && it.policy.keep == keepCached {
			a.destroyPage(id)
			n++
		}
	}
	if n == 0 {
		a.toast("没有可销毁的缓存页面")
		return
	}
	a.toast("已销毁 " + itoa(n) + " 个缓存页面")
}

// formatMB 把 MB 数值格式化成一位小数。
func formatMB(v float64) string {
	return itoa(int(v)) + "." + itoa(int(v*10)%10) + " MB"
}

// formatPX 滚动位置（DIP）显示。
func formatPX(v float32) string {
	if v <= 0 {
		return "顶部"
	}
	return itoa(int(v)) + " px"
}
