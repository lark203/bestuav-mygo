# 20261103 MyGo 与 CrychicUI 选型结论

## 一、背景

针对 `https://github.com/lark203/CrychicUI.git` 与 `https://github.com/egoist/mygo.git`
两个仓库做对比，判断 `bestuav-mygo` 应当"直接使用"哪一个。

## 二、结论

**两者不是同类竞品，不构成二选一。** 所属层次不同：

| 仓库 | 定位 | 本项目现状 |
| --- | --- | --- |
| `egoist/mygo` | 桌面应用框架：窗口、菜单、托盘、对话框、GPU 自绘原生 UI、打包/更新（Sparkle 式增量） | **已在用**，`require github.com/egoist/mygo v0.7.1`，并以其 `ui` 包写全部界面 |
| `ELY-Open-Source/CrychicUI`（即 `lark203/CrychicUI` 上游） | 架在 mygo 之上的 **原生 UI 组件库**：1500 个组件 / 38 个包，含合成音效 | 未使用 |

因此：**框架层必须继续用 `egoist/mygo`**（没有替代品，CrychicUI 自己 `require` 的也是它）；
**CrychicUI 只能作为可选增量组件层**，用不用是独立决策，不影响"用不用 mygo"。

## 三、事实核验（2026-11-03 实测）

### 3.1 模块身份

| 项 | mygo | CrychicUI |
| --- | --- | --- |
| module path | `github.com/egoist/mygo` | `github.com/ELY-Open-Source/CrychicUI` |
| 有 tag 发布 | 有，至 `v0.7.4` | **无**，只有伪版本 `v0.0.0-20261027050735-bf15ddf6600a`（2026-10-27） |
| 声明 go 版本 | — | `go 1.27.1` |
| 声明框架版本 | — | `require github.com/egoist/mygo v0.5.5` |
| License | MIT | MIT |
| 代码规模 | — | 1974 个 `.go`，其中 325 个 `_test.go`；模块 13 MB（`icon/` 占 2.9 MB，内嵌 Lucide） |

### 3.2 路径陷阱（重要）

用仓库 URL 直接拉取 **会失败**：

```
$ go get github.com/lark203/CrychicUI@latest
go: github.com/lark203/CrychicUI@latest (v0.0.0-20261027050735-...) requires ...:
	module declares its path as: github.com/ELY-Open-Source/CrychicUI
	        but was required as: github.com/lark203/CrychicUI
```

即：`lark203/CrychicUI` 是 fork（末次提交 Oct 20，落后上游 Oct 27），**go 工具链只认上游路径**。
若要改组件源码，必须加 `replace` 指向本地目录，不能靠改 import 路径。

### 3.3 版本兼容性实测（关键风险点已排除）

构造临时模块，同时 `require` CrychicUI（伪版本）与 `mygo v0.7.1`，**空导入其 40 个公开包**后编译：

- `go mod tidy` 后 `go build ./...` → **退出码 0，零错误**。
- 说明 CrychicUI 虽然只在 v0.5.5 上开发，但在 v0.7.1 的 API 面下**全部可编译**（MVS 会取 mygo v0.7.1）。
- 但这只证明**能编译**，不等于行为正确；未做运行时渲染验证（桌面 GUI 需显示器 + GPU）。

### 3.4 体积成本实测

`-trimpath -ldflags="-s -w"`，windows/amd64：

| 组成 | 产物大小 |
| --- | --- |
| 仅 `mygo/ui`（Column + Text + PrimaryButton） | 3,193,856 B（3.05 MB） |
| 再加 CrychicUI `button` + `layout` + `data` | 4,380,672 B（4.18 MB） |

**增量约 +1.13 MB**（连带 `icon`、`sound`、`kit`）。只引用到哪些包就只链接哪些，不会全量带入 13 MB 源码。

## 四、风险与注意事项

1. **无 semver tag**：只能按伪版本锁定，一旦 `go get -u` 会跳到任意提交，API 可能在无预警下变动。
   若引入，`go.mod` 必须写死伪版本，并禁止 `go get -u` 该模块。
2. **上游未覆盖 mygo 升级**：CrychicUI 的 `go.sum` 还停在 `x/image v0.46.0` / `x/net v0.59.0` / `purego v0.11.1`，
   与本项目 `x/image v0.47.0` / `x/net v0.62.0` / `purego v0.11.2` 不同 —— 在本项目里引入需要跑一次 `go mod tidy` 重新对齐。
