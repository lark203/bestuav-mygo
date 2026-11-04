package view

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// waitText 反复绘制直到出现目标文本：异步加载要等后台协程 + 下一次重绘。
func waitText(t *testing.T, tt *ui.Tester, want string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		tt.Frame()
		if tt.HasText(want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待 %q 超时；当前文本: %v", want, tt.Texts())
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// newAsyncApp 设置演示页的人为延迟（测试里调小，但保留到足以观察「加载中」）。
func newAsyncApp(t *testing.T, delay time.Duration) *App {
	t.Helper()
	old := asyncDemoDelay
	asyncDemoDelay = delay
	t.Cleanup(func() { asyncDemoDelay = old })
	a := newTestApp()
	a.page = asyncDemoPageID
	return a
}

// TestAsyncDemoLoads 首次进入：先显示加载中，随后渲染数据；再点「重新加载」还能刷回来。
func TestAsyncDemoLoads(t *testing.T) {
	a := newAsyncApp(t, 200*time.Millisecond)
	tt := render(a)

	// 第一帧就应当发起加载并显示占位，而不是空白或卡住。
	mustHave(t, tt, "正在加载…")
	waitText(t, tt, "同步授权数据")

	if err := tt.Click("重新加载"); err != nil {
		t.Fatalf("点击「重新加载」失败: %v", err)
	}
	waitText(t, tt, "同步授权数据")
}

// TestAsyncDemoFailureAndRetry 失败显示错误卡片；点「重试」后恢复。
func TestAsyncDemoFailureAndRetry(t *testing.T) {
	a := newAsyncApp(t, 10*time.Millisecond)
	st := pageData(a, asyncDemoPageID, "demo", func() asyncDemoState { return asyncDemoState{} })
	st.fail = true

	tt := render(a)
	waitText(t, tt, "加载失败")
	mustHave(t, tt, "模拟失败：服务暂时不可用", "重试")
	if tt.HasText("同步授权数据") {
		t.Error("失败时不应同时显示上一次的数据")
	}

	st.fail = false
	if err := tt.Click("重试"); err != nil {
		t.Fatalf("点击「重试」失败: %v", err)
	}
	waitText(t, tt, "同步授权数据")
}

// TestAsyncDemoEmpty 空列表按正常结果渲染（由页面自己给空态提示）。
func TestAsyncDemoEmpty(t *testing.T) {
	a := newAsyncApp(t, 10*time.Millisecond)
	st := pageData(a, asyncDemoPageID, "demo", func() asyncDemoState { return asyncDemoState{} })
	st.empty = true

	tt := render(a)
	waitText(t, tt, "没有数据（空列表也是一种正常结果）")
}

// TestAsyncReloadsOnPageReenter 缓存页切走再回来会重新加载（进入即刷新）。
func TestAsyncReloadsOnPageReenter(t *testing.T) {
	a := newAsyncApp(t, 10*time.Millisecond)
	tt := render(a)
	waitText(t, tt, "同步授权数据")

	// 让下一次加载失败，切走再回来：若没有自动重载，页面仍是旧数据。
	st := pageData(a, asyncDemoPageID, "demo", func() asyncDemoState { return asyncDemoState{} })
	st.fail = true
	a.page = "task"
	tt.Frame()
	a.page = asyncDemoPageID
	tt.Frame()
	waitText(t, tt, "加载失败")
}

// TestAsyncDiscardsStaleResult 过期代次的结果不采纳（切页后迟到的加载）。
func TestAsyncDiscardsStaleResult(t *testing.T) {
	a := newTestApp()
	a.page = asyncDemoPageID
	h := pageAsync(a, "unit", func() (int, error) { return 1, nil })
	h.st.gen = 7
	h.st.loading = true

	// 旧的一次加载（gen=6）迟到：应被丢弃，仍在加载中。
	h.st.mu.Lock()
	h.st.pending = &asyncResult[int]{gen: 6, value: 42}
	h.st.mu.Unlock()
	h.drain()
	if !h.Loading() || h.Value() != 0 {
		t.Fatalf("过期结果不应被采纳: loading=%v value=%v", h.Loading(), h.Value())
	}

	// 最新的那次（gen=7）到达：采纳。
	h.st.mu.Lock()
	h.st.pending = &asyncResult[int]{gen: 7, value: 42}
	h.st.mu.Unlock()
	h.drain()
	if h.Loading() || h.Value() != 42 {
		t.Fatalf("最新结果应被采纳: loading=%v value=%v", h.Loading(), h.Value())
	}
}

// TestAsyncPanicBecomesError 页面 load 里 panic 只变成一次「加载失败」，不崩窗口。
func TestAsyncPanicBecomesError(t *testing.T) {
	a := newTestApp()
	a.page = asyncDemoPageID
	h := pageAsync(a, "panic", func() (int, error) { panic("boom") })
	h.start()

	deadline := time.Now().Add(2 * time.Second)
	for {
		h.st.mu.Lock()
		pending := h.st.pending
		h.st.mu.Unlock()
		if pending != nil {
			h.drain()
			if h.Err() == nil {
				t.Fatal("panic 应转成加载错误")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("等待加载结果超时")
		}
		time.Sleep(time.Millisecond)
	}
}
