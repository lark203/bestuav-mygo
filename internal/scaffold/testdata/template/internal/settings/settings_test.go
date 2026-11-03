package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bestuav-mygo/internal/appmeta"
)

// isolate 把「用户配置目录」重定向到临时目录，避免测试写坏开发机上的真实偏好。
//
// os.UserConfigDir 在 Windows 读 %AppData%，在 Linux 读 $XDG_CONFIG_HOME，在
// macOS 读 ~/Library/Application Support；同时设置三者即可跨平台生效。
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	return dir
}

// TestSaveLoadRoundTrip 验证偏好（含新增的背景图与多实例字段）能原样存取。
func TestSaveLoadRoundTrip(t *testing.T) {
	isolate(t)

	s := Default()
	s.Theme = "dark"
	s.FontSize = 16
	s.CloseMode = "minimize"
	s.SidebarCollapsed = true
	s.AllowMultiInstance = true
	s.BackgroundImage = filepath.Join("tmp", "wall.png")
	s.MenuOpacity = 0.5
	s.ContentOpacity = 0.7
	if err := s.Save(); err != nil {
		t.Fatalf("保存失败: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if got.Theme != "dark" || got.FontSize != 16 || got.CloseMode != "minimize" ||
		!got.SidebarCollapsed || !got.AllowMultiInstance ||
		got.BackgroundImage != s.BackgroundImage ||
		got.MenuOpacity != 0.5 || got.ContentOpacity != 0.7 {
		t.Fatalf("读取结果与写入不一致: %+v", got)
	}
}

// TestPathUnderAppDir 验证配置文件位于以应用名命名的子目录下
// （曾经直接写在用户配置目录根下，与其他应用混在一起）。
func TestPathUnderAppDir(t *testing.T) {
	dir := isolate(t)
	p, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if !strings.Contains(p, appmeta.Name) {
		t.Errorf("配置路径应包含应用名 %q，实际 %q", appmeta.Name, p)
	}
	if !strings.HasPrefix(filepath.Clean(p), filepath.Clean(dir)) {
		t.Errorf("配置路径应位于隔离目录内，实际 %q", p)
	}
}

// TestNormalizeBadValues 验证脏数据/越界值被纠正为合法值（含老文件缺字段时
// 反序列化得到的 0 值）。
func TestNormalizeBadValues(t *testing.T) {
	isolate(t)
	p, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	bad, _ := json.Marshal(map[string]any{
		"theme":          "purple",
		"fontSize":       99,
		"closeMode":      "explode",
		"menuOpacity":    0,   // 老文件没有该字段 → 0，应回落默认
		"contentOpacity": 9.9, // 越界
	})
	if err := os.WriteFile(p, bad, 0o644); err != nil {
		t.Fatalf("写脏数据: %v", err)
	}

	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Theme != "system" || s.FontSize != 14 || s.CloseMode != "ask" {
		t.Errorf("枚举/数值未纠正: %+v", s)
	}
	if s.MenuOpacity != DefaultMenuOpacity || s.ContentOpacity != DefaultContentOpacity {
		t.Errorf("不透明度未回落到默认: %v / %v", s.MenuOpacity, s.ContentOpacity)
	}
}

// TestLegacyMigration 验证老版本配置（直接放在用户配置目录根下的 settings.json）
// 会被读取并迁移到新的按应用名分目录的位置，用户不丢偏好。
func TestLegacyMigration(t *testing.T) {
	dir := isolate(t)
	legacy, err := legacyPath()
	if err != nil {
		t.Fatalf("legacyPath: %v", err)
	}
	old, _ := json.Marshal(map[string]any{"theme": "light", "fontSize": 18})
	if err := os.WriteFile(legacy, old, 0o644); err != nil {
		t.Fatalf("写旧配置: %v", err)
	}

	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Theme != "light" || s.FontSize != 18 {
		t.Fatalf("未从旧路径迁移: %+v", s)
	}

	// 迁移后新位置应已有文件
	newPath, _ := Path()
	if _, statErr := os.Stat(newPath); statErr != nil {
		t.Fatalf("迁移后新配置文件不存在: %v (dir=%s)", statErr, dir)
	}
}
