package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot 返回本仓库根目录（测试运行在 internal/scaffold 下）。
func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(abs, "mygo.json")); statErr != nil {
		t.Skipf("不在仓库内运行（%s 下没有 mygo.json），跳过", abs)
	}
	return abs
}

func TestNormalizeValidation(t *testing.T) {
	base := Options{
		TargetDir:   filepath.Join(t.TempDir(), "proj"),
		ProjectName: "demo-app",
		BrandName:   "演示应用",
		GitHubRepo:  "someone/demo-app",
	}
	ok := base
	if err := ok.Normalize(); err != nil {
		t.Fatalf("合法输入被拒: %v", err)
	}
	if ok.Identifier != "com.demoapp" {
		t.Errorf("identifier 自动推导 = %q，期望 com.demoapp", ok.Identifier)
	}

	bad := []struct {
		name string
		mut  func(o *Options)
	}{
		{"项目名含大写", func(o *Options) { o.ProjectName = "Demo" }},
		{"项目名含下划线", func(o *Options) { o.ProjectName = "demo_app" }},
		{"项目名为空", func(o *Options) { o.ProjectName = "" }},
		{"品牌名为空", func(o *Options) { o.BrandName = "  " }},
		{"仓库缺少 owner", func(o *Options) { o.GitHubRepo = "demo-app" }},
		{"标识不是反向域名", func(o *Options) { o.Identifier = "demoapp" }},
	}
	for _, c := range bad {
		o := base
		o.TargetDir = filepath.Join(t.TempDir(), "p2")
		c.mut(&o)
		if err := o.Normalize(); err == nil {
			t.Errorf("%s：应当报错但通过了", c.name)
		}
	}
}

// TestNormalizeRejectsNonEmptyTarget 目标目录非空时必须拒绝，避免覆盖用户文件。
func TestNormalizeRejectsNonEmptyTarget(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := Options{
		TargetDir:   dir,
		ProjectName: "demo-app",
		BrandName:   "演示",
		GitHubRepo:  "a/b",
	}
	if err := o.Normalize(); err == nil {
		t.Fatal("目标目录非空时应报错")
	}
}

// TestCutRegistry 用真实 registry.go 验证：保留「工作台」时，其它分组被整组移除，
// 且生成器自己的 new-project 菜单项也被摘掉。
func TestCutRegistry(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "internal", "view", "registry.go")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(t.TempDir(), "registry.go")
	if err := os.WriteFile(work, data, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := cutRegistry(work, []string{"工作台"}, []string{"new-project"})
	if err != nil {
		t.Fatalf("裁剪失败: %v", err)
	}
	if len(res.RemovedGroups) == 0 {
		t.Fatal("应当移除若干分组")
	}
	if _, statErr := os.Stat(work); statErr != nil {
		t.Fatal(statErr)
	}
	out, err := os.ReadFile(work)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{`title: "工作台"`, `id: "task"`, `id: "home"`, "settingsNav"} {
		if !strings.Contains(text, want) {
			t.Errorf("应保留 %q，实际文件里没有", want)
		}
	}
	for _, gone := range []string{`title: "组件演示"`, `title: "官方示例"`, `title: "功能演示"`, "comp-buttons", "official-piano"} {
		if strings.Contains(text, gone) {
			t.Errorf("应移除 %q，实际仍存在", gone)
		}
	}
	if !strings.Contains(text, `id: "task"`) {
		t.Error("工作台的异步任务页应保留")
	}
}

