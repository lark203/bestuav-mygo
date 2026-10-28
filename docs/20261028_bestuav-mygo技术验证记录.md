# BestUAV-mygo 技术验证记录

> 日期：2026-10-28
> 目的：GUI 语言选型 —— 试水 `egoist/mygo` 的**原生 UI 路线**（纯 Go，GPU 自绘，无 webview），并**还原 bestuav-javafx 的整体框架**（截图参照）。

## 一、项目坐标与来源

- 框架：`github.com/egoist/mygo` **v0.7.1**（作者 egoist，MIT，2026-09-26 建库，极年轻）
- 初始化：`go run github.com/egoist/mygo/cmd/mygo@latest init -template native bestuav-mygo`
- 本机 Go：go1.27.3（满足 mygo 要求的 Go 1.27+）
- 参照源码：`D:\workbuddy\bestuav-javafx`（同款管理后台，AtlantaFX 主题）
- 注意：`ufec/mygo` 是落后主线 427 commits 的 fork，不要拉错。

## 二、已搭建内容（还原截图框架）

以 bestuav-javafx 运行时的真实样子为参照，用原生 UI 复刻其整体框架：**自定义深色标题栏 + 左侧导航 + 右侧内容区 + 紫色强调色**。

| 区块 | 实现（mygo 原生控件） |
|---|---|
| 自定义标题栏 | `TitleBarStyle: TitleBarHidden` + `TitleBarHeight: 44`，自绘 `Row`：左侧紫色标志 + "量子科学"，右侧网络/铃铛图标；`DragWindow()` 让整条标题栏可拖动窗口；`c.TitleBar()` 的 `Left/Right` 预留系统窗口控制按钮空间 |
| 左侧导航 | `Sidebar`（`selected *string`）：首页为顶层 `SidebarItem`；图标/个人资料/组件演示/功能演示 为 `SidebarSection`（带箭头可折叠 + 子项缩进）；其余为顶层 `SidebarItem`，菜单顺序与截图一致 |
| 首页内容 | 页头"首页" → "项目概览"区块（标题 + "欢迎来到AtlantaFX 世界！" + **紫框描边按钮"点击交互"**）→ "概览卡片"三张卡（总销售额/新订单/活跃用户，含图标）→ "最近产品" `Table`（名称/分类/价格，4 行 mock 数据） |
| 弹窗 | 点击"点击交互" → `Modal`"系统通知"（"已应用到 config.json" + "我知道了"关闭） |
| 其它页面 | 非首页菜单项渲染统一占位页（标题 + "演示占位页面"），体现导航切换框架 |
| 主题 | `mygo.Theme.SetSource(mygo.ThemeDark)` 强制深色；在 `view` 内把 `Accent` 系列覆盖为紫色 `#7c5cff`（截图签名色） |

关键文件：`main.go`（状态 + 视图 + 图标集）、`main_test.go`（无窗口 tester 断言）、`mygo.json`（打包配置）。

## 三、验证结论（实测，非传闻）

| 项 | 结果 |
|---|---|
| `go vet ./...` | 通过 |
| `go test ./...`（无窗口 tester） | **5 项全绿**，0.070s（含点击"点击交互"弹窗的交互断言） |
| 编译产物体积 | **14.4 MB**（未 strip；加 `-ldflags="-s -w -H windowsgui"` 可显著更小） |
| 进程启动后常驻内存 | **~62 MB**（与官方宣称原生 UI ~44MB 同量级） |
| 运行时初始化 | 进程成功拉起、`App.Run` 阻塞事件循环、**无 panic** |

**验证方式符合工程师规范**：优先用框架自带的"无窗口 tester"（`ui.NewTester`）做渲染与交互断言（含 `Click("点击交互")` → 弹窗出现），不靠肉眼截图下结论。

## 四、控件可用性评估（对 bestuav 后台）

- **充足**：`Table`（排序/选中/虚拟滚动）、`Sidebar`/`SidebarSection`、`Modal`、`Form` 全家桶（`Switch`/`Select`/`Slider`/`NumberInput`）、`Badge`、`Progress`、`Toast`、`Image`(SVG) 一应俱全，足以撑起管理后台。
- **已验证可行**：自定义标题栏（`TitleBarHidden` + `DragWindow` + `c.TitleBar()` 预留系统按钮）、运行时改 `Accent` 强调色、`ui.Hex` 解析颜色、`ui.MustParseSVG` 内联图标。
- **踩到的坑（已解决）**：
  - `c.Theme()` 返回 `*ui.Theme`（指针），页面函数参数须用指针类型。
  - `Row().Wrap()` 为**无参**方法，不能传 `Wrap(true)`。
  - mygo v0.7.1 **没有描边按钮变体**（无 `OutlineButton`/`.Outline()`），"点击交互"紫框按钮用 `Button().Background(t.Background).Border(1, t.Accent).TextColor(t.Accent)` 手动实现。
  - 无窗口 tester 只能 `Click` 按钮/输入框文本，**点不到 `SidebarItem` / `Switch`**，故侧栏导航点击、文件夹展开、设置开关的真实交互需在有显示器的桌面会话里验证。

## 五、限制与未决风险

1. **本沙箱无显示器**，无法验证窗口的真实渲染与鼠标交互（GPU/Direct3D 在有头会话才出画面）。用户本机需执行 `go tool mygo dev` 实测：标题栏拖动、窗口控制按钮位置、侧栏导航、文件夹展开、弹窗、紫框按钮。
2. **v0.7.1 极早期**：API 仍可能变动，仓库 84 个 open issues；不要现在把核心产品压上去。
3. 原生 UI 走 Metal/D3D/OpenGL，**依赖显卡驱动质量**，Windows 上尤其要吃驱动。
4. 当前仅为"整体框架"还原，业务页（生成 license、异步任务、组件演示各子页等）仍是占位，未接真实逻辑。

## 六、后续建议（按优先级）

1. 在本机有显示器的会话跑 `go tool mygo dev`，逐项点测：标题栏拖动/系统按钮、侧栏导航与文件夹展开、紫框按钮弹窗、占位页切换。
2. 对照 `go tool mygo build` 打一次安装包，确认签名/公证/自动更新链路在 Windows 上的可行性。
3. 若决定采用，再开一个 webview 路线分支，做体积/内存/开发成本横向对比（这是 mygo 相对 Wails 的独有卖点）。
4. 把占位页逐步替换为真实功能（优先 生成 license / 异步任务 / 组件演示），数据层从 mock 切到 `bestuav-go` 真实接口（注意 clientid 头、api-decrypt 等已知坑）。
