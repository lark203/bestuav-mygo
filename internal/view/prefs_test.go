package view

import (
	"testing"

	"bestuav-mygo/internal/settings"
)

// orderFilter 测试用的页面私有状态（含嵌套与切片，确认 JSON 能往返）。
type orderFilter struct {
	Kind  string   `json:"kind"`
	Tags  []string `json:"tags"`
	Limit int      `json:"limit"`
}

// TestPrefsRoundTrip 页面私有状态：写进去读得回来，换一个 App 实例（模拟重启）也在。
func TestPrefsRoundTrip(t *testing.T) {
	s := openTempStore(t)

	a := newApp(settings.Default())
	a.AttachStore(s)
	a.page = "orders"

	want := orderFilter{Kind: "批发", Tags: []string{"华北", "加急"}, Limit: 20}
	if err := a.Prefs().Set("filter", want); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var got orderFilter
	if !a.Prefs().Get("filter", &got) {
		t.Fatal("刚写的状态应能读到")
	}
	if got.Kind != want.Kind || got.Limit != want.Limit || len(got.Tags) != 2 {
		t.Fatalf("读回的值不一致: %+v", got)
	}

	// 模拟重启：新 App + 同一个库（内存缓存为空，必须走数据库）。
	b := newApp(settings.Default())
	b.AttachStore(s)
	b.page = "orders"
	var afterRestart orderFilter
	if !b.Prefs().Get("filter", &afterRestart) {
		t.Fatal("重启后应仍能读到")
	}
	if afterRestart.Kind != want.Kind {
		t.Errorf("重启后值不一致: %+v", afterRestart)
	}
}

// TestPrefsNilStoreFallsBackToMemory 数据库不可用时退化为纯内存：本次运行内可用。
func TestPrefsNilStoreFallsBackToMemory(t *testing.T) {
	a := newTestApp() // 不挂库
	a.page = "orders"

	if err := a.Prefs().Set("filter", orderFilter{Kind: "零售"}); err != nil {
		t.Fatalf("无库时 Set 不应报错: %v", err)
	}
	var got orderFilter
	if !a.Prefs().Get("filter", &got) || got.Kind != "零售" {
		t.Fatalf("无库时应走内存缓存，实际 %+v", got)
	}
}

// TestPrefsIsolatedPerPage 同一 key 在不同页面之间互不干扰。
func TestPrefsIsolatedPerPage(t *testing.T) {
	a := newTestApp()
	a.AttachStore(openTempStore(t))

	a.page = "orders"
	if err := a.Prefs().Set("filter", orderFilter{Kind: "批发"}); err != nil {
		t.Fatal(err)
	}

	a.page = "task"
	var got orderFilter
	if a.Prefs().Get("filter", &got) {
		t.Fatalf("另一个页面不应读到 orders 的状态: %+v", got)
	}
}

// TestPrefsBadJSONTreatedAsUnset 历史 JSON 解析失败按「没存过」处理，不能把页面卡死。
func TestPrefsBadJSONTreatedAsUnset(t *testing.T) {
	s := openTempStore(t)
	a := newApp(settings.Default())
	a.AttachStore(s)
	a.page = "orders"

	// 绕过序列化直接写入脏值（模拟旧版本留下的结构）。
	if err := s.SetPagePref("orders", "filter", `{"kind":`); err != nil {
		t.Fatal(err)
	}
	var got orderFilter
	if a.Prefs().Get("filter", &got) {
		t.Fatal("脏数据应被当作未设置")
	}
	if got.Kind != "" {
		t.Errorf("脏数据不应污染目标值: %+v", got)
	}
}

// TestPrefsDelete 删除后回到「没存过」，内存缓存也不能留旧值。
func TestPrefsDelete(t *testing.T) {
	a := newTestApp()
	a.AttachStore(openTempStore(t))
	a.page = "orders"

	if err := a.Prefs().Set("filter", orderFilter{Kind: "批发"}); err != nil {
		t.Fatal(err)
	}
	a.Prefs().Delete("filter")

	var got orderFilter
	if a.Prefs().Get("filter", &got) {
		t.Fatalf("删除后不应再读到: %+v", got)
	}
}

// TestPrefsRejectsUnserializable 不能序列化的值返回错误（页面可选择提示）。
func TestPrefsRejectsUnserializable(t *testing.T) {
	a := newTestApp()
	a.page = "orders"
	if err := a.Prefs().Set("bad", make(chan int)); err == nil {
		t.Fatal("channel 不能序列化，应返回错误")
	}
}
