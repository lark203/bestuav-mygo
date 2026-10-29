# 发布与更新手册

> 面向维护者：**客户端怎么更新、发一个版本要在 GitHub 上点什么、产物分别是什么**，都在这份文档里。
> 最后核对：2026-10-30（`v0.1.1` 已正式发布，12 个资产；三平台 CI 全绿）

---

## 一、先理解这套机制（3 分钟）

三个事实决定了下面所有操作：

1. **主仓库在 AtomGit，更新源在 GitHub。**
   mygo 的自动更新只认 **GitHub Release**（或自托管 URL），所以代码要镜像到 GitHub，且**发版 tag 必须推到 GitHub**；AtomGit 只做代码备份，推它**不会**触发任何构建。

2. **更新源是编译进二进制的。**
   `mygo.json` 的 `updates.github` 在 `go tool mygo build` 时被链接进 exe；**改了仓库地址就必须重新发版**，旧版本永远指向旧地址。

3. **draft 对客户端不可见。**
   CI 把产物传到一个 **draft release**，你**手动 Publish** 之后，已安装客户端才看得到更新。

数据流：

```
git tag v0.1.2 ──push──► GitHub
                           │
                    GitHub Actions（4 个作业，串行，约 8 分钟）
              ┌────────────┼────────────┐
        windows-latest  macos-latest  ubuntu-latest
              └────────────┼────────────┘
                   draft release（12 个资产）
                           │  人工 Publish
                           ▼
                  正式 Release ◄──── 客户端 Updater.Check
                           │            （GitHub API 列 releases，
                           │              取最新「非 draft、非 prerelease」的 tag）
                           ▼
             下载 tar.gz → ed25519 验签 → 就地替换 → 重启
```

---

## 二、一次性配置（只做一次）

### 2.1 GitHub 仓库

- 仓库：`lark203/bestuav-mygo`，**必须 public**。
  私有仓库的 release 资产匿名拉不到，客户端检查更新会直接失败。
- 更新源写在 `mygo.json`：

```json
"updates": {
  "publicKey": "wNxnpVQgF+9p0lYvXzcVAR5FFVg0c1kqumK+DX1eyxI=",
  "github": "lark203/bestuav-mygo",
  "tagPrefix": "v"
}
```

### 2.2 签名密钥：生成、保存与配置

更新包靠 **ed25519 签名**，防止有人伪造 release 骗客户端下载恶意包。密钥对**只生成一次**，整个产品生命周期共用。

#### 生成

```bash
go tool mygo keygen
```

默认写到用户配置目录（Windows 即 `%APPDATA%\mygo\update-keys\`），产出两个文件：

| 文件 | 内容 | 去向 |
| --- | --- | --- |
| `mygo-update.pub` | 公钥，base64 单行（44 字符） | 填进 `mygo.json` 的 `updates.publicKey`，**随代码提交** |
| `mygo-update.key` | 私钥，base64 单行（88 字符） | **绝不进仓库**：密码管理器 + CI Secret 各存一份 |

命令结束会直接打印一段可粘贴进 `mygo.json` 的片段：

```json
"updates": { "publicKey": "…", "github": "owner/name" }
```

其它选项：`-o <dir>` 指定输出目录；`-force` 覆盖已存在的密钥（**极度谨慎**，见本节末尾）。

本仓库当前公钥（已在 `mygo.json` 中）：

```
wNxnpVQgF+9p0lYvXzcVAR5FFVg0c1kqumK+DX1eyxI=
```

#### 签名在构建期自动完成

`go tool mygo build` 时：

1. 从 `MYGO_UPDATER_PRIVATE_KEY`（或 `mygo.json` 的 `updates.privateKey`，值是**文件路径**）读私钥；
2. 打包 `*-<target>.tar.gz` 的同时**流式计算归档的 SHA-256**；
3. 用私钥对该摘要做 ed25519 签名，base64 后写进 `update-<target>.json`：

```json
{
  "version": "0.1.1",
  "date": "2026-10-29T15:51:17Z",
  "url": "https://github.com/lark203/bestuav-mygo/releases/download/v0.1.1/quantum-0.1.1-windows-amd64.tar.gz",
  "size": 9216596,
  "signature": "Ho2Hh3N6JFXRXxzDZ9163XltCwy3ShX59RAf37B7oGKJsAc/LYaQFEBUVtPd941EdDUmmT9LPnHYFgNwAtVeDA=="
}
```

**没有私钥的后果**（必须知道）：`linux-*` 仍会产出**未签名**的 tar.gz（供 `install.sh` 本地安装），但 **windows / macos 根本不产出归档**，只打印 `Not signing an update: set MYGO_UPDATER_PRIVATE_KEY or updates.privateKey`。这就是 CI 里那道 `Check update signing key` 存在的理由——它会提前失败，而不是让你的产物"看起来正常、实际无法更新"。

#### 客户端如何校验

`Updater` 下载归档 → 计算 SHA-256 → 用**编译进二进制的公钥**（`-X packageUpdateKey`）执行 `ed25519.Verify` → 通过才解包替换，失败直接报错中止。私钥只在构建期出现，**从不进入客户端**。

#### 配置到 CI

仓库 **Settings → Secrets and variables → Actions → New repository secret**

| Name | Value |
| --- | --- |
| `MYGO_UPDATER_PRIVATE_KEY` | `%APPDATA%\mygo\update-keys\mygo-update.key` 全文（base64 单行，88 字符） |

本机复制到剪贴板（去掉末尾换行）：

```powershell
(Get-Content "$env:APPDATA\mygo\update-keys\mygo-update.key" -Raw).Trim() | Set-Clipboard
```

- 该私钥对应 `mygo.json` 里的 `updates.publicKey`；
- **必须另存一份到密码管理器**。换私钥 = 已安装的旧版本再也验证不了新包，只能让用户手动重装；
- 缺这个 secret 时，CI 的 `Check update signing key` 会直接失败——这是故意的，防止产出"装了却更新不了"的包。

### 2.3 开启 Workflow 写权限

**Settings → Actions → General → Workflow permissions → Read and write → Save**

默认是只读，`GITHUB_TOKEN` 无法创建/上传 release，`gh release create` 会返回 403。

### 2.4 配置两个 remote

```bash
# GitHub：代码镜像 + 更新源（push tag 会触发 CI）
git remote add github https://github.com/lark203/bestuav-mygo.git

