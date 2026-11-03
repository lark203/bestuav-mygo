package scaffold

import (
	"fmt"
	"os/exec"
	"strings"
)

// initGit 在生成目录里初始化仓库并指好远程，但**不推送**。
//
// 不自动 push 的理由：仓库是否已存在、本机凭据是否可用都不由生成器决定；一旦
// 在最后一步失败，用户看到的是一堆明明成功了却被标红的步骤。所以只把该配的配好，
// 推送命令写进 README 与手册里让用户自己执行。
func initGit(root, project, repo string) (remote string, warnings []string, err error) {
	remote = "https://github.com/" + repo + ".git"

	if _, lerr := exec.LookPath("git"); lerr != nil {
		return "", []string{"未找到 git 命令，已跳过仓库初始化"}, nil
	}

	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, rerr := cmd.CombinedOutput()
		if rerr != nil {
			return fmt.Errorf("git %s 失败: %v\n%s", strings.Join(args, " "), rerr, truncate(string(out), 800))
		}
		return nil
	}

	if err := run("init", "-b", "main"); err != nil {
		return "", nil, err
	}
	if err := run("remote", "add", "origin", remote); err != nil {
		// 已存在同名远程（重复生成到同一目录的极端情况）时不当致命错误。
		warnings = append(warnings, "添加远程 origin 失败："+err.Error())
	}
	if err := run("add", "-A"); err != nil {
		return remote, warnings, err
	}
	msg := "chore: 初始化 " + project + "（由 bestuav-mygo 脚手架生成）"
	if err := run("commit", "-m", msg); err != nil {
		// 未配置 user.name/user.email 时提交会失败：这不是生成失败，交给用户补。
		warnings = append(warnings,
			"初始提交失败（通常是 git 未配置 user.name/user.email）："+err.Error())
		return remote, warnings, nil
	}
	return remote, warnings, nil
}

// GitPushCommand 返回给 UI 展示的「推送这条命令」（目录用引号包住，空格路径也能直接粘）。
func GitPushCommand(dir string) string {
	return `cd "` + dir + `" && git push -u origin main`
}
