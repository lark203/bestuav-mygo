# 界面截图

README 里引用的截图都在这里，全部由 **离屏渲染** 产出（不需要显示器）。

## 如何重新生成

临时新建 `internal/view/zz_shots_test.go`（跑完删掉，不要提交），内容大致如下：

```go
package view

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"bestuav-mygo/internal/settings"
)

func TestShots(t *testing.T) {
	s := settings.Default()
	s.BackgroundImage = `C:/path/to/your/wallpaper.jpg` // 留空则为纯主题背景
	s.MenuOpacity, s.ContentOpacity = 0.54, 0.58

	shots := map[string]func(a *App){
		"01-home.png":       func(a *App) {},
		"02-task.png":       func(a *App) { a.page = "task" },
		"03-license.png":    func(a *App) { a.page = "gen-license"; a.generateLicense() },
		"04-basic-info.png": func(a *App) { a.page = "basic-info" },
		"05-components.png": func(a *App) { a.page = "comp-buttons" },
		"06-features.png":   func(a *App) { a.page = "fn-todo" },
		"07-settings.png":   func(a *App) { a.page = "settings" },
		"08-mask.png": func(a *App) {
			a.page = "task"
			a.heavyRunning, a.heavyLabel, a.heavyProgress = true, "重建检索索引", 0.62
		},
	}
	for name, setup := range shots {
		a := newApp(s)
		a.ensureBackground()
		setup(a)
		img := render(a).Image()
		f, _ := os.Create(filepath.Join("../../docs/screenshots", name))
		_ = png.Encode(f, img)
		f.Close()
	}
}
```

要点：

- `render(a)` 就是 `ui.NewTester(a.View, 1100, 720)`，无窗口；
- 彩带这种动画需要 `tt := render(a); a.celebrate(4); for i:=0;i<8;i++ { tt.Advance(60*time.Millisecond); tt.Frame() }` 再取 `tt.Image()`；
- 截图会带上当前主题（浅色/深色）与背景图，想要深色版就把 `settings.Default().Theme` 改成 `"dark"`。
