package view

import (
	"testing"
	"time"

	"bestuav-mygo/internal/settings"
)

// TestPageCacheKeepsState 验证「缓存页面」：离开再回来，页面状态还在。
func TestPageCacheKeepsState(t *testing.T) {
	a := newTestApp()

	// 进入 task 并写入页面私有状态
	a.page = "task"
	a.syncPages()
	v := pageData(a, "task", "counter", func() int { return 0 })
	*v = 42

	// 离开（task 是 keepCached，状态应保留）
	a.page = "home"
	a.syncPages()
	if _, ok := a.pages["task"]; !ok {
		t.Fatal("缓存页面离开后不应被销毁")
	}

	// 再进入：状态应还是 42
	a.page = "task"
	a.syncPages()
	v2 := pageData(a, "task", "counter", func() int { return 0 })
	if *v2 != 42 {
		t.Fatalf("缓存页面状态丢失，期望 42，实际 %d", *v2)
	}
}

// TestPageFreshDestroysState 验证「不缓存页面」：离开即销毁，回来是全新状态。
func TestPageFreshDestroysState(t *testing.T) {
	a := newTestApp()

	a.page = "gen-license"
	a.syncPages()
	v := pageData(a, "gen-license", "draft", func() string { return "" })
	*v = "未提交的草稿"

	// 离开（keepFresh → 立即销毁）
	a.page = "home"
	a.syncPages()
	if _, ok := a.pages["gen-license"]; ok {
		t.Fatal("不缓存的页面离开后应被销毁")
	}

	// 再进入：草稿应已丢失
	a.page = "gen-license"
	a.syncPages()
	v2 := pageData(a, "gen-license", "draft", func() string { return "" })
	if *v2 != "" {
		t.Fatalf("不缓存页面应重建状态，实际仍为 %q", *v2)
	}
}

// TestPagePinnedSurvivesIdle 验证常驻页面（首页）不会被闲置淘汰。
func TestPagePinnedSurvivesIdle(t *testing.T) {
	a := newTestApp()
	a.page = "home"
	a.syncPages()

	// 手动把最后访问时间推到很久以前
	a.pages["home"].lastSeen = time.Now().Add(-10 * time.Hour)
	a.evictIdlePages(time.Now())

	if _, ok := a.pages["home"]; !ok {
		t.Fatal("常驻页面不应被闲置淘汰")
	}
}

// TestIdleEviction 验证缓存页面闲置超时后被销毁，且 onDestroy 语义（数据清空）生效。
func TestIdleEviction(t *testing.T) {
	s := settings.Default()
	s.PageIdleMinutes = 10 // 闲置 10 分钟销毁
	a := newApp(s)

	a.page = "task"
	a.syncPages()
	v := pageData(a, "task", "counter", func() int { return 0 })
	*v = 7

	// 模拟闲置 11 分钟
	a.pages["task"].lastSeen = time.Now().Add(-11 * time.Minute)
	a.evictIdlePages(time.Now())

	if _, ok := a.pages["task"]; ok {
		t.Fatal("闲置超时的缓存页面应被销毁")
	}
}

// TestIdleZeroDisablesEviction 验证设置里填 0 = 不按闲置销毁。
func TestIdleZeroDisablesEviction(t *testing.T) {
	s := settings.Default()
	s.PageIdleMinutes = 0
	a := newApp(s)

	a.page = "task"
	a.syncPages()
	a.pages["task"].lastSeen = time.Now().Add(-72 * time.Hour)
	a.evictIdlePages(time.Now())

	if _, ok := a.pages["task"]; !ok {
		t.Fatal("闲置销毁为 0 时不应淘汰缓存页面")
	}
}

// TestScrollStateKeptPerPage 验证每页各自持有滚动状态，销毁后归零。
func TestScrollStateKeptPerPage(t *testing.T) {
	a := newTestApp()
	a.page = "task"
	a.syncPages()
	a.ScrollFor("task").Y = 320

	// 切走再切回：位置仍在
	a.page = "home"
	a.syncPages()
	a.page = "task"
	a.syncPages()
	if got := a.ScrollFor("task").Y; got != 320 {
		t.Fatalf("缓存页面的滚动位置应保留 320，实际 %v", got)
	}

	// 销毁后归零
	a.destroyPage("task")
	if got := a.ScrollFor("task").Y; got != 0 {
		t.Fatalf("销毁后滚动位置应归零，实际 %v", got)
	}
}

// TestRegistryConsistency 验证注册表自身的一致性：
// 有 build 的页面必须有唯一 id、非空标题与图标，避免出现「加了菜单点不开」。
func TestRegistryConsistency(t *testing.T) {
	seen := map[string]bool{}
	eachPage(func(it navItem) {
		if it.id == "" || it.label == "" || it.icon == nil {
			t.Errorf("菜单项字段不完整: %+v", it)
		}
		if seen[it.id] {
			t.Errorf("菜单 id 重复: %q", it.id)
		}
		seen[it.id] = true
	})
	if len(seen) < 10 {
		t.Errorf("注册表页面数量异常: %d", len(seen))
	}
}
