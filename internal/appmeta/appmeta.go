// Package appmeta 集中存放应用的「产品标识」常量，供全项目统一引用。
//
// 动机：产品名、窗口标题、配置目录名、应用唯一标识符等字符串如果散落在
// main.go / sidebar.go / settings.go / settings_page.go 等多处各自硬编码，
// 一旦需要改名（例如从 demo 名换成正式名）极易改漏、改不一致。把它们收敛到
// 本包一处定义，所有引用方 import 本包并使用常量，即可「改一处、全项目生效」。
//
// 与 mygo.json 的对应关系（改名时两边都要动，Go 常量无法自动同步到 JSON）：
//
//	mygo.json "name"       ≡ Title      （窗口标题 / 打包产品名）
//	mygo.json "identifier" ≡ Identifier （打包标识符 / 单实例锁 key）
//	mygo.json "slug"       ≈ Name 的语义（安装目录名），当前为 quantum
//
// 注意：
//   - Name 同时是 os.UserConfigDir 下的配置子目录名。改名会导致旧用户的
//     settings.json 无法被直接读取，改名时请同步迁移用户配置或接受重新生成。
//   - 本包只放纯字符串/常量，不依赖任何 UI 或业务包，避免循环依赖。
package appmeta

// Name 产品名 / 配置目录名（per-user 配置路径形如 ~/.config/bestuav-mygo）。
// 同时用作窗口状态键（记住窗口大小与位置）。
const Name = "bestuav-mygo"

// Title 窗口标题与界面品牌名（侧栏品牌头、托盘提示、关于页等）。
const Title = "量子科学"

// Identifier 应用唯一标识符，用于单实例锁、打包标识符等。
const Identifier = "com.bestuav.quantum"
