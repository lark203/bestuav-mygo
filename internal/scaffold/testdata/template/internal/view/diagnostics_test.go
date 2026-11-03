package view

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bestuav-mygo/internal/settings"
)

// TestDiagnosticsArchive 验证诊断包内容：settings、环境、说明齐备，库未启用时也能导出。
func TestDiagnosticsArchive(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	a := newTestApp()
	// 写一份真实的设置文件，让诊断包里有东西可打。
	if err := a.settings.Save(); err != nil {
		t.Fatal(err)
	}

	zipPath, err := writeDiagnosticsArchive(a.store, a.basicInfoText())
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	defer os.Remove(zipPath)

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("打开 zip 失败: %v", err)
	}
	defer r.Close()

	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	for _, want := range []string{"settings.json", "env.txt", "README.txt"} {
		if !names[want] {
			t.Errorf("诊断包缺少 %s（实际: %v）", want, names)
		}
	}
	// 未启用数据库时不应有快照，但也不该报错。
	if names["data.snapshot.db"] {
		t.Error("store 为空时不应包含数据库快照")
	}
}

// TestSettingsPathRedirected 确保测试用的临时配置目录生效（防止误写真实用户目录）。
func TestSettingsPathRedirected(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	p, err := settings.Path()
	if err != nil {
		t.Fatal(err)
	}
	if !withinDir(p, dir) {
		t.Fatalf("配置路径 %q 不在临时目录 %q 内", p, dir)
	}
}

func withinDir(path, dir string) bool {
	return len(path) > len(dir) && path[:len(dir)] == dir
}

// TestAddRecentLogsRespectsBudget 诊断包只带得动预算内的日志，且必须优先带最新的。
//
// 日志分片后（app-<日期>.N.log）三天可能有上百 MB，全塞进 zip 会让导出变成巨型下载；
// 但反过来把最新日志漏掉更糟——排查就是要看最新的那几行。
func TestAddRecentLogsRespectsBudget(t *testing.T) {
	old := diagLogBudget
	diagLogBudget = 100
	defer func() { diagLogBudget = old }()

	dir := t.TempDir()
	now := time.Now()
	// 三个各 60 字节的日志，修改时间分别为 3 小时前 / 2 小时前 / 1 小时前。
	files := []struct {
		name string
		age  time.Duration
	}{
		{"app-" + now.Format("2006-01-02") + ".log", 3 * time.Hour},
		{"app-" + now.Format("2006-01-02") + ".2.log", 2 * time.Hour},
		{"app-" + now.Format("2006-01-02") + ".3.log", time.Hour},
	}
	for _, f := range files {
		p := filepath.Join(dir, f.name)
		if err := os.WriteFile(p, make([]byte, 60), 0o644); err != nil {
			t.Fatal(err)
		}
		mt := now.Add(-f.age)
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := addRecentLogs(zw, dir); err != nil {
		t.Fatalf("打包日志: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range r.File {
		got = append(got, strings.TrimPrefix(f.Name, "logs/"))
	}
	// 预算 100、每个 60：只装得下一个，且必须是最新的那个。
	if len(got) != 1 {
		t.Fatalf("预算内应只挑 1 个日志，实际 %v", got)
	}
	if want := "app-" + now.Format("2006-01-02") + ".3.log"; got[0] != want {
		t.Errorf("应优先带最新日志 %q，实际 %q", want, got[0])
	}
}
