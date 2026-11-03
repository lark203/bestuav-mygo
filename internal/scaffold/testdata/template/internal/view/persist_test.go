package view

import (
	"path/filepath"
	"testing"

	"bestuav-mygo/internal/settings"
	"bestuav-mygo/internal/store"
)

// openTempStore 在临时目录开一个数据库（测试结束自动关闭）。
func openTempStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// TestAttachStoreSeedsDemoOnce 验证首次挂载数据库时灌入演示任务，且只灌一次
// （用户清空后不会在下次启动又冒出来）。
func TestAttachStoreSeedsDemoOnce(t *testing.T) {
	s := openTempStore(t)

	a := newApp(settings.Default())
	a.AttachStore(s)
	if got, want := len(a.tasks), len(demoTaskRows()); got != want {
		t.Fatalf("首次应灌入 %d 条演示任务，实际 %d", want, got)
	}

	// 用户清空已完成后，重新挂载不应再次灌入。
	a.clearFinishedTasks()
	before := len(a.tasks)

	b := newApp(settings.Default())
	b.AttachStore(s)
	if len(b.tasks) != before {
		t.Errorf("不应重复灌入演示数据: 期望 %d 条，实际 %d", before, len(b.tasks))
	}
}

// TestTasksPersistAcrossAppRestart 验证任务写入数据库后，新 App 实例能读回来。
func TestTasksPersistAcrossAppRestart(t *testing.T) {
	s := openTempStore(t)

	a := newApp(settings.Default())
	a.AttachStore(s)
	a.pushTaskRow("单元测试任务", "轻量级", "已完成")

	// 模拟重启：全新的 App，接同一个库。
	b := newApp(settings.Default())
	b.AttachStore(s)

	if len(b.tasks) == 0 {
		t.Fatal("重启后任务列表为空")
	}
	if b.tasks[0].name != "单元测试任务" {
		t.Errorf("最新任务应在最前: %+v", b.tasks[0])
	}
	if b.tasks[0].kind != "轻量级" || b.tasks[0].status != "已完成" {
		t.Errorf("任务字段丢失: %+v", b.tasks[0])
	}
}

// TestLicenseHistoryPersists 验证生成的授权凭证写入数据库并可读回。
func TestLicenseHistoryPersists(t *testing.T) {
	s := openTempStore(t)

	a := newApp(settings.Default())
	a.AttachStore(s)
	a.lic.subject = "单元测试公司"
	a.generateLicense()
	if a.lic.result == "" {
		t.Fatal("应生成凭证")
	}

	b := newApp(settings.Default())
	b.AttachStore(s)
	if len(b.lic.history) == 0 {
		t.Fatal("重启后授权历史为空")
	}
	if b.lic.history[0].subject != "单元测试公司" {
		t.Errorf("授权历史内容错误: %+v", b.lic.history[0])
	}
}

// TestClearFinishedAlsoClearsDB 验证「清空已完成」同时作用于数据库。
func TestClearFinishedAlsoClearsDB(t *testing.T) {
	s := openTempStore(t)

	a := newApp(settings.Default())
	a.AttachStore(s)
	a.clearFinishedTasks()

	rows, err := s.RecentTasks(100)
	if err != nil {
		t.Fatalf("RecentTasks: %v", err)
	}
	for _, r := range rows {
		if r.Status == "已完成" {
			t.Errorf("库里仍有已完成记录: %+v", r)
		}
	}
}

// TestNilStoreDegradesGracefully 验证没有数据库时全部读写安全退化（不 panic）。
func TestNilStoreDegradesGracefully(t *testing.T) {
	a := newApp(settings.Default()) // 未 AttachStore
	a.pushTaskRow("无库任务", "轻量级", "已完成")
	a.clearFinishedTasks()
	a.lic.subject = "无库公司"
	a.generateLicense()

	if a.StorePath() != "" {
		t.Errorf("无库时 StorePath 应为空，实际 %q", a.StorePath())
	}
	if got := a.DBStats(); got != (store.Stats{}) {
		t.Errorf("无库时统计应为零值，实际 %+v", got)
	}
	if a.RecentEvents(5) != nil {
		t.Error("无库时事件列表应为 nil")
	}
}
