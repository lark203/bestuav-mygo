package view

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/scaffold"
	"bestuav-mygo/internal/ui/kit"
)

// npGroup 向导里的一个分组开关。
type npGroup struct {
	title string
	on    bool
}

// npEvent 后台生成过程向 UI 线程投递的事件（channel 传值，避免数据竞争）。
type npEvent struct {
	prog *scaffold.Progress
	res  *scaffold.Result
	err  error
}

// newProjectState 新建项目页的全部状态。
type newProjectState struct {
	project string
	brand   string
	// genDir 生成目录（父目录）。最终项目落在 genDir/<项目名>，项目名不再手填进路径。
	genDir string
	repo   string
	ident  string
	// iconPath 用户挑选的项目图标（PNG）。为空时沿用模板自带的 resources/icon.png。
	iconPath string
	// identTouched 用户手动改过应用标识后，就不再跟着项目名自动推导。
	identTouched bool
	// lastDerived 上一次自动推导出的标识，用来识别「用户是否改过」。
	lastDerived string

	groups []npGroup

	running  bool
	step     string
	done     int
	total    int
	finished bool
	errText  string
	showKey  bool
	res      scaffold.Result

	// srcDir 模板来源；为空时由 scaffold 回退到 exe 内置快照。
	srcDir string
	// ch 后台事件通道；仅在 running 期间非空。
	ch chan npEvent
}

// secretsUpdaterKey 更新签名私钥在 GitHub Actions 里的 Secret 名称。
//
// 与 .github/workflows/release.yml 里引用的名字必须完全一致；集中成常量是为了
// 让「改名字」只需要改一处，而不是在文案与按钮里各找一遍。
const secretsUpdaterKey = "MYGO_UPDATER_PRIVATE_KEY"

// npState 向导的页面状态。
//
// 为什么用包级变量而不是 App 字段：这个页面（连同 internal/scaffold）会在生成新
// 项目时被整体剔除出产物。状态若挂在 App 结构体上，删掉本文件就会留下
// `undefined: newProjectState` 的编译错误——生成器必须能把自己干净地摘掉。
// 应用是单窗口的，本页也只有一份，包级单例不会产生多实例互相干扰的问题。
var npState newProjectState

// newProjectPage 「新建项目」向导：以本工程为基调生成一个新项目。
//
// 页面本身不做任何文件操作，全部逻辑在 internal/scaffold 里（可单测）。这里只负责
// 收集参数、显示进度、把结果（尤其是更新私钥）展示给用户。
func (a *App) newProjectPage(c *ui.Context, t *ui.Theme) {
	np := a.ensureNewProjectState()
	a.drainNewProjectEvents(np)
	// 生成期间持续请求帧：进度条与步骤文案要靠逐帧读取 channel 才能动起来，
	// 而这段时间页面本身没有动画，不主动请求就一帧都不会重绘。
	if np.running {
		a.requestFrame()
	}

	ui.Column(c).Gap(kit.PageGap).Children(func() {
		ui.Text(c, "以本工程为基调，生成一个可以直接开工、能编译、能自动更新的新项目。").
			TextColor(t.TextMuted)

		npBasicSection(c, t, a, np)
		npMenuSection(c, t, np)
		npActionSection(c, t, a, np)
		if np.errText != "" {
			npErrorSection(c, t, np)
		}
		if np.finished {
			npResultSection(c, t, a, np)
		}
	})
}

// ensureNewProjectState 惰性初始化页面状态（首次进入时把分组列表从注册表读出来）。
func (a *App) ensureNewProjectState() *newProjectState {
	np := &npState
	if len(np.groups) > 0 {
		return np
	}
	np.groups = nil
	for _, g := range navRegistry() {
		if g.title == "" {
			continue // 首页不参与勾选，永远保留
		}
		np.groups = append(np.groups, npGroup{title: g.title})
	}
	np.srcDir = detectTemplateSource()
	return np
}

