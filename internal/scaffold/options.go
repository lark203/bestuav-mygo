// Package scaffold 以本仓库为「基调」，生成一个可以直接开工的新项目。
//
// 设计目标：生成物必须是**能立刻 go build 通过**的完整工程，而不是一堆待手工
// 修补的模板文件。因此这里做的不是简单的字符串替换，而是：
//
//  1. 白名单复制（只带走运行所需的源码，丢掉本项目的 docs 历史与截图）；
//  2. 模块路径全量重写（Go module 名在 import 里出现近百次，漏一处就编不过）；
//  3. 用 AST 裁剪菜单注册表（按分组过滤），再按「谁还引用它」决定删哪些页面文件；
//  4. 删完跑 go build 自检，编不过就自动回滚可疑文件——宁可多留文件，也不能
//     交出一个坏工程；
//  5. 生成 ed25519 更新密钥、把公钥写进 mygo.json，私钥交给调用方去配 CI Secret；
//  6. git init + 指好远程，但不 push（推送涉及凭据与仓库是否存在，交给用户）。
//
// 本包不依赖 UI，全部逻辑可用单测覆盖。
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DefaultModuleName 模板来源仓库的 module 名（重写时用它做匹配）。
const DefaultModuleName = "bestuav-mygo"

// Options 生成一个新项目所需的全部输入。
type Options struct {
	// SourceDir 模板源目录（本仓库源码）。为空时使用编译进 exe 的快照。
	SourceDir string
	// TargetDir 目标目录（父目录 + 项目名已拼好）。必须不存在或为空目录。
	TargetDir string
	// ProjectName 项目名，同时用作目录名与 Go module 路径，例如 bestuav-foo。
	ProjectName string
	// BrandName 品牌名称（窗口标题、安装包产品名、侧栏品牌头）。
	BrandName string
	// Identifier 应用唯一标识符，例如 com.bestuav.foo。
	Identifier string
	// GitHubRepo 更新源仓库 owner/repo。
	GitHubRepo string
	// KeepGroups 保留的菜单分组标题；首页与设置永远保留，不在此列。
	KeepGroups []string

	// IconPath 自定义项目图标（PNG）。为空时沿用模板自带的 resources/icon.png。
	IconPath string
	// KeyDir 私钥落盘目录；为空时用 <用户配置目录>/mygo/update-keys/<项目名>。
	KeyDir string
	// SkipGit 为 true 时不初始化 git（单测与试跑用）。
	SkipGit bool
	// SkipBuild 为 true 时不跑 go build 自检（单测用；生产路径必须跑）。
	SkipBuild bool
}

// Progress 一次进度回调的载荷。
type Progress struct {
	Step  string // 当前步骤（人话，直接显示在向导里）
	Done  int    // 已完成步数（从 0 开始）
	Total int    // 总步数
}

// Result 生成结果。
type Result struct {
	Dir            string   // 生成的项目目录
	ModulePath     string   // 新的 Go module 路径
	FileCount      int      // 复制的文件数（裁剪前）
	FinalFileCount int      // 裁剪后的最终文件数（不含 .git）
	RemovedFiles   []string // 因不保留分组而删除的页面文件（相对路径）
	RemovedPkgs    []string // 因无人引用而整包删除的演示子包（相对路径）
	KeptFiles      []string // 因仍被引用而未删除的候选文件
	PublicKey      string   // 更新公钥（已写进 mygo.json）
	PrivateKey     string   // 更新私钥（仅回传给 UI 展示/复制，不落进仓库）
	PrivateKeyPath string   // 私钥在本机的位置
	GitRemote      string   // 已配置的 origin 地址（未初始化 git 时为空）
	BuildOK        bool     // go build 自检是否通过
	BuildOutput    string   // 自检输出（失败时是错误摘要）
	Warnings       []string // 非致命提示
}

// projectNameRe 项目名：小写字母开头、允许数字与中划线（Go module 路径与目录名双重要求）。
var projectNameRe = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// identifierRe 反向域名标识：至少两段，允许字母数字中划线。
var identifierRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*(\.[a-zA-Z0-9][a-zA-Z0-9-]*)+$`)

// repoRe owner/repo。
var repoRe = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// Normalize 补全可推导的字段并做基础校验。
//
// identifier 缺省时按 com.<项目名去中划线> 推导；项目名允许含中划线，但 Go 的
// import 路径与目录名都允许，故不额外转换。
func (o *Options) Normalize() error {
	o.ProjectName = strings.TrimSpace(o.ProjectName)
	o.BrandName = strings.TrimSpace(o.BrandName)
	o.Identifier = strings.TrimSpace(o.Identifier)
	o.GitHubRepo = strings.TrimSpace(o.GitHubRepo)
	o.SourceDir = strings.TrimSpace(o.SourceDir)
	o.TargetDir = strings.TrimSpace(o.TargetDir)

	if !projectNameRe.MatchString(o.ProjectName) {
		return fmt.Errorf("项目名 %q 不合法：只能小写字母开头，含小写字母/数字/中划线，例如 bestuav-foo", o.ProjectName)
	}
	if o.BrandName == "" {
		return fmt.Errorf("品牌名称不能为空")
	}
	if o.Identifier == "" {
		o.Identifier = "com." + strings.ReplaceAll(o.ProjectName, "-", "")
	}
	if !identifierRe.MatchString(o.Identifier) {
		return fmt.Errorf("应用标识 %q 不合法：需为反向域名形式，例如 com.example.app", o.Identifier)
	}
	if !repoRe.MatchString(o.GitHubRepo) {
		return fmt.Errorf("GitHub 仓库 %q 不合法：需为 owner/repo 形式，例如 lark203/bestuav-foo", o.GitHubRepo)
	}
	if o.TargetDir == "" {
		return fmt.Errorf("目标目录不能为空")
	}
	if o.IconPath = strings.TrimSpace(o.IconPath); o.IconPath != "" {
		if !strings.EqualFold(filepath.Ext(o.IconPath), ".png") {
			return fmt.Errorf("图标必须是 PNG：%s", o.IconPath)
		}
		if info, statErr := os.Stat(o.IconPath); statErr != nil {
			return fmt.Errorf("图标文件不存在: %w", statErr)
		} else if info.IsDir() {
			return fmt.Errorf("图标路径是目录而不是文件: %s", o.IconPath)
		}
	}
	abs, err := filepath.Abs(o.TargetDir)
	if err != nil {
		return fmt.Errorf("目标目录无效: %w", err)
	}
	o.TargetDir = abs
	if info, statErr := os.Stat(abs); statErr == nil {
		if !info.IsDir() {
			return fmt.Errorf("目标路径已存在且不是目录: %s", abs)
		}
		entries, rerr := os.ReadDir(abs)
		if rerr != nil {
			return fmt.Errorf("读取目标目录失败: %w", rerr)
		}
		if len(entries) > 0 {
			return fmt.Errorf("目标目录非空（需空目录或不存在的路径）: %s", abs)
		}
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("检查目标目录失败: %w", statErr)
	}
	return nil
}