# AtomGit：主仓库 / 代码备份（不触发任何 CI）
git remote add atomgit https://atomgit.com/bestuav/bestuav-mygo.git
```

检查：

```bash
git remote -v
```

> - `git remote add` **只需执行一次**；已存在会报 `remote ... already exists`，改地址用 `git remote set-url <name> <url>`。
> - 本仓库当前的 `origin` 已指向 GitHub；下文统一用显式的 `github` / `atomgit` 两个名字，避免"到底推了哪个"的歧义。

### 2.5 让本机 git 走 Clash（WorkBuddy 环境专属坑）

如果 `git push` 报下面两种错之一：

```
CONNECT tunnel failed, response 502
schannel: server closed abruptly (missing close_notify)
```

执行：

```bash
git config --global http.https://github.com.proxy http://127.0.0.1:7890
```

原因：WorkBuddy 工具进程会注入 `HTTP_PROXY=127.0.0.1:5437`（`sandbox-cli.exe` 的沙箱代理），它拦 `github.com` 但放行 `api.github.com`；而本机 Clash 监听 `7890`。该配置**只影响 github.com**，AtomGit / Gitee / 内网仓库不受影响。

回滚：`git config --global --unset http.https://github.com.proxy`

---

## 三、每次发版（照抄即可）

### 3.1 改版本号

`mygo.json` 的 `version` 与 tag 必须严格对应：`0.1.2` ↔ `v0.1.2`。
CI 的 prepare 作业会校验，不一致**直接失败**（防止把错版本号发给用户）。

### 3.2 提交 + 打 tag + 推两个仓库

```bash
cd D:/workbuddy/bestuav-mygo

# 1) 改 mygo.json 的 version（例如 0.1.1 → 0.1.2），提交
git add mygo.json
git commit -m "release: v0.1.2"

# 2) main 推到两个仓库
git push github main
git push atomgit main

# 3) 打 tag —— 只打一次！两个 remote 共用同一个本地 tag
git tag v0.1.2

# 4) tag 推两个仓库：GitHub 触发 CI，AtomGit 仅备份
git push github v0.1.2
git push atomgit v0.1.2
```

> ⚠️ **不要把 `git tag v0.1.2` 写两遍。**
> tag 是本地对象，只需 `git push` 两次；重复执行 `git tag` 会报 `tag already exists`。
> 顺序建议：**先 GitHub 再 AtomGit**——早一点触发 CI，构建要 8 分钟。

### 3.3 等 CI

<https://github.com/lark203/bestuav-mygo/actions>

四个作业**串行**执行，约 8 分钟：

```
Prepare draft release → Build windows/amd64 → Build darwin/universal → Build linux/amd64
```

串行是刻意的（`max-parallel: 1`）：三个平台都会写 `SHA256SUMS`，而 mygo 的合并逻辑是「下载已有 → 追加本平台条目 → 重传」，并发会互相覆盖、丢行。

### 3.4 人工发布（关键一步，别忘）

CI 完成后产物只是 draft，**客户端此时看不到**。三平台全绿后：