// drainNewProjectEvents 把后台投递的事件应用到 UI 状态上（每帧最多处理若干条）。
func (a *App) drainNewProjectEvents(np *newProjectState) {
	if np.ch == nil {
		return
	}
	for {
		select {
		case ev := <-np.ch:
			switch {
			case ev.prog != nil:
				np.step = ev.prog.Step
				np.done = ev.prog.Done
				np.total = ev.prog.Total
			case ev.err != nil:
				np.running = false
				np.errText = ev.err.Error()
				np.ch = nil
			case ev.res != nil:
				np.res = *ev.res
				np.running = false
				np.finished = true
				np.ch = nil
			}
		default:
			return
		}
	}
}

// npBasicSection 基本信息表单。
func npBasicSection(c *ui.Context, t *ui.Theme, a *App, np *newProjectState) {
	kit.Section(c, t, "基本信息", func() {
		npTextRow(c, t, "项目名", &np.project, "bestuav-foo（同时作目录名与 Go module 路径）")
		npTextRow(c, t, "品牌名称", &np.brand, "窗口标题与安装包名，例如「云鹰调度」")

		// 生成目录：只填父目录，项目名自动追加在后面（见下方预览），避免用户
		// 两边都写项目名、拼出 a/b/demo-app/demo-app 这种路径。
		ui.Row(c).Gap(kit.RowGap).AlignItems(ui.Center).Margin(0, 0, 6, 0).Children(func() {
			ui.Text(c, "生成目录").Width(kit.SettingsLabelWidth).Shrink(0).SingleLine()
			ui.TextInput(c, &np.genDir).Width(300)
			if ui.Button(c, "选择…").Clicked() {
				pickDirectory(a, np.genDir, func(dir string) { np.genDir = dir })
			}
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "将生成到").Width(kit.SettingsLabelWidth).Shrink(0).SingleLine().
				FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			ui.Text(c, npFinalPath(np)).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		})

		npTextRow(c, t, "GitHub 仓库", &np.repo, "owner/repo，例如 lark203/bestuav-foo")

		// 项目图标：只接受 PNG。go:embed 与 mygo 的打包链路都按 resources/icon.png 取图，
		// 换成别的扩展名会在打包时直接失效（而且不报错）。
		ui.Row(c).Gap(kit.RowGap).AlignItems(ui.Center).Margin(0, 0, 6, 0).Children(func() {
			ui.Text(c, "项目图标").Width(kit.SettingsLabelWidth).Shrink(0).SingleLine()
			ui.TextInput(c, &np.iconPath).Width(300)
			if ui.Button(c, "选择…").Clicked() {
				pickIconFile(a, np.iconPath, func(p string) { np.iconPath = p })
			}
			ui.Text(c, "仅 PNG，建议 1024×1024；留空用默认图标").
				TextColor(t.TextMuted).FontSize(t.FontSize-2).Margin(8, 0, 0, 0)
		})

		// 应用标识：默认跟着项目名推导；一旦用户改过就地固定，不再被覆盖。
		derived := deriveIdentifier(np.project)
		if !np.identTouched {
			if np.ident != "" && np.ident != np.lastDerived {
				np.identTouched = true
			} else {
				np.ident = derived
				np.lastDerived = derived
			}
		}
		ui.Row(c).Gap(kit.RowGap).AlignItems(ui.Center).Margin(0, 0, 6, 0).Children(func() {
			ui.Text(c, "应用标识").Width(kit.SettingsLabelWidth).Shrink(0).SingleLine()
			ui.TextInput(c, &np.ident).Width(260)
			hint := "按项目名自动推导"
			if np.identTouched {
				hint = "已手动指定"
			}
			ui.Text(c, hint).TextColor(t.TextMuted).FontSize(t.FontSize-2).Margin(8, 0, 0, 0)
		})
	})
}

