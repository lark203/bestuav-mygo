package profile

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bestuav-mygo/internal/store"
)

// fixture 一个「旧机器」：设置文件、数据库、背景图。
type fixture struct {
	dir          string
	settingsPath string
	dbPath       string
	dataDir      string
	bgPath       string
	zipPath      string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	root := t.TempDir()
	f := fixture{
		dir:          root,
		settingsPath: filepath.Join(root, "config", "settings.json"),
		dbPath:       filepath.Join(root, "data", "data.db"),
		dataDir:      filepath.Join(root, "data"),
		bgPath:       filepath.Join(root, "wall.png"),
		zipPath:      filepath.Join(root, "me.mygo-profile.zip"),
	}
	if err := os.MkdirAll(filepath.Dir(f.settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(f.dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	settingsJSON, err := json.MarshalIndent(map[string]any{
		"theme":              "dark",
		"palette":            "teal",
		"tone":               "green",
		"fontSize":           16,
		"shortcuts":          map[string]string{"nav.task": "cmd+9"},
		"openAtLogin":        true,
		"allowMultiInstance": true,
		"ignoredVersion":     "0.1.2",
		"syncFolder":         "/old/machine/sync",
		"backgroundImage":    f.bgPath,
		"futureOption":       map[string]bool{"keep": true},
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, f.settingsPath, string(settingsJSON)+"\n")
	writeFile(t, f.bgPath, "PNGDATA")
	s, err := store.Open(f.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.InsertTask("同步授权数据", "轻量级", "已完成", "0.8s"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPagePref("orders", "filter", `{"kind":"批发"}`); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	return f
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exportFixture(t *testing.T, f fixture, includeData bool) ExportResult {
	t.Helper()
	res, err := Export(ExportOptions{
		Out:           f.zipPath,
		SettingsPath:  f.settingsPath,
		DBPath:        f.dbPath,
		Background:    f.bgPath,
		IncludeData:   includeData,
		AppID:         "com.bestuav.quantum",
		AppVersion:    "0.1.10",
		SchemaVersion: store.SchemaVersion(),
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	return res
}

// TestExportImportRoundTrip 设置、页面状态、任务与背景图都能搬过去；
// 设备相关字段保留目标机器的值。
func TestExportImportRoundTrip(t *testing.T) {
	src := newFixture(t)
	res := exportFixture(t, src, true)
	if !res.IncludesData || !res.IncludesImage {
		t.Fatalf("导出内容不完整: %+v", res)
	}

	m, err := Inspect(src.zipPath)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if m.Format != Format || m.App != "com.bestuav.quantum" || m.Device == "" ||
		!contains(m.Includes, "data") || !contains(m.Includes, "background") {
		t.Fatalf("manifest 不正确: %+v", m)
	}

	// 目标机器：已有自己的设置与库。
	dst := fixture{
		dir:          t.TempDir(),
		settingsPath: filepath.Join(t.TempDir(), "settings.json"),
		dbPath:       filepath.Join(t.TempDir(), "data.db"),
		dataDir:      t.TempDir(),
	}
	writeFile(t, dst.settingsPath, `{"theme":"light","openAtLogin":false,"allowMultiInstance":false,"ignoredVersion":"","fontSize":12}`)
	s, err := store.Open(dst.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.InsertTask("本机旧任务", "轻量级", "已完成", "0.1s"); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	imp, err := Import(ImportOptions{
		In:            src.zipPath,
		SettingsPath:  dst.settingsPath,
		DBPath:        dst.dbPath,
		DataDir:       dst.dataDir,
		AppID:         "com.bestuav.quantum",
		SchemaVersion: store.SchemaVersion(),
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if !imp.ImportedData || !imp.ImportedImage {
		t.Fatalf("导入结果不完整: %+v", imp)
	}
	if imp.SettingsBackup == "" || imp.DBBackup == "" {
		t.Error("导入前应备份现有设置与数据库")
	}
	if _, err := os.Stat(imp.SettingsBackup); err != nil {
		t.Errorf("设置备份不存在: %v", err)
	}

	// 设置：可移植字段来自包，设备字段保留本机值，未知字段也保留。
	data, err := os.ReadFile(dst.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("导入后的设置不是合法 JSON: %v\n%s", err, data)
	}
	if got["theme"] != "dark" || got["palette"] != "teal" || got["tone"] != "green" {
		t.Errorf("可移植设置未导入: %v", got)
	}
	if got["fontSize"].(float64) != 16 {
		t.Errorf("字号未导入: %v", got["fontSize"])
	}
	if got["openAtLogin"] != false || got["allowMultiInstance"] != false {
		t.Errorf("设备相关字段应保留本机值: %v / %v", got["openAtLogin"], got["allowMultiInstance"])
	}
	if got["futureOption"] == nil {
		t.Error("包内的未知字段应原样保留（向前兼容）")
	}
	if bg, _ := got["backgroundImage"].(string); bg == "" || bg == src.bgPath {
		t.Errorf("背景图路径应改写为本机新路径，实际 %q", bg)
	}
	if _, err := os.Stat(got["backgroundImage"].(string)); err != nil {
		t.Errorf("背景图应复制到本机: %v", err)
	}

	// 数据库：任务与页面状态来自包（本机旧任务被替换）。
	s2, err := store.Open(dst.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	rows, err := s2.RecentTasks(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Name != "同步授权数据" {
		t.Errorf("任务应来自配置包，实际 %+v", rows)
	}
	if v, ok, _ := s2.PagePref("orders", "filter"); !ok || !strings.Contains(v, "批发") {
		t.Errorf("页面状态未导入: %q ok=%v", v, ok)
	}
}

// TestImportWithoutData 只导出设置时不碰目标数据库。
func TestImportWithoutData(t *testing.T) {
	src := newFixture(t)
	exportFixture(t, src, false)

	dst := fixture{
		dir:          t.TempDir(),
		settingsPath: filepath.Join(t.TempDir(), "settings.json"),
		dbPath:       filepath.Join(t.TempDir(), "data.db"),
		dataDir:      t.TempDir(),
	}
	writeFile(t, dst.settingsPath, `{"theme":"light"}`)
	s, _ := store.Open(dst.dbPath)
	_, _ = s.InsertTask("保留我", "轻量级", "已完成", "1s")
	_ = s.Close()

	imp, err := Import(ImportOptions{
		In: src.zipPath, SettingsPath: dst.settingsPath, DBPath: dst.dbPath,
		DataDir: dst.dataDir, AppID: "com.bestuav.quantum", SchemaVersion: store.SchemaVersion(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if imp.ImportedData {
		t.Error("包内没有 data.db 时不应导入数据")
	}
	s2, _ := store.Open(dst.dbPath)
	defer s2.Close()
	rows, _ := s2.RecentTasks(10)
	if len(rows) != 1 || rows[0].Name != "保留我" {
		t.Errorf("不应改动本机数据库: %+v", rows)
	}
}

// TestImportRejects 不兼容的包在改文件之前就被拒绝。
func TestImportRejects(t *testing.T) {
	src := newFixture(t)
	exportFixture(t, src, true)

	dst := fixture{
		dir:          t.TempDir(),
		settingsPath: filepath.Join(t.TempDir(), "settings.json"),
		dbPath:       filepath.Join(t.TempDir(), "data.db"),
		dataDir:      t.TempDir(),
	}
	writeFile(t, dst.settingsPath, `{"theme":"light"}`)

	if _, err := Import(ImportOptions{
		In: src.zipPath, SettingsPath: dst.settingsPath, DBPath: dst.dbPath,
		DataDir: dst.dataDir, AppID: "com.other.app", SchemaVersion: store.SchemaVersion(),
	}); err == nil || !strings.Contains(err.Error(), "另一个应用") {
		t.Errorf("应用标识不符应拒绝，实际 %v", err)
	}
	if _, err := Import(ImportOptions{
		In: src.zipPath, SettingsPath: dst.settingsPath, DBPath: dst.dbPath,
		DataDir: dst.dataDir, AppID: "com.bestuav.quantum", SchemaVersion: "1",
	}); err == nil || !strings.Contains(err.Error(), "更新版本") {
		t.Errorf("未来 schema 应拒绝，实际 %v", err)
	}
	// 被拒绝后本机设置原样不动。
	data, _ := os.ReadFile(dst.settingsPath)
	if string(data) != `{"theme":"light"}` {
		t.Errorf("校验失败不应改设置: %s", data)
	}

	// 不是 zip 的文件。
	bad := filepath.Join(t.TempDir(), "bad.zip")
	writeFile(t, bad, "not a zip")
	if _, err := Inspect(bad); err == nil {
		t.Error("损坏的包应报错")
	}
}

// TestListAndDeviceName 同步文件夹列表按导出时间倒序，坏文件跳过。
func TestListAndDeviceName(t *testing.T) {
	src := newFixture(t)
	exportFixture(t, src, false)

	dir := t.TempDir()
	// 复制两份，只改 manifest 里的时间，验证排序。
	zr, err := zip.OpenReader(src.zipPath)
	if err != nil {
		t.Fatal(err)
	}
	var settingsBytes, manifestBytes []byte
	for _, f := range zr.File {
		rc, _ := f.Open()
		buf := make([]byte, f.UncompressedSize64)
		_, _ = rc.Read(buf)
		_ = rc.Close()
		switch f.Name {
		case manifestName:
			manifestBytes = buf
		case settingsName:
			settingsBytes = buf
		}
	}
	_ = zr.Close()
	for i, name := range []string{"a-old" + FileExt, "b-new" + FileExt} {
		var m map[string]any
		_ = json.Unmarshal(manifestBytes, &m)
		if i == 0 {
			m["exportedAt"] = "2026-01-01T00:00:00Z"
		} else {
			m["exportedAt"] = "2026-06-01T00:00:00Z"
		}
		writeZip(t, filepath.Join(dir, name), m, settingsBytes)
	}
	writeFile(t, filepath.Join(dir, "broken"+FileExt), "nope")

	bundles, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundles) != 2 {
		t.Fatalf("应列出 2 个有效包，实际 %d", len(bundles))
	}
	if !strings.Contains(bundles[0].Path, "b-new") {
		t.Errorf("应按导出时间倒序，实际 %v", bundles)
	}
	if DeviceName() == "" {
		t.Error("设备名不应为空")
	}
}

// writeZip 生成一个最小配置包（测试用）。
func writeZip(t *testing.T, path string, manifest map[string]any, settings []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create(manifestName)
	_ = json.NewEncoder(w).Encode(manifest)
	w, _ = zw.Create(settingsName)
	_, _ = w.Write(settings)
	_ = zw.Close()
	_ = f.Close()
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
