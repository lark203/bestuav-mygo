package view

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
)

// TestUpdateDialogLongNotesKeepsActionsVisible 发布说明很长时，说明区必须滚动，
// 底部「忽略 / 立即更新」不能被挤出可视区。
func TestUpdateDialogLongNotesKeepsActionsVisible(t *testing.T) {
	a := newApp(settings.Default())
	a.page = "settings"
	a.lastUpdateCheck = time.Now() // 阻止设置页 onEnter 的自动检查覆盖测试里的假更新
	a.showUpdateDialog = true
	a.availableUpdate = &mygo.Update{
		Version: "0.1.7",
		Date:    time.Now(),
		Notes:   strings.Repeat("这是一段很长的发布说明，用来验证弹窗不会把按钮挤出屏幕。\n", 80),
	}

	// 比截图更矮的窗口：如果有布局问题，这里最容易暴露。
	tt := ui.NewTester(a.View, 606, 600)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"发现新版本", "0.1.7", "立即更新", "忽略"} {
		if !tt.HasText(want) {
			t.Fatalf("弹窗缺少 %q，当前文本: %v", want, tt.Texts())
		}
	}

	if err := tt.Click("忽略"); err != nil {
		t.Fatalf("长说明下「忽略」按钮不可点击: %v", err)
	}
	tt.Frame()
	if a.showUpdateDialog {
		t.Fatal("点击「忽略」后弹窗应关闭")
	}
	if a.saveTimer != nil {
		a.saveTimer.Stop()
	}
}
