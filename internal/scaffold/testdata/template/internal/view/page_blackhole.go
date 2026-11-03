package view

import (
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
	"bestuav-mygo/internal/view/blackhole"
)

// ---- 官方示例：黑洞（GPU 着色器）----
//
// 移植自 mygo examples/blackhole，代码与三份 .kage 着色器在 internal/view/blackhole 包里。
// 本文件只负责给它一块固定高度的画布区域。

// blackholeSceneKey 黑洞场景在页面私有状态里的键名。
const blackholeSceneKey = "scene"

// blackholePage 黑洞演示页。
//
// 场景对象（着色器 uniform、相机、帧时钟）存在本页的 pageState.data 里：
// 切走再回来接着转；被 idle 回收时随 data 一起丢弃（GPU 资源随之释放）。
func (a *App) blackholePage(c *ui.Context, t *ui.Theme) {
	scene := pageStateOf(a, "official-blackhole", blackholeSceneKey, blackhole.New)
	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "黑洞（GPU 着色器）",
			"光线在引力场中被逐像素反向追踪：吸积盘背面绕到上下、爱因斯坦环、多普勒增亮。"+
				"拖拽旋转 · 滚轮缩放", nil)
		// 画布区域：固定高度 + 圆角裁剪。页面整体是可滚动的，而画布需要一个确定的边界。
		ui.Column(c).Height(430).Radius(kit.TileRadius).Clip().Children(func() {
			scene.View(c)
		})
	})
}