<https://github.com/lark203/bestuav-mygo/releases> → 找到该 draft → **Edit**（可顺手补发布说明）→ 取消勾选 Draft → **Publish release**

> - 本机没装 `gh` CLI，所以 `gh release edit v0.1.2 --draft=false --latest` 用不了，直接网页点。
> - 发布说明会显示在客户端「发现新版本」弹窗里。`--generate-notes` 首版只生成一行 `**Full Changelog**: ...`，建议手动写几条。

### 3.5 验证更新闭环

1. 装/解压**上一版**（例如 0.1.1）；
2. 打开 → 进「设置」→ 标题栏右侧出现蓝色**「更新」胶囊**；
3. 点击 → 弹窗显示新版本号与说明 → 「立即更新」→ 自动下载、验签、替换、重启为新版。

---

## 四、产物说明（release 里的 12 个资产）

| 资产 | 用途 |
| --- | --- |
| `quantum-setup-<ver>-amd64.exe` | Windows NSIS 安装包；默认装到 `%LOCALAPPDATA%\Programs`，**免管理员** |
| `quantum-<ver>.dmg` | macOS 磁盘映像（内含 `量子科学.app`，universal 二进制） |
| `quantum_<ver>_amd64.deb` | Debian / Ubuntu 包 |
| `quantum-<ver>-<target>.tar.gz` | **通用归档**：Windows 免安装、Linux 免安装、macOS 手动安装 |
| `update-<target>.json` | ed25519 **签名清单**，客户端更新时读它（`version` / `url` / `size` / `signature`） |
| `SHA256SUMS` | 校验和 |
| `install.sh` / `install.ps1` | 安装脚本 |

### 4.1 为什么 release 里没有 `量子科学.exe`？（已实测，**不需要改代码**）

两个独立原因：

**① mygo 的 `-upload` 白名单不含裸 exe。**
`cmd/mygo/publish.go` 的 `uploads()` 只收：`*-setup-*.exe`、`*.dmg`、`*.deb`、`*.tar.gz`、`*.delta`、`update-*.json`、`SHA256SUMS`、`install.sh`、`install.ps1`。裸 exe 不在其中，所以永远不会被上传。

**② GitHub Releases 不接受中文资产名**（2026-10-30 实测）：

| 上传名 | 结果 |
| --- | --- |
| `a.exe` | 正常，资产名 `a.exe` |
| `中文测试.exe` | 返回 200 **"成功"**，但资产名被静默改成 **`default.exe`** |

也就是说：不是报错，而是**文件名主体被丢弃、只留扩展名**。真把 `量子科学.exe` 传上去，得到的是 `default.exe`，而且多个中文名会互相覆盖。

> 附带的坑：上传资产必须打 `https://uploads.github.com/...`，打 `api.github.com/.../releases/{id}/assets` 会返回 404。

**结论**：中文 exe 通过 **tar.gz 分发**，归档**内部**的文件名不受 GitHub 限制。实测 `quantum-0.1.1-windows-amd64.tar.gz` 内容：

```
icon.png         85 KB
量子科学.exe     20.3 MB
```

所以 Windows 免安装用户解压即可；且**必须连同 `icon.png` 一起解压**（exe 从同目录读资源，单独拷 exe 会缺图标）。

> 如果一定要在 release 页面直接给"绿包"，正确做法是打一个**英文名**的 zip（如 `quantum-<ver>-windows-portable.zip`，内含中文 exe + icon.png），在 workflow 里 `Compress-Archive` 后用 `gh release upload` 补传。
> 当前**未启用**：tar.gz 已等价，多一份反而容易让人下错。

---

## 五、Windows 免安装（不装 setup.exe）

**可以，而且免安装同样支持自动更新。**

依据（mygo 源码 `updater.go`）：`installTarget()` 在非 macOS 上直接返回 `filepath.Dir(当前 exe)`——**不看注册表、不判断是否由安装器安装**；`Enabled()` 只额外要求该目录**可写**。

```powershell
# 1. 建目录（必须可写，不要放 C:\Program Files）
mkdir D:\apps\quantum

# 2. 解压（Win10+ 自带 tar；右键用 7-Zip 解压也行）
tar -xzf quantum-0.1.1-windows-amd64.tar.gz -C D:\apps\quantum

# 3. 双击运行
D:\apps\quantum\量子科学.exe
```

两种方式对比：

| | `setup.exe` 安装版 | `tar.gz` 免安装 |
| --- | --- | --- |
| 开始菜单 / 桌面快捷方式 | 有 | 无（自己建） |
| 卸载 | 控制面板卸载项 | 直接删目录 |
| **自动更新** | ✅ | ✅（要求所在目录可写） |
| 默认位置 | `%LOCALAPPDATA%\Programs\量子科学` | 自己定 |
| 放到 `C:\Program Files` | 不适用 | ⚠️ 更新会失败（不可写），设置页显示"未启用自动更新" |

