# ============================================================
# bestuav-mygo - Windows build script (PowerShell)
# ------------------------------------------------------------
# Outputs:
#   build/bestuav-mygo-<version>-windows-amd64.exe        (raw executable)
#   build/bestuav-mygo-setup-<version>-amd64.exe          (NSIS installer)
# ============================================================
#
# [Prerequisites]
#   1. Go 1.27+ on PATH (check with: go version).
#      NOTE: go.mod requires go >= 1.27.3; older Go refuses to build.
#   2. mygo CLI: declared as a tool in go.mod
#      (tool github.com/egoist/mygo/cmd/mygo), invoked via `go tool mygo`.
#      No separate install needed; first run auto-builds it (needs network).
#   3. Run on Windows. NSIS is auto-downloaded by mygo when missing.
#   4. (Optional) code-signing cert: configure in mygo.json "windows"
#      section or pass -sign; without it the installer is unsigned and
#      SmartScreen will warn at install time.
#
# [About MSI]
#   mygo produces an NSIS .exe installer, NOT .msi.
#   If .msi is mandated by policy, wrap the raw exe with WiX Toolset
#   separately. This script follows the standard mygo build path only.
#
# [Cross-platform note]
#   `go tool mygo build` is pure-Go cross-compilation (no cgo), so Windows
#   can also emit macOS/Linux raw binaries; but .dmg/.deb packages must be
#   produced on macOS/Linux respectively. This script targets Windows only.
#
# [Usage - MUST run from the REPO ROOT (where mygo.json lives)]
#   powershell -ExecutionPolicy Bypass -File scripts/build-windows.ps1
#   (PowerShell 7 / pwsh also works: pwsh scripts/build-windows.ps1)
#   Do NOT double-click the .ps1; launch it from a PowerShell terminal.
#
# [Encoding note]
#   This file is ASCII-only on purpose: Windows PowerShell 5.1 decodes
#   .ps1 via the system codepage (GBK on Chinese Windows), so a UTF-8
#   file without BOM misreads Chinese bytes and breaks string literals.
#   If you prefer Chinese comments, save this file as UTF-8 WITH BOM.
# ============================================================

$ErrorActionPreference = "Stop"

# Read version from mygo.json (kept in sync with build output naming).
$config = Get-Content -Raw -Encoding utf8 .\mygo.json | ConvertFrom-Json
$version = $config.version
Write-Host "==> Building Windows target (version: $version)"

# Build Windows/amd64 only. Output dir is set by mygo.json "out" (build/).
go tool mygo build -platform windows/amd64

# List produced artifacts for confirmation.
Write-Host "==> Build done. Artifacts in build/:"
Get-ChildItem build\*-windows-* -ErrorAction SilentlyContinue | ForEach-Object { Write-Host "    $($_.Name)" }

Write-Host "==> Done. Double-click bestuav-mygo-setup-*-amd64.exe to install."
