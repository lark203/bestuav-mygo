package view

import "testing"

// TestNotifyTaskDoneNoWindow 没有窗口（离屏 / 单测）时通知是空操作，不能 panic。
func TestNotifyTaskDoneNoWindow(t *testing.T) {
	a := newTestApp()
	a.notifyTaskDone("测试任务")
}