---

## 六、客户端是怎么更新的（用户视角）

1. 用户打开软件 → 进「**设置**」页 → `onEnter` 钩子在后台线程发起检查（不卡界面）；
2. **标题栏右侧**的更新控件随状态变化：
   - 无更新 / 检查中 / 已禁用 / 出错 / 已忽略 → 一个下载图标按钮（图标点击 = 重新检查，检查中显示呼吸点）；
   - 有更新 → 变成蓝色**「更新」胶囊**（下载图标 + 文字）；
   - 安装中 → 显示「更新中」并禁用。
3. 点胶囊 → 弹窗（`updateDialog`）：新版本号、发布说明、发布时间、当前版本，按钮「忽略」/「立即更新」；
4. 点「立即更新」→ `mygo.Updater` 下载 tar.gz → **ed25519 验签** → 就地替换 → `App.Relaunch()` 重启到新版本；
   安装期间页面「更新」区显示进度条。
5. 点「忽略」只关掉本次提示；下次进设置会重新检查并再次提示。

限制：

- 检查更新走**匿名** GitHub API，**单 IP 60 次/小时**。正常使用足够；用户量上来后可改自托管（`updates.url`），但需自行上传签名归档，且 `mygo build -upload` 不再适用。
- **只升不降**：Updater 只在新版本更高时提示，降级必须手动装旧包。

---

## 七、故障排查

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| CI 失败于 `Check update signing key` | secret 缺失 / 为空 | 重新配置 `MYGO_UPDATER_PRIVATE_KEY`（见 2.2） |
| `gh release create/upload` 返回 403 | Workflow permissions 仍是只读 | 改成 Read and write（见 2.3） |
| prepare 失败：`tag ... 与 version ... 不匹配` | tag 与 `mygo.json` 版本不一致 | 改 `version` 提交后重新打 tag |
| `git push` 报 502 / `schannel ... abruptly` | WorkBuddy 注入的 5437 代理拦了 github.com | 见 2.5，配 per-URL 代理走 7890 |
| 客户端「检查失败」 | release 还是 draft / 仓库是 private / API 限流 | Publish release；确认仓库 public；等一小时 |
| 客户端显示「已是最新」但明明发了新版 | tag 不是 `v*`，或 release 勾了 pre-release | 改 tag 或取消 prerelease |
| 想改发布说明 / 补文件 | — | 网页 Edit 后重新 Publish；资产可单独删/传 |
| 误删了 release，想重建某版本 | — | Actions 里找到那次 run → **Re-run all jobs**（prepare 会重建 draft）；或删 tag 重推：`git push github :refs/tags/v0.1.2` 后重新打 |
| 免安装版点更新没反应 / 提示未启用 | exe 所在目录不可写（如 Program Files） | 换到用户可写目录，如 `D:\apps\quantum` |
| 免安装版图标缺失 | 只拷了 exe，没带 `icon.png` | 把 `icon.png` 与 exe 放同一目录 |

---

## 八、回滚与撤销

- **不想让客户端更新到某版本**：把该 release 改回 draft（Edit → 勾 Draft → Update），或直接删 release；
- **撤回打错的 tag**：

```bash
git tag -d v0.1.2
git push github :refs/tags/v0.1.2
git push atomgit :refs/tags/v0.1.2
```

- **已安装的客户端不会自动回退**（Updater 只升不降），降级必须让用户手动安装旧包。

---

## 九、关键约束速查

| 约束 | 说明 |
| --- | --- |
| tag 格式 | 必须 `v` + `mygo.json.version`，prepare 会校验 |
| tag 推哪里 | **必须 GitHub**，AtomGit 不触发 CI |
| 平台顺序 | 串行 `max-parallel: 1`，避免 `SHA256SUMS` 互相覆盖 |
| draft 可见性 | 对客户端不可见，**必须手动 Publish** |
| 仓库可见性 | 必须 **public**，否则 release 资产匿名拉不到 |
| 资产文件名 | **只能 ASCII**；中文名会被改成 `default.exe` |
| 更新源 | 编译期写死，换仓库/换公钥都必须重新发版 |
| 签名私钥 | 只在构建期用（签归档的 SHA-256）；缺失则 windows/macos 不产归档；**丢失 = 已装旧版永久无法更新** |
| 发布说明 | 有 `CHANGELOG.md` 时必须含 `## <version>` 段，否则构建报错；无该文件则 notes 为空 |
| API 限流 | 客户端匿名检查，单 IP 60 次/小时 |
| cgo | CI 显式 `CGO_ENABLED=0`；本地跨平台构建前也要 `unset CGO_ENABLED` |