3. ~~**外观体系冲突**：混用会出现两套圆角/间距/底色语言。~~
   **【2026-11-03 读源码后修正：此判断错误】** 组件的配色、圆角、间距**全部取自宿主主题**
   `c.Theme()`，不是自带一套：
   - `button/solid.go`：`t := c.Theme()`，用 `t.Accent` / `t.AccentHover` / `t.AccentPressed` / `t.AccentText`；
   - `kit/kit.go:81`：`Background(t.Background).Border(1, t.Border).Radius(t.Radius)`；
   - `layout/card.go`：`t.Space(2)` / `t.Space(4)` 取间距。
   README 说的"锌灰底、6 DIP 圆角"只是 **mygo 默认主题**的取值，而本项目同样跑在默认主题上
   （`applyBackdropTheme` 只调 `Background/Surface/Hover/Pressed` 的透明度，不动 `Accent/Border/Text`）。
   结论：CrychicUI 与 `internal/ui/kit` **吃的是同一份 `*ui.Theme`**，会跟随本项目主题与背景图半透明策略，
   不存在"两套设计语言"。真正需要对齐的只是**同一语义两套实现**（都提供 Card/Pill），属冗余而非冲突。
4. **默认音效**：所有可交互组件默认发声（合成音，不读音频文件）。本项目是工控/科学类工具，
   需在启动时 `sound.SetEnabled(false)` 显式关掉。
5. ~~未做运行时验证~~：gallery 已在本机真实窗口跑通并截图（见第六节）；
   组件库接入本项目的编译/vet/测试验证见第七节。

## 五、建议

- **框架：维持 `egoist/mygo`**，无需讨论。可考虑从 `v0.7.1` 升到 `v0.7.4`（修复若干原生 UI 焦点/弹窗问题），
  但升级属独立事项，须单独验证，不在本次结论内。
- **组件库：按"只取局部、不动全局"的方式评估**，不建议整体替换现有 `internal/ui/kit`：
  优先只引 `chart`、`data`（DataTable/Tree/看板）、`datetime`、`code` 这类**本项目自研成本高**的包；
  按钮、卡片、表单这类已有封装且强绑主题的，继续用 kit。
- 引入前需补两项验证：① `go vet ./...` + `go tool mygo vet ./...`；② 真实窗口下跑一次 gallery 级对比截图。

## 六、界面截图（2026-11-03 本机实跑）

上游仓库**不含任何截图**：全模块 0 个 `.png/.jpg/.gif/.webp`，只有 `icon/svg/` 下的 Lucide 图标；
也没有 `docs/`、`screenshots/`、`assets/` 目录。线上 `crychicui.zhangyanghaha0407.workers.dev`
是把 `cmd/gallery` 编译成 WASM 的**可交互 gallery**，不是静态截图页。

因此本轮在本机直接编译并运行官方 gallery（`go build ./cmd/gallery`，28.7 MB），
真实窗口抓图为证，存放于 `docs/screenshots/20261103/`：

| 文件 | 内容 |
| --- | --- |
| `01_home.png` | 首页：1500 组件 / 38 分类 / 17 音效提示；左栏 38 个分类导航；右上 System / Light / Dark 主题切换与音效开关 |
| `02_button_solid.png` | 单组件页范式：标题 + 一行描述 + 实测渲染区 + 下方 Go 代码 |
| `03_data_table.png` | DataTable 真实渲染：表头可排序箭头、5 行示例数据、圆角容器 |
| `04_chart_line.png` | Line 图真实渲染：Y 轴刻度、横向网格、双序列曲线 + 图例 + Shuffle data 按钮 |
| `05_feedback_toaster.png` | Toaster：三个触发按钮（info/success/error）+ 代码区 |

**观察结论（截图直接可见）**：

1. 观感确实是"mygo 自己的样子"：锌灰底、6 DIP 圆角、系统字体、细边框分区，克制、偏工具型。
   （早期判断"两套圆角/间距语言会打架"已在第四节第 3 条更正：组件全部读 `c.Theme()`，会跟随宿主主题。）