// TestGenerateFromSource 从本地源码生成一个项目，校验关键文件的改写结果。
//
// 跳过 git 与编译（这两步由 TestGenerateEndToEnd 在真实环境里验证）。
func TestGenerateFromSource(t *testing.T) {
	root := repoRoot(t)
	target := filepath.Join(t.TempDir(), "demo-app")

	res, err := Generate(Options{
		SourceDir:   root,
		TargetDir:   target,
		ProjectName: "demo-app",
		BrandName:   "演示应用",
		GitHubRepo:  "someone/demo-app",
		KeepGroups:  nil, // 只留首页与设置
		KeyDir:      t.TempDir(),
		SkipGit:     true,
		SkipBuild:   true,
	}, nil)
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if res.FileCount == 0 {
		t.Error("没有复制任何文件")
	}
	if res.PublicKey == "" || res.PrivateKey == "" {
		t.Error("应生成更新密钥对")
	}

	// go.mod 的 module 行。
	gomod := mustRead(t, filepath.Join(target, "go.mod"))
	if !strings.Contains(gomod, "module demo-app") {
		t.Errorf("go.mod 未改成新模块名：%s", firstLine(gomod))
	}

	// appmeta 三个常量。
	appmeta := mustRead(t, filepath.Join(target, "internal", "appmeta", "appmeta.go"))
	for _, want := range []string{
		`const Name = "demo-app"`,
		`const Title = "演示应用"`,
		`const Identifier = "com.demoapp"`,
	} {
		if !strings.Contains(appmeta, want) {
			t.Errorf("appmeta.go 缺少 %q", want)
		}
	}

	// mygo.json 全字段。
	data, err := os.ReadFile(filepath.Join(target, "mygo.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Name       string `json:"name"`
		Slug       string `json:"slug"`
		Identifier string `json:"identifier"`
		Version    string `json:"version"`
		Updates    struct {
			GitHub    string `json:"github"`
			PublicKey string `json:"publicKey"`
		} `json:"updates"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("mygo.json 不合法: %v", err)
	}
	if cfg.Name != "演示应用" || cfg.Slug != "demo-app" || cfg.Version != "0.1.0" {
		t.Errorf("mygo.json 基本字段不对: %+v", cfg)
	}
	if cfg.Updates.GitHub != "someone/demo-app" || cfg.Updates.PublicKey != res.PublicKey {
		t.Errorf("mygo.json 更新字段不对: %+v", cfg.Updates)
	}

	// 生成器自身不应出现在产物里。
	for _, gone := range []string{
		"internal/scaffold",
		"internal/view/page_newproject.go",
		"docs/screenshots",
		"CHANGELOG.md.bak",
	} {
		if _, statErr := os.Stat(filepath.Join(target, filepath.FromSlash(gone))); statErr == nil {
			t.Errorf("产物里不应包含 %s", gone)
		}
	}

	// 新生成的文档。
	for _, want := range []string{"README.md", "CHANGELOG.md", "docs/首次发布.md"} {
		if _, statErr := os.Stat(filepath.Join(target, filepath.FromSlash(want))); statErr != nil {
			t.Errorf("缺少生成文档 %s", want)
		}
	}
	manual := mustRead(t, filepath.Join(target, "docs", "首次发布.md"))
	for _, want := range []string{"MYGO_UPDATER_PRIVATE_KEY", "someone/demo-app", "更新签名"} {
		if !strings.Contains(manual, want) {
			t.Errorf("发布手册缺少关键内容 %q", want)
		}
	}

	// 注册表只剩首页与设置。
	reg := mustRead(t, filepath.Join(target, "internal", "view", "registry.go"))
	if strings.Contains(reg, `title: "组件演示"`) {
		t.Error("未勾选的分组应从注册表移除")
	}
	if !strings.Contains(reg, `id: "home"`) {
		t.Error("首页必须保留")
	}

	// .gitignore 应排除私钥。
	gi := mustRead(t, filepath.Join(target, ".gitignore"))
	if !strings.Contains(gi, "mygo-update.key") {
		t.Error(".gitignore 应排除更新私钥")
	}
}

func mustRead(t *testing.T, p string) string {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读取 %s: %v", p, err)
	}
	return string(data)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// TestGenerateEndToEnd 真实走一遍：生成 + go build 自检。
//
// 比较慢（要编译整个工程），默认在 -short 下跳过；发版前请手动跑一次
// `go test ./internal/scaffold/ -run TestGenerateEndToEnd -count=1`。
func TestGenerateEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("编译自检耗时，-short 下跳过")
	}
	root := repoRoot(t)
	target := filepath.Join(t.TempDir(), "e2e-app")

	res, err := Generate(Options{
		SourceDir:   root,
		TargetDir:   target,
		ProjectName: "e2e-app",
		BrandName:   "端到端验证",
		GitHubRepo:  "someone/e2e-app",
		KeepGroups:  []string{"工作台"},
		KeyDir:      t.TempDir(),
		SkipGit:     true,
	}, nil)
	if err != nil {
		t.Fatalf("生成失败: %v\n%s", err, res.BuildOutput)
	}
	if !res.BuildOK {
		t.Fatalf("编译自检未通过：%s", res.BuildOutput)
	}
	// 保留「工作台」时异步任务页应还在（它的文件不该被误删）。
	if _, statErr := os.Stat(filepath.Join(target, "internal", "view", "page_task.go")); statErr != nil {
		t.Errorf("保留的分组页面被误删: %v", statErr)
	}
	// 未保留的官方示例页面应被删除。
	if _, statErr := os.Stat(filepath.Join(target, "internal", "view", "page_piano.go")); statErr == nil {
		t.Error("未保留分组的页面文件应被删除")
	}
	// 只服务于测试的包应被整包删除（页面被删后它就没有引用了）。
	if _, statErr := os.Stat(filepath.Join(target, "internal", "imgtest")); statErr == nil {
		t.Error("无人引用的 internal/imgtest 应被删除")
	}
	// 裁剪必须真的减少文件数（这是用户关心的「别复制一堆用不上的东西」）。
	if res.FinalFileCount >= res.FileCount {
		t.Errorf("最终文件数 %d 未小于复制数 %d，说明裁剪没生效",
			res.FinalFileCount, res.FileCount)
	}
	t.Logf("复制 %d 个文件 → 最终 %d 个；删页面 %d 个（保留 %d），删演示包 %d 个 %v",
		res.FileCount, res.FinalFileCount, len(res.RemovedFiles), len(res.KeptFiles),
		len(res.RemovedPkgs), res.RemovedPkgs)
}
