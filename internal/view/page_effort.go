package view

import (
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
	"bestuav-mygo/internal/view/effort"
)

// ---- 官方示例：投入滑块（自绘动画）----
//
// 移植自 mygo examples/effort-slider（它还原的是 ozzy 那个 "galaxy brain effort
// slider"）：一张卡片，滑块从 Low 一路到 Max，拇指是一颗大脑；再往右推过 Max 就进
// Galaxy —— 大脑炸开、卡片抖一下、轨道里填满闪烁的像素、点状光线从大脑里射出来。
//
// 整张卡片是**自绘**的（effort 包用 ui.Painter 逐帧画路径、渐变、位图光晕），页面里
// 唯一的真控件是压在轨道上的 SliderBase：它一个像素都不画，只负责接指针与方向键、
// 并让辅助技术看见「这里有个滑块」。所以既有好看的动画，也不牺牲键盘与读屏。
//
// 两个移植时的坑（都写在 effort 包里了）：
//  1. 模糊的字形要靠 ui.Render 离屏先栅格化，而 ui.Render 自己要跑一整帧，
//     不能在窗口的某一帧里调用 —— 所以 Slider 在「第一次进入本页」时由主循环任务
//     在帧外构建（见 effortSlider），启动时不再预建；
//  2. 示例自己是暗色调的（还原网页），不跟主题走，故卡片外的底色由它自己铺。

// effortCanvasH 画布高度：直接跟 effort 包要（卡片 129.5 居中，剩下的留给阴影）。
const effortCanvasH = effort.CanvasH

// 该页的 id（与 registry 里登记的一致）与状态键名。
const (
	effortPageID      = "official-effort"
	effortSliderKey   = "slider"
	effortBuildingKey = "slider-building"
)

// effortSlider 取本页的滑块；还没构建就先排一次「帧外构建」，本帧返回 nil 画占位。
//
// 为什么不在 newApp 里预建：effort.New() 要调 ui.Render 做一次离屏栅格化，实测占
// 启动耗时的 61%（2.36ms 里的 1.4ms、1MB 内存），而这个演示页用户可能一辈子不打开。
// 挪到「第一次进这一页」既不影响观感（一次构建约 1.4ms，下一帧就绪），又让启动变快。
func (a *App) effortSlider() *effort.Slider {
	st := a.pageStat(effortPageID)
	if s, ok := st.data[effortSliderKey].(*effort.Slider); ok {
		return s
	}
	if st.data[effortBuildingKey] == nil {
		st.data[effortBuildingKey] = true
		go func() {
			// RunOnMain 的任务在主循环里执行，不在任何一帧的中间 —— 这正是
			// ui.Render 要求的「帧外」。单测里没有事件循环，任务会被丢弃，
			// 页面就停在占位态（由 newTestApp 显式建好来回避）。
			mygo.RunOnMain(func() {
				a.buildEffortSlider()
				delete(a.pageStat(effortPageID).data, effortBuildingKey)
				a.requestFrame()
			})
		}()
	}
	return nil
}

// buildEffortSlider 构造滑块并放进页面状态，返回它。
//
// 只允许在帧外调用（newApp 之前、主循环任务里、或单测里）。
func (a *App) buildEffortSlider() *effort.Slider {
	s := effort.New()
	a.pageStat(effortPageID).data[effortSliderKey] = s
	return s
}

// effortPage 投入滑块演示页。
func (a *App) effortPage(c *ui.Context, t *ui.Theme) {
	s := a.effortSlider()
	if s == nil {
		// 占位：构建只差一帧，这里不显示「请重新打开」这类吓人的话。
		kit.Card(c, t, func() {
			kit.CardHeader(c, t, "投入滑块（自绘动画）",
				"首次打开需要先离屏栅格化一次字形表，随后就是逐帧自绘", nil)
			ui.Text(c, "正在准备画布…").TextColor(t.TextMuted)
		})
		return
	}

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "投入滑块（自绘动画）",
			"拖动大脑、点击轨道，或点一下轨道再按方向键；推到最右进 Galaxy：大脑炸开、卡片抖动、轨道填满闪烁像素",
			func() {
				ui.Text(c, "当前档位 "+s.Level()).
					FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			})

		// 工具条：给不方便拖动的人一条键盘之外的路径（离屏测试也靠它们点）。
		ui.Row(c).Gap(kit.CardGap).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "上一档").Clicked() {
				s.SetLevel(max(s.LevelIndex()-1, 0), time.Now())
			}
			if ui.Button(c, "下一档").Clicked() {
				s.SetLevel(min(s.LevelIndex()+1, effort.Galaxy), time.Now())
			}
			if ui.Button(c, "推荐档位").Tooltip("Medium —— 轨道上那根竖条就是它").Clicked() {
				s.SetLevel(1, time.Now())
			}
			if ui.Button(c, "拉满（Galaxy）").Clicked() {
				s.SetLevel(effort.Galaxy, time.Now())
			}
			ui.Spacer(c).Grow(1)
			ui.Text(c, effortNote(s)).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})

		// 画布：固定高度 + 撑满宽度（不加 Fill()，否则高度会被父容器压回最小）。
		// Slider 自己在里面居中，四周铺示例的暗底色；底色是铺满的矩形，
		// 圆角与裁剪由这一层负责。
		ui.Box(c).Height(effortCanvasH).FillWidth().Radius(kit.CardRadius).Clip().
			Children(func() {
				s.View(c)
			})

		ui.Text(c, "卡片里每个像素都是 Painter 画的：圆角按 CSS 的 border-radius 走三次贝塞尔，"+
			"换档时档位文字逐个字母上浮、锐化、淡入（模糊是 CPU 上的高斯核），"+
			"Galaxy 的光晕是预生成的位图 —— Painter 没有形状模糊，只能拿位图顶。"+
			"这些手法都在 internal/ui/{anim,canvas} 里，不止这张卡片能用。").
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})
}

// effortNote 状态行右侧的一句话：Galaxy 里才有话说，平时只提示操作方式。
func effortNote(s *effort.Slider) string {
	if s.LevelIndex() == effort.Galaxy {
		return "已进 Galaxy：卡片在抖、轨道在闪、光线在射 —— 退回来时这些都会淡出"
	}
	return "第 " + strconv.Itoa(s.LevelIndex()+1) + " / 6 档 · 拖动或按方向键"
}
