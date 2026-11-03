package store

import (
	"path/filepath"
	"testing"
)

// openTemp 在临时目录里打开一个全新的数据库，测试结束自动关闭。
func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// TestTaskRoundTrip 验证任务写入后能按时间倒序读出，字段不丢。
func TestTaskRoundTrip(t *testing.T) {
	s := openTemp(t)

	if _, err := s.InsertTask("同步授权数据", "轻量级", "已完成", "0.8s"); err != nil {
		t.Fatalf("InsertTask: %v", err)
	}
	if _, err := s.InsertTask("重建检索索引", "重量级", "排队", "—"); err != nil {
		t.Fatalf("InsertTask: %v", err)
	}

	rows, err := s.RecentTasks(10)
	if err != nil {
		t.Fatalf("RecentTasks: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("应有 2 条记录，实际 %d", len(rows))
	}
	// 最近写入的排在最前（列表就是按这个顺序展示的）。
	if rows[0].Name != "重建检索索引" || rows[1].Name != "同步授权数据" {
		t.Fatalf("顺序错误: %+v", rows)
	}
	if rows[0].Kind != "重量级" || rows[0].Status != "排队" || rows[0].Cost != "—" {
		t.Errorf("字段丢失: %+v", rows[0])
	}
	if rows[0].CreatedAt.IsZero() {
		t.Error("created_at 未正确解析")
	}
}

// TestPersistAcrossReopen 验证关库重开后数据仍在（真正的「持久化」）。
func TestPersistAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")

	s1, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := s1.InsertTask("导出月度报表", "轻量级", "已完成", "2.1s"); err != nil {
		t.Fatalf("InsertTask: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("重新 Open: %v", err)
	}
	defer s2.Close()

	rows, err := s2.RecentTasks(10)
	if err != nil {
		t.Fatalf("RecentTasks: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "导出月度报表" {
		t.Fatalf("重开后数据丢失: %+v", rows)
	}
}

// TestClearFinishedTasks 验证只删「已完成」，排队/运行中的保留。
func TestClearFinishedTasks(t *testing.T) {
	s := openTemp(t)
	for _, r := range []struct{ name, status string }{
		{"已完成的任务", "已完成"},
		{"排队中的任务", "排队"},
		{"运行中的任务", "运行中"},
	} {
		if _, err := s.InsertTask(r.name, "轻量级", r.status, "1s"); err != nil {
			t.Fatalf("InsertTask: %v", err)
		}
	}

	n, err := s.DeleteFinishedTasks()
	if err != nil {
		t.Fatalf("DeleteFinishedTasks: %v", err)
	}
	if n != 1 {
		t.Errorf("应删除 1 条，实际 %d", n)
	}

	rows, err := s.RecentTasks(10)
	if err != nil {
		t.Fatalf("RecentTasks: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("应剩 2 条，实际 %d", len(rows))
	}
	for _, r := range rows {
		if r.Status == "已完成" {
			t.Errorf("已完成的记录不应残留: %+v", r)
		}
	}
}

// TestLicenseAndEvent 验证授权记录、事件入库与计数统计。
func TestLicenseAndEvent(t *testing.T) {
	s := openTemp(t)

	if _, err := s.InsertLicense("北京必优航空科技", "企业版", "1 年", "payload.sig"); err != nil {
		t.Fatalf("InsertLicense: %v", err)
	}
	lics, err := s.RecentLicenses(10)
	if err != nil {
		t.Fatalf("RecentLicenses: %v", err)
	}
	if len(lics) != 1 || lics[0].Subject != "北京必优航空科技" || lics[0].Token != "payload.sig" {
		t.Fatalf("授权记录错误: %+v", lics)
	}

	if err := s.RecordEvent("INFO", "app", "应用启动", "os=windows"); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if err := s.RecordEvent("ERROR", "store", "写入失败", "err=disk full"); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	events, err := s.RecentEvents(10)
	if err != nil {
		t.Fatalf("RecentEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("应有 2 条事件，实际 %d", len(events))
	}
	if events[0].Level != "ERROR" || events[0].Kind != "store" {
		t.Errorf("事件顺序/内容错误: %+v", events[0])
	}
	if events[0].Detail != "err=disk full" {
		t.Errorf("detail 丢失: %q", events[0].Detail)
	}

	st, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if st.Licenses != 1 || st.Events != 2 || st.Errors != 1 || st.Tasks != 0 {
		t.Errorf("统计错误: %+v", st)
	}
}

// TestFlag 验证元数据读写（首次运行标记就靠它）。
func TestFlag(t *testing.T) {
	s := openTemp(t)

	if _, ok, err := s.Flag("seeded"); err != nil || ok {
		t.Fatalf("未写入时应 ok=false: ok=%v err=%v", ok, err)
	}
	if err := s.SetFlag("seeded", "1"); err != nil {
		t.Fatalf("SetFlag: %v", err)
	}
	if v, ok, err := s.Flag("seeded"); err != nil || !ok || v != "1" {
		t.Fatalf("读取失败: v=%q ok=%v err=%v", v, ok, err)
	}
	// 覆盖写不应报错。
	if err := s.SetFlag("seeded", "2"); err != nil {
		t.Fatalf("SetFlag 覆盖: %v", err)
	}
	if v, _, _ := s.Flag("seeded"); v != "2" {
		t.Errorf("覆盖失败: %q", v)
	}
}

// TestOpenIsIdempotent 验证同一路径重复打开（建表语句幂等）不会报错或清空数据。
func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	for i := 0; i < 2; i++ {
		s, err := Open(path)
		if err != nil {
			t.Fatalf("第 %d 次 Open: %v", i+1, err)
		}
		if i == 0 {
			if _, err := s.InsertTask("首次运行写入", "轻量级", "已完成", "1s"); err != nil {
				t.Fatalf("InsertTask: %v", err)
			}
		} else {
			rows, err := s.RecentTasks(10)
			if err != nil {
				t.Fatalf("RecentTasks: %v", err)
			}
			if len(rows) != 1 {
				t.Errorf("重复建表不应丢数据，实际 %d 条", len(rows))
			}
		}
		_ = s.Close()
	}
}

// TestRecentEventsLimit 验证 limit 为 0/负数时被兜底处理（不会返回全表或报错）。
func TestRecentEventsLimit(t *testing.T) {
	s := openTemp(t)
	for i := 0; i < 3; i++ {
		if err := s.RecordEvent("INFO", "unit", "e", ""); err != nil {
			t.Fatalf("RecordEvent: %v", err)
		}
	}
	if _, err := s.RecentEvents(0); err != nil {
		t.Errorf("limit=0 不应报错: %v", err)
	}
	if rows, err := s.RecentEvents(-1); err != nil || len(rows) != 3 {
		t.Errorf("limit=-1 应回落到默认值: n=%d err=%v", len(rows), err)
	}
}
