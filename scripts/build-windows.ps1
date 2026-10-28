# ============================================================
# bestuav-mygo · Windows 打包脚本 (PowerShell)
# ------------------------------------------------------------
# 产出：
#   build/bestuav-mygo-<version>-windows-amd64.exe        （裸可执行文件）
#   build/bestuav-mygo-setup-<version>-amd64.exe          （NSIS 安装包）
# ============================================================
#
# 【前置依赖】
#   1. Go 1.27+，且 `go` 在 PATH 中。
#   2. mygo 工具：本项目 go.mod 已声明 `tool github.com/egoist/mygo/cmd/mygo`，
#      直接用 `go tool mygo` 调用，无需单独安装。
#   3. 本机为 Windows；NSIS 安装包制作工具 mygo 会在缺失时自动下载，
#      无需手动安装。
#   4. （可选）代码签名证书：在 mygo.json 的 windows 段配置，或通过
#      -sign 传入；不配置则生成未签名安装包（用户安装时会有 SmartScreen 提示）。
#
# 【关于 MSI】
#   mygo 原生产出的是 NSIS 制作的 .exe 安装包，而非 .msi。
#   若组织强制要求 .msi，需要另行引入 WiX Toolset 把裸 exe 封装为 .msi，
#   本脚本不覆盖该场景（保持 mygo 标准链路）。
#
# 【跨平台说明】
#   `go tool mygo build` 是纯 Go 交叉编译（无 cgo），在 Windows 上也能
#   同时产出 macOS/Linux 的裸二进制；但 .dmg / .deb 安装包必须在对应
#   系统（macOS / Linux）上才能生成。故本脚本只打 Windows 目标。
#
# 用法（在仓库根目录执行）：
#   pwsh scripts/build-windows.ps1
#   或：powershell -ExecutionPolicy Bypass -File scripts/build-windows.ps1
# ============================================================

$ErrorActionPreference = "Stop"

# 读取版本号（与 mygo.json 保持一致）——简单解析 JSON，避免引入额外依赖。
$config = Get-Content -Raw -Encoding utf8 .\mygo.json | ConvertFrom-Json
$version = $config.version
Write-Host "==> 开始构建 Windows 平台 (version: $version)"

# 调用 mygo 构建 Windows/amd64 目标。
# -platform windows/amd64 只构建 Windows，避免在本机尝试生成其它平台安装包。
# 输出目录由 mygo.json 的 "out" 字段决定（本项目为 build/）。
go tool mygo build -platform windows/amd64

# 列出产物，方便确认。
Write-Host "==> 构建完成，产物位于 build/ ："
Get-ChildItem build\*-windows-* -ErrorAction SilentlyContinue | ForEach-Object { Write-Host "    $($_.Name)" }

Write-Host "==> 完成。双击 bestuav-mygo-setup-*-amd64.exe 即可安装。"
