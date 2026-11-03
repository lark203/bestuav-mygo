package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// step 数量用于进度条（每步都会回调一次）。
const stepTotal = 9

// Generate 按 Options 生成一个新项目。
//
// 失败时会尽力清理半成品目录：这里的目录是本函数刚创建、且创建前校验为空，
// 不涉及用户已有文件。
func Generate(o Options, progress func(Progress)) (res Result, err error) {
	report := func(n int, step string) {
		if progress != nil {
			progress(Progress{Step: step, Done: n, Total: stepTotal})
		}
	}

	if err := o.Normalize(); err != nil {
		return res, err
	}
	res.Dir = o.TargetDir
	res.ModulePath = o.ProjectName

	report(0, "检查模板来源")
	tpl, origin, err := templateFS(o.SourceDir)
	if err != nil {
		return res, err
	}
	res.Warnings = append(res.Warnings, "模板来源："+origin)

	// 目录可能是「已存在但为空」，记下是否由本函数新建，供失败清理判断。
	_, statErr := os.Stat(o.TargetDir)
	created := os.IsNotExist(statErr)
	if err := os.MkdirAll(o.TargetDir, 0o755); err != nil {
		return res, fmt.Errorf("创建目标目录失败: %w", err)
	}
	defer func() {
		if err == nil || !created {
			return
		}
		if rmErr := os.RemoveAll(o.TargetDir); rmErr != nil {
			res.Warnings = append(res.Warnings,
				"生成失败后清理半成品目录未完成："+rmErr.Error())
		}
	}()

	report(1, "复制工程骨架")
	res.FileCount, err = copyTemplate(tpl, o.TargetDir)
	if err != nil {
		return res, err
	}
	if err = applyProjectIcon(o.TargetDir, o.IconPath); err != nil {
		return res, err
	}

	report(2, "重写模块路径")
	if _, err = rewriteModule(o.TargetDir, DefaultModuleName, o.ProjectName); err != nil {
		return res, err
	}

	report(3, "生成更新密钥")
	kp, err := generateKeys(o.TargetDir, o.ProjectName, o.KeyDir)
	if err != nil {
		return res, err
	}
	res.PublicKey = kp.PublicKey
	res.PrivateKey = kp.PrivateKey
	res.PrivateKeyPath = kp.KeyPath

	report(4, "改写产品标识")
	if err = rewriteAppmeta(o.TargetDir, DefaultModuleName, o.ProjectName, o.BrandName, o.Identifier); err != nil {
		return res, err
	}
	if err = rewriteMygoJSON(o.TargetDir, o.ProjectName, o.BrandName,
		o.Identifier, o.GitHubRepo, kp.PublicKey); err != nil {
		return res, err
	}
	if _, err = rewriteScripts(o.TargetDir, DefaultModuleName, o.ProjectName); err != nil {
		return res, err
	}
	if err = rewriteTextRefs(o.TargetDir, DefaultModuleName, o.ProjectName); err != nil {
		return res, err
	}
	if err = writeGitignore(o.TargetDir); err != nil {
		return res, err
	}

	report(5, "裁剪菜单注册表")
	regPath := filepath.Join(o.TargetDir, "internal", "view", "registry.go")
	cut, err := cutRegistry(regPath, o.KeepGroups, []string{"new-project"})
	if err != nil {
		return res, err
	}
	if len(cut.RemovedGroups) > 0 {
		res.Warnings = append(res.Warnings,
			"已移除菜单分组："+strings.Join(cut.RemovedGroups, "、"))
	}

	// 菜单裁掉后，快捷键动作表里会留下指向不存在页面的动作：设置页仍把它们列出来，
	// 按了却没反应。按裁剪结果删掉这些动作，并把导航键位重排成 cmd+1..N。
	removedActions, err := pruneShortcuts(o.TargetDir, cut.removedPageSet())
	if err != nil {
		return res, err
	}
	if len(removedActions) > 0 {
		res.Warnings = append(res.Warnings,
			"已移除失效的快捷键动作："+strings.Join(removedActions, "、")+
				"（其余导航键位已重排为连续的 cmd+1..N）")
	}

	report(6, "删除被移除分组的页面文件与演示包")
	if o.SkipBuild {
		// 没有编译能力时不做文件清理：删错文件而不自检，比留着多余文件糟得多。
		res.Warnings = append(res.Warnings, "已跳过编译自检与页面文件清理（保留全部页面文件）")
	} else {
		res.RemovedFiles, res.KeptFiles, err = pruneRemovedPages(o.TargetDir, cut.RemovedFuncs)
		if err != nil {
			return res, err
		}
		// 页面删完后，只服务于这些页面的演示子包多半也没人引用了：整包删掉，
		// 能显著减少新项目的文件数与依赖（同样是删完编译验证）。
		res.RemovedPkgs, err = pruneDemoPackages(o.TargetDir)
		if err != nil {
			return res, err
		}
	}
	// 视图层测试断言的是演示页面的内容，只要动过菜单分组，它们就必然失败，一并清掉。
	if len(cut.RemovedGroups) > 0 {
		if err = dropViewPackageTests(o.TargetDir); err != nil {
			return res, err
		}
		res.Warnings = append(res.Warnings,
			"已删除 internal/view 下的演示页测试（它们断言的是被移除的菜单分组）")
	}

	report(7, "生成项目文档")
	if err = writeProjectDocs(o.TargetDir, o, kp); err != nil {
		return res, err
	}

	report(8, "初始化 git 仓库")
	if !o.SkipGit {
		remote, warns, gerr := initGit(o.TargetDir, o.ProjectName, o.GitHubRepo)
		if gerr != nil {
			return res, gerr
		}
		res.GitRemote = remote
		res.Warnings = append(res.Warnings, warns...)
	}

	res.FinalFileCount = countProjectFiles(o.TargetDir)

	report(9, "编译自检")
	if !o.SkipBuild {
		if buildErr := goBuild(o.TargetDir); buildErr != nil {
			res.BuildOutput = buildErr.Error()
			return res, fmt.Errorf("生成完成但编译自检失败（工程可能仍可用，请按输出定位）: %w", buildErr)
		}
		res.BuildOK = true
	}
	return res, nil
}

// dropViewPackageTests 删除 internal/view 顶层的测试文件。
//
// 这些测试断言的是「这台演示程序有哪些菜单、每页显示什么」，任意裁剪分组都会让
// 其中几条失败。留着必然红花一片，删掉又会让新项目失去「这里原来怎么写测试」的
// 参照——因此只在**裁剪过分组**时才删，并把这一点写进生成结果提示里。
//
// 其它包的测试（kit / canvas / settings / store / logging）与菜单无关，全部保留。
func dropViewPackageTests(root string) error {
	dir := filepath.Join(root, "internal", "view")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// countProjectFiles 统计产物里的文件数（跳过 .git，那是初始化后才有的一大堆文件）。
func countProjectFiles(root string) int {
	n := 0
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}