2. 分类页只是**索引列表**（编号 + 名称 + 一行说明），真实预览必须点进单个组件页；
   即"1500 个组件"要逐个点开看，无法一屏总览。
3. 每页都附「Code」区展示该组件的实际调用方式，对评估接入成本很友好。
4. 状态栏（底部 Log / Cues）在 1280 宽下文本会被压得很扁，属肉眼可见的细节瑕疵。

复现方式（临时副本中给 `cmd/gallery/main.go` 加了路由启动参数，原仓库未改）：

```bash
go build -trimpath -ldflags="-s -w" -o gallery.exe ./cmd/gallery
./gallery.exe "/c/data/data-table"    # 路由即 /c/<分类>/<组件 slug>
```

## 七、作为普通 Go 包接入本项目的可行性验证（2026-11-03 副本实测）

**验证方式**：把 `bestuav-mygo` 整仓复制到 `D:\tmp-mygo-int`（真仓库零改动），
在其中新增 `internal/view/crychic_probe.go`，用本项目主题真实渲染
`layout.Card` + `button.Solid` + `data.DataTable` + `chart.Line`，然后走完整检查链。

| 检查项 | 结果 |
| --- | --- |
| `go get github.com/ELY-Open-Source/CrychicUI@latest` | 成功；其声明的 mygo v0.5.5 被 MVS 取高，实际仍是本项目 v0.7.1 |
| `go mod tidy` 后依赖变化 | **只新增 1 个 module**（CrychicUI 本身）；`modernc.org/sqlite` 由 indirect 转 direct（本就直连，仅注释修正）；**无任何新增间接依赖** |
| `go.sum` 行数 | 40 → 66（新增的是 CrychicUI 及其测试依赖哈希，非新增运行时模块） |
| `go build` | 通过 |
| `go vet ./...` | 通过（0 输出） |
| `go tool mygo vet ./...` | 通过（0 输出） |
| `go test ./...` | **19 个包全绿**（含 `internal/view` 2.487s、`internal/ui/kit`、`internal/store`） |
| 产物体积 | 21,529,088 B → 22,661,632 B，**+1,132,544 B（+1.08 MB）** |
| `go.mod` 的 `tool` 指令 | 保持不变（`tool github.com/egoist/mygo/cmd/mygo`） |

**结论：可以。** 它是标准 Go 模块，`import` 即用；不引入构建期插件、代码生成或额外工具链，
不改 module 路径与 `tool` 指令，对现有测试零回归。接入后必须遵守四条约束：

1. **版本写死伪版本**（上游无 tag）。推荐加 `replace` 指向自建镜像或直接 vendor，避免 `go get -u` 漂移。
2. **显式静音**：启动时 `sound.SetEnabled(false)`，否则每个可交互组件都会发声。
3. **按需引包**：只 `import` 用到的分类包，体积随引用量线性增长，不会全量带 13 MB 源码。
4. **与 `internal/ui/kit` 的分工定死**：两者都有 Card/Pill 这类同名语义，
   建议"数据密集与复杂交互（表格、图表、日历、代码视图）走 CrychicUI；框架、主题、布局与既有页面维持 kit"，
   否则同一页面出现两套 Card 会让人以为是两个产品。

## 八、附：本次实测所用命令（可复现）

```bash
# 版本清单
go list -m -versions github.com/egoist/mygo
go list -m -json github.com/ELY-Open-Source/CrychicUI@latest

# 兼容性（空导入全包）
go get github.com/ELY-Open-Source/CrychicUI@latest
go get github.com/egoist/mygo@v0.7.1     # 自动从 v0.5.5 升到 v0.7.1
go mod tidy && go build ./...

# 整仓接入验证（在副本内）
go get github.com/ELY-Open-Source/CrychicUI@latest
go mod tidy && go build ./... && go vet ./... && go tool mygo vet ./... && go test ./...
```

验证产物已删除并复核：`D:\tmp-crychic-probe`、`D:\tmp-crychic-probe2`、
`D:\tmp-crychic-gallery`、`D:\tmp-crychic-shots`、`D:\tmp-mygo-int`。
真仓库 `git status --porcelain` 仅显示本文档与 `docs/screenshots/20261103/` 两项未跟踪新增，
`git diff go.mod go.sum` 无输出（依赖未改）。