// npMenuSection 分组勾选：勾上的分组会被保留，未勾的整组从注册表移除。
func npMenuSection(c *ui.Context, t *ui.Theme, np *newProjectState) {
	kit.Section(c, t, "保留哪些菜单", func() {
		ui.Text(c, "「首页（欢迎页）」与「设置」始终保留，不在此列。").
			TextColor(t.TextMuted).FontSize(t.FontSize - 2)
		ui.Text(c, "不勾选的分组会从菜单注册表移除；只被该分组使用的页面文件也会一并删除"+
			"（仍被其它代码引用的文件会保留，以保证新项目一定能编译）。").
			TextColor(t.TextMuted).FontSize(t.FontSize - 2)
		for i := range np.groups {
			g := &np.groups[i]
			ui.Row(c).Gap(10).AlignItems(ui.Center).Margin(6, 0, 0, 0).Children(func() {
				ui.Switch(c, &g.on).Shrink(0)
				ui.Text(c, g.title)
			})
		}
	})
}

// npActionSection 生成按钮与进度。
func npActionSection(c *ui.Context, t *ui.Theme, a *App, np *newProjectState) {
	kit.Section(c, t, "生成", func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			if np.running {
				ui.Text(c, "正在生成…").Bold()
			} else if ui.Button(c, "开始生成").Clicked() {
				startNewProject(a, np)
			}
			if np.running && np.total > 0 {
				ui.Progress(c, float64(np.done)/float64(np.total)).Width(240).Height(6)
				ui.Text(c, fmt.Sprintf("%s（%d/%d）", np.step, np.done, np.total)).
					TextColor(t.TextMuted).FontSize(t.FontSize - 2)
			}
		})
		ui.Text(c, "生成过程会复制工程、改写模块路径、裁剪菜单、生成更新密钥、初始化 git，"+
			"最后跑一次编译自检，通常需要一到两分钟。").
			TextColor(t.TextMuted).FontSize(t.FontSize-2).Margin(6, 0, 0, 0)
	})
}

// npErrorSection 失败提示。
func npErrorSection(c *ui.Context, t *ui.Theme, np *newProjectState) {
	kit.Section(c, t, "生成失败", func() {
		ui.Text(c, np.errText).TextColor(ui.RGBA(220, 80, 80, 1))
	})
}

