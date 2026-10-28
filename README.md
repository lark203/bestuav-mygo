# 量子科学 · bestuav-mygo

> 基于 [mygo](https://github.com/egoist/mygo) 的纯 Go 原生桌面应用。GPU 自绘、无 webview、跨平台（Windows / macOS / Linux）。

---

## 一、项目简介

`bestuav-mygo` 是「量子科学」业务系统的桌面端，采用 Go 原生 UI 框架 [mygo](https://github.com/egoist/mygo)（v0.7.1）开发。

当前项目处于**骨架成型、功能待填充**阶段：

- ✅ 整体应用布局（系统标题栏 + 左侧分组导航 + 右侧内容区）
- ✅ 内容区**固定页头**：左侧页面标题 + 右侧自定义图标工具条（按钮多时可横向滚动，滚动条不可见）
- ✅ 内容区**浅分割线**：与侧栏品牌头下的浅线严格同高，并兼作**轻量级任务进度条**的位置
- ✅ 跟随系统明暗主题（不自定义标题栏、不使用固定强调色）
- ✅ 左侧菜单按功能域重排、可折叠、展开可滚动、设置项固定底部
- ✅ 侧栏可整体收缩为图标栏（点击品牌头切换，仅图标 + 悬浮提示，状态持久化）
- ✅ 每项菜单独立 Lucide 风格线性图标
- ✅ 窗口 / 任务栏 / 启动器图标（运行时 `SetIcon` + 构建时自动嵌入）
- ✅ 系统托盘（通知区域常驻图标）
  - **双击托盘图标打开主面板**（Linux 上为双击激活；Windows/macOS 上为单击，mygo 托盘模型未暴露独立双击事件）
  - **右键托盘图标弹出菜单**：「打开主界面 / 退出」
- ✅ 关闭按钮行为可配置（见下方「设置与持久化」）
- ✅ **背景图片** + **菜单区 / 内容区分区透明度**（可一键恢复默认）
- ✅ 设置页（分区域：外观 / 背景 / 窗口与关闭 / 启动 / 关于）
- ✅ 主题可切换：跟随系统 / 浅色 / 深色（不再仅依赖系统，可在设置中手动指定）
- ✅ 字体大小可调（滑块 11–20px，全局生效）
- ✅ 单实例 / 多实例可选（默认单实例；开启后每次启动都是独立窗口）
- ✅ 产品标识字符串集中到 `internal/appmeta`，改名一处生效
- ✅ 标准 Go 目录结构 + 三平台打包脚本
- 🚧 业务功能（生成 license、公钥私钥、异步任务等）以占位页呈现，后续按 `internal/` 业务包逐步落地

---

## 二、技术栈

| 项 | 选型                                                                                      |
|---|-------------------------------------------------------------------------------------------|
| 语言 | Go 1.27+（CGO **不**需要，纯 Go 交叉编译）                                                |
| UI 框架 | [egoist/mygo](https://github.com/egoist/mygo) v0.7.1（即时模式 UI，GPU 自绘，无 webview） |
| 图标 | 内联 SVG（`ui.MustParseSVG`），线性 + `currentColor` 风格（Lucide 思路）                  |
| 打包 | mygo CLI（`go tool mygo build`），纯 Go 跨平台编译                                        |
| 测试 | `ui.NewTester` 无窗口渲染断言（文本 / 交互 / 像素级几何），不依赖显示器                      |

---

## 三、目录结构（最终结构，B 方案：入口留根）

```
bestuav-mygo/
├── main.go                     # 极薄入口（package main）：单实例锁 + WhenReady + NewWindow + 托盘 + 图标 + 注入重绘触发器
├── mygo.json                   # 打包配置；name/identifier 与 internal/appmeta 的 Title/Identifier 需人工保持一致
├── go.mod / go.sum             # 模块名 bestuav-mygo
├── resources/
│   └── icon.png               # 应用图标（1024×1024 PNG），构建与运行时共用
├── internal/
│   ├── view/                  # package view：UI 聚合层（唯一允许 import ui 的业务侧包）
│   │   ├── app.go             #   App 状态（page/dialog/各分组折叠/背景图/进度）+ View 分发器 + 弹窗
│   │   ├── content.go         #   内容区：固定页头（标题 + 可横向滚动的工具条）/ 分割线槽位与进度条 / 可滚动页面
│   │   ├── sidebar.go         #   侧栏品牌头 + 分组导航 + 固定底部「设置」项
│   │   ├── settings_page.go   #   设置页：外观 / 背景 / 窗口与关闭 / 启动 / 关于
│   │   ├── pages.go           #   首页 / 概览卡 / 产品表 / 占位页 / 弹窗实现
│   │   ├── icons.go           #   Lucide 风格 SVG 图标集（菜单图标 + 页头工具条图标）
│   │   └── view_test.go       #   无窗口 UI 断言（含「两条分割线同一行」的像素级断言）
│   ├── appmeta/               # 产品标识常量（名称/标题/应用标识）——改一处全项目生效
│   ├── license/               # 业务包：生成 license（纯 Go，可独立单测，禁止 import ui）
│   ├── crypto/                # 业务包：公钥私钥 / 密码生成
│   ├── tasks/                 # 业务包：异步任务
│   └── settings/              # 业务包：设置持久化（含 settings_test.go 的存取/迁移/归一化测试）
├── scripts/                   # 三平台打包脚本（注释详尽）
│   ├── build-windows.ps1      #   Windows：裸 exe + NSIS 安装包
│   ├── build-macos.sh         #   macOS：.app + .dmg（需 macOS 环境）
│   └── build-linux.sh         #   Linux：裸二进制 + .deb + 便携 tar.gz（需 Linux 环境）
├── docs/                      # 设计 / 重构 / 验证记录
├── .agents/                   # WorkBuddy 维护技能
└── .gitignore
```

### 依赖方向（无环）

```
main → internal/view → { license, crypto, tasks, settings, appmeta }
main → internal/appmeta（窗口标题、窗口状态键）
```

- `internal/view` 是唯一的 UI 聚合层，持有 `App` 状态与所有视图；业务包**绝不**反向 `import ui` 或 `view`，以保证可脱离窗口独立单测。
- `view` → 业务包通过**传参调用**（而非反向 import）拿到结果，规避循环依赖。
- `internal/appmeta` 只放常量，不依赖任何包，可被任意层引用（含 `main`）。

---

## 四、环境要求

| 依赖         | 说明 |
|--------------|---|
| Go 1.27+     | 构建本项目 |
| mygo CLI     | 已在 `go.mod` 以 `tool github.com/egoist/mygo/cmd/mygo` 声明，直接 `go tool mygo dev/build` 即可（首次运行自动编译，需联网）；亦可 `go install github.com/egoist/mygo/cmd/mygo@latest` |
| Windows 打包 | 本机 `build-windows.ps1`（mygo 原生产出 NSIS `.exe` 安装包） |
| macOS 打包   | 需在 **macOS** 上跑 `build-macos.sh`（`.dmg`/签名需 Xcode + Developer ID） |
| Linux 打包   | 需在 **Linux** 上跑 `build-linux.sh`（`.deb` 需 `dpkg-dev`/`fakeroot`） |

> 裸二进制可跨平台交叉编译；但 **`.dmg` / `.deb` 安装包必须在对应目标系统生成**。

---

## 五、快速开始

```bash
# 1. 安装 mygo CLI（仅需一次）
go install github.com/egoist/mygo/cmd/mygo@latest

# 2. 拉取依赖
go mod download

# 3. 开发模式（热重载，窗口标题栏与任务栏显示图标）
go tool mygo dev

# 4. 运行全部测试（无需显示器）
go test ./...

# 5. 仅编译校验
go build ./... && go vet ./...
```

---

## 六、打包发布

各平台脚本均已写好详细注释（前置依赖、签名、跨平台限制、编码说明）。直接用即可：

### Windows（`build-windows.ps1`）

> ⚠️ **必须先 `cd` 到仓库根目录**（脚本内 `.\mygo.json` 为相对路径）。**不要双击 `.ps1`**，从 PowerShell 终端显式调用。本机若只有 PowerShell 5.1（`powershell.exe`）、未装 PowerShell 7（`pwsh`），请用下面第一条命令。

```powershell
# 推荐：在仓库根目录执行（PowerShell 5.1 也能跑；-ExecutionPolicy Bypass 绕过默认脚本策略）
powershell -ExecutionPolicy Bypass -File scripts/build-windows.ps1

# 若已装 PowerShell 7，也可：
pwsh scripts/build-windows.ps1
```

- **编码说明**：该脚本刻意写成**纯 ASCII**（英文注释 + 英文输出）。原因——Windows PowerShell 5.1 按系统代码页（中文 Windows 为 GBK/CP936）解码 `.ps1`，UTF-8 无 BOM 的中文会被误读、导致字符串引号被提前闭合而报错。若你一定要中文注释，请把文件存为 **UTF-8 带 BOM** 再改。
- 首跑会联网自动下载 NSIS 制作安装包。
- **产出（均在 `build/` 目录）**：
  - `bestuav-mygo-<ver>-windows-amd64.exe`（裸可执行文件）
  - `bestuav-mygo-setup-<ver>-amd64.exe`（NSIS 安装包，双击即可安装）

### macOS（`build-macos.sh`）

```bash
# 必须在 macOS 上执行（.dmg 与签名/公证需 Xcode + Developer ID）
./scripts/build-macos.sh
# 产出（build/）：bestuav-mygo-<ver>-darwin-universal.app + bestuav-mygo-<ver>-darwin-universal.dmg
```

### Linux（`build-linux.sh`）

```bash
# 必须在 Linux 上执行（.deb 需 dpkg-dev / fakeroot）
./scripts/build-linux.sh
# 产出（build/）：bestuav-mygo-<ver>-linux-amd64（裸文件）
#                + bestuav-mygo-<ver>-linux-amd64.deb
#                + bestuav-mygo-<ver>-linux-amd64.tar.gz（含 install.sh，装到 ~/.local）
```

> ⚠️ **关于 `.msi`**：mygo 原生产出的是 **NSIS 制作的 `.exe` 安装包**，并非 `.msi`。若下游强制要求 `.msi`，需另行引入 WiX Toolset 将裸 exe 封装，属于 mygo 标准链路之外的额外步骤（已在 `build-windows.ps1` 注释中说明）。
> ⚠️ **关于安装包平台限制**：裸二进制可跨平台交叉编译，但 `.dmg` / `.deb` 安装包**必须在对应目标系统生成**（`.dmg` 需 macOS，`.deb` 需 Linux）。

---

## 七、设置与持久化

用户偏好保存在 **per-user 配置目录下的 JSON 文件**（业界桌面应用标准做法，优于数据库）：

- Windows：`%APPDATA%\bestuav-mygo\settings.json`
- macOS：`~/Library/Application Support/bestuav-mygo/settings.json`
- Linux：`~/.config/bestuav-mygo/settings.json`

写入采用「临时文件 + 重命名」原子写，避免写一半断电导致配置损坏；文件缺失或解析失败时回退默认并归一化脏数据（实现见 `internal/settings`）。

> 兼容说明：早期版本把 `settings.json` 直接放在用户配置目录根下（没有按应用名分子目录）。启动时若新位置不存在而旧位置存在，会自动读取旧文件并迁移到新位置，用户偏好不丢。

可配置项（在设置页内修改并即时持久化）：

| 项 | 取值 | 说明 |
|---|---|---|
| `theme` | `system` / `light` / `dark` | 跟随系统 / 浅色 / 深色，运行时切换 |
| `fontSize` | 11–22（px，设置页滑块 11–20） | 文本基准字号，全局生效 |
| `closeMode` | `ask` / `minimize` / `quit` | 关闭按钮行为 |
| `sidebarCollapsed` | `true` / `false` | 侧栏整体是否处于收缩（图标栏）状态；默认 `false`（展开） |
| `allowMultiInstance` | `true` / `false` | 是否允许同时运行多个实例；默认 `false`（单实例），**需重启生效** |
| `backgroundImage` | 图片绝对路径 / `""` | 背景图片；空串表示不使用（主题默认外观） |
| `menuOpacity` | 0.2–1.0 | 侧栏浮层「不透明度」；仅设置背景图后生效，默认 0.80 |
| `contentOpacity` | 0.2–1.0 | 内容区浮层「不透明度」；仅设置背景图后生效，默认 0.88 |

**侧栏展开 / 收缩**：

- 点击左上角「品牌头」（原子图标）即可在**展开 ↔ 收缩**之间切换：展开为 220px 的「图标 + 文字」布局；收缩为 64px 的图标栏（rail），仅显示图标，鼠标悬停提示菜单名。
- 切换状态即时写入上面的 `settings.json`，**下次启动自动保持现状**。
- 展开时菜单图标、品牌头图标、底部「设置」图标的中心在同一条竖线上。

**背景图片与透明度**：

- 设置页「背景」区可「选择图片…」（系统原生文件选择框）或「恢复默认」（清除背景图并复位两个不透明度）。
- 透明度拆成**菜单区**与**内容区**两个独立滑块，调节的是「不透明度」：100% = 完全遮蔽背景图，数值越小背景图越透出。
- 实现方式：背景图铺满窗口最底层（`ui.Image(...).Fit(ui.Cover)`），侧栏与内容区各自画一层半透明主题色浮层。未设置背景图时两处浮层恒为不透明，界面与主题默认外观完全一致。

**多实例**：

- 默认单实例：重复启动的进程会把参数转发给已在运行的实例后退出，首个实例把窗口显示出来（`RequestSingleInstanceLock` + `OnSecondInstance`）。
- 设置页「启动」区打开「允许多开」后，每次启动都是独立窗口；该开关**需重启应用**才生效（单实例锁在进程启动时获取）。

**关闭按钮行为**：

- `ask`（默认）：首次点击关闭弹出选择弹窗（最小化到托盘 / 直接退出），可勾选「记住我的选择」持久化；之后按所选执行，也可在设置页随时改回「每次询问」。
- `minimize`：点击关闭最小化到托盘。
- `quit`：点击关闭直接退出应用。

---

## 八、内容区布局与轻量任务进度

内容区（`internal/view/content.go`）自上而下分三段：

1. **固定页头**（高度 `headerHeight = 52`，不参与滚动）：左侧页面标题，右侧「自定义图标按钮」工具条。
   - 工具条放在横向滚动容器里：按钮少时按内容宽度靠右排列；空间不足时被压缩到可用宽度，超出部分**横向滚动**，滚动条已全局透明化（画不出来但仍可滚）。
   - 每个按钮是 28×28 的点击区，悬停有圆角底色，悬浮提示说明用途；按钮清单按页面在 `pageTools(page)` 中声明（首页给了 8 个用于演示溢出滚动）。
2. **分割线槽位**（高度 `dividerSlot = 3`）：常态画一条 1px 浅线；执行轻量级任务时**同一槽位**改画进度条（`ui.Progress`）。
   - 槽位高度固定，所以进度条出现/消失都不会让下方内容跳动。
   - 这条线与侧栏「品牌头下的浅线」严格同高：两侧都由 `Space(1) + 52` 得出（内容区根容器上内边距 4px + 页头 52px），测试 `TestDividersOnSameRow` 用离屏渲染的真实像素断言二者落在同一行。
3. **可滚动内容区**：各页面本体，`ui.Scroll` + 内边距 24/28。

**接一条真实轻量任务**：

```go
a.startLightTask("导出")   // 任务名会显示在页头标题右侧
// 真实业务把内部模拟循环换成自己的工作；
// 完成后调 a.endTask()（进度归零，分割线恢复常态）
```

进度由后台 goroutine 推进，通过 `SetInvalidator(win.Invalidate)` 注入的重绘触发器请求重绘（`mygo.RunOnMain` 保证在主线程改状态）。

---

## 九、图标说明

- 源文件：`resources/icon.png`（1024×1024 PNG）。
- **运行时**：`main.go` 通过 `//go:embed` 将图标编入二进制，在 `win.OnReadyToShow` 中调用 `(*Window).SetIcon`，保证开发模式（`go tool mygo dev`）窗口标题栏左上角与任务栏图标可见。
- **构建时**：mygo 自动把 `resources/icon.png` 嵌入安装包（Windows `.syso` / macOS `.icns` / Linux `.desktop`）。
- 如需给开发版单独配图标，可放 `resources/icon-dev.png`，mygo 会自动优先采用。

---

## 十、主题与视觉约定

- 主题来源：默认 `mygo.Theme.SetSource(mygo.ThemeSystem)`——**跟随操作系统明暗**；可在设置页切换为固定浅色 / 深色（`settings.json` 的 `theme`）。
- 不使用自定义强调色与自定义标题栏；侧栏选中态、图标高亮均取自**系统强调色**（`t.Accent`）。
- 图标统一线性 `currentColor` 风格，选中/悬停随系统文字色与强调色自适应。
- **滚动条统一透明化**：每帧 `t.Scrollbar = t.Scrollbar.Alpha(0)`，滚动能力（滚轮 / 触控板 / 横向滚动）保留，只是不绘制滚动条。这是全局行为，侧栏、页头工具条、正文滚动区一致。
- **背景图分层**：背景图铺满窗口最底层（`Fit(ui.Cover)`），侧栏用 `t.Surface.Alpha(menuOpacity)`、内容区用 `t.Background.Alpha(contentOpacity)` 画半透明浮层；未设置背景图时两者均不透明，与主题默认外观完全一致。

---

## 十一、开发约定

1. **UI 全部在 `internal/view`**；纯业务逻辑下沉到 `internal/{license,crypto,tasks,settings}`，且不得 `import ui`。
2. **产品标识字符串只写一处**：应用名、窗口标题、配置目录名、应用标识符统一取自 `internal/appmeta`（`Name` / `Title` / `Identifier`）。改名时只改这一个文件，全项目生效，不会出现「某个类里还留着旧名字」的漏改。新增此类全局字符串时也放进 `appmeta`，不要在业务代码里硬编码。
3. **Go 跨文件无需 import**：同属 `package main` / `package view` 的文件共享同一命名空间，方法/函数直接互相调用（可见性由首字母大小写决定）。这点和 Java「换文件常需 import」的体感不同——根因是 Go 的编译单元是「包」而非「文件」，且目录=包=编译单元三者绑定。
4. **测试用 `ui.NewTester`（无窗口、无需显示器）**：可断言文本与交互（品牌头、侧栏分组、点击弹窗、设置页各区块等），也可用 `Tester.Image()` 取真实像素做几何断言——例如 `TestDividersOnSameRow` 就靠扫描像素行来保证「侧栏浅线」与「内容区浅线」严格同高。需要肉眼确认整体观感时，也可在临时测试里把 `Image()` 存成 PNG 查看。
5. 菜单新增项：在 `internal/view/sidebar.go` 的 `navMenu` 对应分组 `[]navItemDef` 中追加一项（`id` / `icon` / `label`），并配套 `app.go` 的 `pageTitles` 与（如需要）占位/真实页面。
6. 页头按钮新增项：在 `internal/view/content.go` 的 `pageTools(page)` 里追加 `toolAction{icon, label, tip}`（图标在 `icons.go` 中定义）。

---

## 十二、Roadmap（后续功能开发主战场）

- [ ] `internal/license`：license 生成与校验
- [ ] `internal/crypto`：公钥/私钥、用户密码生成
- [ ] `internal/tasks`：异步任务调度与进度（页头进度条骨架已就绪，见 `startLightTask`）
- [x] `internal/settings`：设置项持久化（主题 / 字号 / 关闭行为 / 侧栏收缩 / 背景图与浮层透明度 / 多实例）
- [ ] 国际化（mygo `I18n`）、自动更新、安装包签名与公证
- [ ] 真实业务页面替换占位页

---

## 十三、许可证

内部项目，详见仓库内许可证文件（如有）。
