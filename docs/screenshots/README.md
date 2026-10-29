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

## 两个必须注意的点

- **进设置页前要抑制 `onEnter`**：设置页的 `onEnter` 会真的发起一次检查更新（网络请求）。截图前把 `a.pagePrev = a.page`，`SyncPages` 就不会触发钩子。
- **更新控件已移到标题栏**，三种状态这样摆：

  ```go
  a.updateChecking = true                                            // 检查中（呼吸点）
  a.availableUpdate = &mygo.Update{Version: "0.1.3", Notes: "…", Date: someDate} // 发现新版本（蓝色胶囊）
  a.showUpdateDialog = true                                          // 再点开确认弹窗
  ```

- **版本号**：`appmeta.BuildVersion()` 在测试二进制里恒为 `dev`。想让截图显示真实版本，用 `go test -overlay` 在**编译期**替换 `internal/appmeta/appmeta.go`（不要改仓库里的文件）：

  ```json
  { "Replace": {
      "D:/workbuddy/bestuav-mygo/internal/appmeta/appmeta.go": "C:/temp/appmeta_shot.go"
  } }
  ```

  覆盖文件内容与原文件相同，只把 `BuildVersion()` 改成 `return "0.1.2"`。

- 尺寸统一 `1100×720`。`12-update-dialog.png` 是「发现新版本」弹窗，原 `12-settings-latest.png` 已废弃删除。
