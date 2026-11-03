package view

import (
	"archive/zip"
	"os"
	"testing"

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

	zipPath, err := a.writeDiagnosticsArchive()
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
