# 修复 CrychicUI 日期组件零值 panic

- 日期：2026-11-03
- 页面：`internal/view/page_crychic.go` 的 `crychicDatetimePage`（侧栏「CrychicUI 组件 → 日期时间」）
- 组件来源：`github.com/ELY-Open-Source/CrychicUI/datetime`（精确伪版本 `v0.0.0-20261027050735-bf15ddf6600a`）

## 一、问题描述

本地运行 `go run .` 打开日期时间对照页，点击/渲染即崩溃：

```
panic: datetime: DateBadge needs a day [recovered, repanicked]
...
github.com/ELY-Open-Source/CrychicUI/datetime.DateBadge(...)
	.../datetime/date_badge.go:14 +0x265
bestuav-mygo/internal/view.(*App).crychicDatetimePage.func3.1.1()
	D:/workbuddy/bestuav-mygo/internal/view/page_crychic.go:98 +0x45
```

## 二、根因分析

`DateBadge` 的源码（`datetime/date_badge.go:13-15`）对传入的 `day time.Time` 做了防御性校验：

```go
func DateBadge(c *ui.Context, day time.Time) ui.Element {
	if day.IsZero() {
		panic("datetime: DateBadge needs a day")
	}
```

崩溃页面第 98 行调用为 `datetime.DateBadge(c, a.cryPick)`，`a.cryPick` 是 `time.Time`。

排查 `App{}` 初始化字面量发现：`cryPick, cryMini, cryNav, cryHeatFrom time.Time` 四个字段声明在一起，
但字面量里**只有 `cryHeatFrom` 给了初值**，`cryPick`/`cryMini`/`cryNav` 都是零值 `time.Time{}`（`.IsZero()` 为 true）。

进一步通读 `datetime` 全包的 panic 点，确认本页涉及的组件对零值的行为分两类：

| 组件（调用位置） | panic 条件 | 本页是否触发 |
|---|---|---|
| `DateBadge(c, a.cryPick)` (L98) | `day.IsZero()` | **触发（当前报错）** |
| `DateNav(c, &a.cryNav, …)` (L99) | `day == nil \|\| day.IsZero() \|\| unit 越界` | **下一步必然触发**（指针非 nil 但 `IsZero()`） |
| `MiniCalendar(c, &a.cryMini, …)` (L78) | 仅 `date == nil` | 不触发（指针非 nil），但显示 1970 年、marks 错位（观感缺陷） |
| `DatePicker/TimePicker/DateTimePicker` (L69-71) | 仅 `label=="" \|\| date==nil` | 不触发（label 有值、指针非 nil） |
| `CalendarHeat/TimeAgo` (L96/L100) | `from.IsZero()` / `at.IsZero()` | 不触发（用 `cryHeatFrom`，有初值） |
| `AnalogClock/TimePicker` 空 `Options{}` | `Size==0`→默认 40 空间；`Step==0`→默认 5 | 内部已加固，不触发 |

结论：**根因是 `App{}` 初始化遗漏**——同组四个 `time.Time` 字段只初始化了一个，导致其余三个为零值，触碰
`DateBadge`/`DateNav` 的 `IsZero()` 防御 panic。这是"数据未初始化"类缺陷，不是 CrychicUI 或 mygo 的 bug。

## 三、解决方案

在 `internal/view/app.go` 的 `App{}` 初始化段补全初值（仅改初始化，不动任何组件逻辑）：

```go
// CrychicUI 对照页初始数据
cryHeatFrom: time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local),
cryPick:     time.Now(),
cryMini:     time.Now(),
cryNav:      time.Now(),
```

`cryMini` 一并补 `time.Now()` 的原因：它虽不 panic，但 `MiniCalendar` 在零值下会渲染 1970 年、marks 基于 1970 计算，
对照页观感失真，属于应一并消除的缺陷。

## 四、影响范围与测试结果

- 改动文件：`internal/view/app.go`（+3 行初始化），其余页面零改动。
- 依赖：`go.mod` / `go.sum` 未变（datetime 早已是 CrychicUI 子包）。
- `go vet ./...`：退出码 0。
- `go test ./...`：全部 `ok`，`internal/view` 2.019s 通过，无回归。
- 验证手段说明：本机为 headless 沙箱，无法实跑 GUI 截图；panic 消除通过**逐组件核对 panic 条件已消除**证明
  （`cryPick`/`cryNav`/`cryMini` 均已非零点）。**建议本地 `go run .` 打开该页肉眼复核一次**。

## 五、后续建议

1. **CrychicUI 用值类型 `time.Time` 的组件（DateBadge、TimeAgo、DatePill、DeadlineBadge、MoonPhase 等）必须保证传入非零点**；
   用 `*time.Time` 的组件多数只查 `nil`，相对安全，但零值时间会带来观感问题，建议也给定初值。
2. 新增任何 CrychicUI 组件演示前，先 `grep -n "panic(" <pkg>/*.go` 通读其防御条件，避免再次踩零值/空 Options 的雷。
3. 这类"初始化遗漏"缺陷编译期不可见，后续可考虑在 `App` 构造后加一个 `debug` 断言遍历 datetime 字段是否为零值。