// npResultSection 结果展示：目录、密钥、下一步命令、统计。
func npResultSection(c *ui.Context, t *ui.Theme, a *App, np *newProjectState) {
	res := np.res
	kit.Section(c, t, "生成完成", func() {
		kit.KV(c, t, "项目目录", res.Dir)
		kit.KV(c, t, "模块路径", res.ModulePath)
		kit.KV(c, t, "最终文件数", fmt.Sprintf("%d 个（复制 %d，裁剪后）", res.FinalFileCount, res.FileCount))
		kit.KV(c, t, "删除页面", fmt.Sprintf("%d 个", len(res.RemovedFiles)))
		if len(res.RemovedPkgs) > 0 {
			kit.KV(c, t, "删除演示包", fmt.Sprintf("%d 个（%s）",
				len(res.RemovedPkgs), strings.Join(baseNames(res.RemovedPkgs), "、")))
		}
		if len(res.KeptFiles) > 0 {
			kit.KV(c, t, "保留页面", fmt.Sprintf("%d 个（仍被其它代码引用）", len(res.KeptFiles)))
		}
		if res.GitRemote != "" {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "远程地址").Width(kit.SettingsLabelWidth).Shrink(0).SingleLine().
					TextColor(t.TextMuted)
				ui.Text(c, res.GitRemote+"（已配置，未推送）").
					FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1)
				if ui.Button(c, "复制仓库地址").Clicked() {
					a.copyText(res.GitRemote, "仓库地址已复制")
				}
			})
		}
		kit.KV(c, t, "编译自检", map[bool]string{true: "通过", false: "跳过"}[res.BuildOK])

		// ---- 更新公钥：已经写进 mygo.json，这里只是让人看得见 ----
		ui.Text(c, "更新公钥（已写入 mygo.json 的 updates.publicKey）").Bold().Margin(10, 0, 4, 0)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, res.PublicKey).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			if ui.Button(c, "复制公钥").Clicked() {
				a.copyText(res.PublicKey, "公钥已复制")
			}
		})

		// ---- 更新私钥：只在这一页显示一次 ----
		ui.Text(c, "更新私钥（**只显示这一次**，请立刻配到 GitHub Secret）").
			Bold().Margin(10, 0, 4, 0)
		ui.Text(c, "GitHub 仓库 → Settings → Secrets and variables → Actions → New repository secret。"+
			"Secret 名称与值分别用下面两个按钮复制，避免手打错一个字符就白等一次 CI。").
			TextColor(t.TextMuted).FontSize(t.FontSize - 2)

		// Secret 名称单独一行并可复制：CI 里写死的变量名，打错就是「secret 为空」
		// 这种最难查的失败。
		ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(6, 0, 0, 0).Children(func() {
			ui.Text(c, "Secret 名称").Width(kit.SettingsLabelWidth).Shrink(0).SingleLine().
				TextColor(t.TextMuted)
			ui.Text(c, secretsUpdaterKey).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1)
			if ui.Button(c, "复制名称").Clicked() {
				a.copyText(secretsUpdaterKey, "Secret 名称已复制")
			}
		})

		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Secret 值").Width(kit.SettingsLabelWidth).Shrink(0).SingleLine().
				TextColor(t.TextMuted)
			shown := "••••••••••••••••••••（已隐藏）"
			if np.showKey {
				shown = res.PrivateKey
			}
			ui.Text(c, shown).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			if ui.Button(c, "显示/隐藏").Clicked() {
				np.showKey = !np.showKey
			}
			if ui.Button(c, "复制私钥").Clicked() {
				a.copyText(res.PrivateKey, "私钥已复制")
			}
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "私钥备份位置").Width(kit.SettingsLabelWidth).Shrink(0).SingleLine().
				TextColor(t.TextMuted)
			ui.Text(c, res.PrivateKeyPath).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1)
			if ui.Button(c, "打开目录").Clicked() {
				openInFileManager(filepath.Dir(res.PrivateKeyPath))
			}
		})

		// ---- 下一步 ----
		ui.Text(c, "下一步").Bold().Margin(10, 0, 4, 0)
		for _, line := range []string{
			"1. 把私钥配成 GitHub Secret（名称 MYGO_UPDATER_PRIVATE_KEY）",
			"2. 进入项目目录执行：git push -u origin main",
			"3. 按项目里的 docs/首次发布.md 打第一个 tag 并发版",
		} {
			ui.Text(c, line).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		}
		if ui.Button(c, "复制推送命令").Clicked() {
			a.copyText(scaffold.GitPushCommand(res.Dir), "命令已复制")
		}

		for _, w := range res.Warnings {
			ui.Text(c, "· "+w).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		}
	})
}

// npTextRow 一行「标签 + 输入框 + 提示」。
func npTextRow(c *ui.Context, t *ui.Theme, label string, bound *string, hint string) {
	ui.Row(c).Gap(kit.RowGap).AlignItems(ui.Center).Margin(0, 0, 6, 0).Children(func() {
		ui.Text(c, label).Width(kit.SettingsLabelWidth).Shrink(0).SingleLine()
		ui.TextInput(c, bound).Width(260)
		ui.Text(c, hint).TextColor(t.TextMuted).FontSize(t.FontSize-2).Margin(8, 0, 0, 0)
	})
}

// startNewProject 在后台跑生成流程。
//
// 用 channel 回传事件而不是直接改共享字段：生成跑在独立 goroutine 上，而渲染在
// 主线程按帧读取状态，直接写字段是数据竞争。
func startNewProject(a *App, np *newProjectState) {
	np.running = true
	np.finished = false
	np.errText = ""
	np.showKey = false
	np.done, np.total = 0, 0
	np.step = "准备中"

	if strings.TrimSpace(np.genDir) == "" {
		np.genDir = defaultGenDir()
	}

	opts := scaffold.Options{
		SourceDir:   np.srcDir,
		TargetDir:   npFinalPath(np),
		IconPath:    strings.TrimSpace(np.iconPath),
		ProjectName: np.project,
		BrandName:   np.brand,
		Identifier:  np.ident,
		GitHubRepo:  np.repo,
		KeyDir:      "",
	}
	for _, g := range np.groups {
		if g.on {
			opts.KeepGroups = append(opts.KeepGroups, g.title)
		}
	}

	ch := make(chan npEvent, 32)
	np.ch = ch
	go func() {
		res, err := scaffold.Generate(opts, func(p scaffold.Progress) {
			prog := p
			ch <- npEvent{prog: &prog}
		})
		if err != nil {
			ch <- npEvent{err: err}
			return
		}
		ch <- npEvent{res: &res}
	}()
	// 生成期间需要持续重绘才能看到进度。
	a.requestFrame()
}

// deriveIdentifier 由项目名推导反向域名标识：demo-app → com.demoapp。
//
// 与 scaffold 的默认推导规则保持一致，否则用户看到的值和实际写进 mygo.json 的
// 值会不一样。
func deriveIdentifier(project string) string {
	p := strings.TrimSpace(project)
	if p == "" {
		return ""
	}
	return "com." + strings.ReplaceAll(p, "-", "")
}

// detectTemplateSource 猜模板源目录：当前工作目录本身是本工程时就用它。
//
// 打包后的应用没有源码，此处返回空串，scaffold 会自动回退到内置快照。
func detectTemplateSource() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for _, name := range []string{"mygo.json", "go.mod"} {
		if _, statErr := os.Stat(filepath.Join(cwd, name)); statErr != nil {
			return ""
		}
	}
	if _, statErr := os.Stat(filepath.Join(cwd, "internal", "view", "registry.go")); statErr != nil {
		return ""
	}
	return cwd
}

// openInFileManager 在系统文件管理器里打开目录（Windows 资源管理器 / macOS 访达）。
//
// 打开失败不弹错：这是个辅助动作，失败了用户自己也能按旁边显示的路径找到。
func openInFileManager(dir string) {
	if dir == "" {
		return
	}
	_ = mygo.Shell.OpenPath(dir)
}

// npFinalPath 最终落盘路径 = 生成目录 + 项目名。
//
// 项目名为空时只显示生成目录，让用户看到「还差一个项目名」，而不是拼出
// 一个以分隔符结尾的怪路径。
func npFinalPath(np *newProjectState) string {
	dir := strings.TrimSpace(np.genDir)
	project := strings.TrimSpace(np.project)
	if dir == "" {
		return "（请先填写生成目录）"
	}
	if project == "" {
		return dir
	}
	return filepath.Join(dir, project)
}

// defaultGenDir 生成目录默认取「当前工作目录；若当前目录就是本仓库，取其上级」。
func defaultGenDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	if detectTemplateSource() != "" {
		return filepath.Dir(cwd)
	}
	return cwd
}

// pickDirectory 打开系统目录选择框，选中后把结果交给回调（在主线程执行）。
//
// 对话框会阻塞调用方，故放到 goroutine；回调里必须回主线程改状态并请求重绘。
func pickDirectory(a *App, current string, onPick func(string)) {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:       "选择生成目录",
			ButtonLabel: "选择",
			Directory:   true,
			DefaultPath: current,
		})
		if err != nil || len(paths) == 0 {
			return
		}
		chosen := paths[0]
		mygo.RunOnMain(func() {
			onPick(chosen)
			a.requestFrame()
		})
	}()
}

// pickIconFile 打开图片选择框（只允许 PNG），选中后交给回调。
//
// 文件类型必须在对话框层面就限制住：go:embed 与 mygo 打包都只认
// resources/icon.png，允许选 .jpg/.ico 只会让用户以为换了图标、实际没换。
func pickIconFile(a *App, current string, onPick func(string)) {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:       "选择项目图标",
			ButtonLabel: "使用该图标",
			DefaultPath: current,
			Filters: []mygo.FileFilter{{
				Name:       "PNG 图片",
				Extensions: []string{"png"},
			}},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		chosen := paths[0]
		mygo.RunOnMain(func() {
			onPick(chosen)
			a.requestFrame()
		})
	}()
}

// baseNames 取相对路径的最后一段，用于「删除演示包（blackhole、piano…）」这类摘要。
func baseNames(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, filepath.Base(filepath.FromSlash(p)))
	}
	return out
}
