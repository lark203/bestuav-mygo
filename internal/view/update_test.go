package view

import "testing"

// TestSafeModeSkipsUpdateCheck 安全模式必须彻底挡住更新链路。
//
// 这一条不是形式主义：README 明确承诺「不检查更新，标题栏不会出现更新图标/胶囊」，
// 而 checkForUpdates 里有一行 `a.updateDisabled = false`——守卫只要写在那行之后，
// 手动点一次标题栏图标就会把安全模式标记抹掉、照常发请求，行为与文档相反。
func TestSafeModeSkipsUpdateCheck(t *testing.T) {
	a := newTestApp()
	a.SetSafeMode(true)
	if !a.updateDisabled {
		t.Fatal("SetSafeMode(true) 之后 updateDisabled 应为 true")
	}

	// 模拟「进入设置页自动检查」与「点标题栏手动检查」两条入口。
	a.checkForUpdates(false)
	a.checkForUpdates(true)

	if a.updateChecking {
		t.Error("安全模式下不应发起更新检查（updateChecking 被置真）")
	}
	if !a.updateDisabled {
		t.Error("安全模式标记被抹掉了（updateDisabled 应为 true）")
	}
	if a.updateChecked {
		t.Error("安全模式下不应产生「已检查过」的状态")
	}
}

// TestSafeModeSkipsInstall 安全模式不允许安装更新（安装会替换可执行文件并重启）。
func TestSafeModeSkipsInstall(t *testing.T) {
	a := newTestApp()
	a.SetSafeMode(true)
	a.installUpdate()
	if a.updateInstalling {
		t.Error("安全模式下不应进入安装流程")
	}
}

// TestSafeModeNoticeOnSettingsPage 安全模式下设置页要写明原因，且不给「检查更新」按钮。
func TestSafeModeNoticeOnSettingsPage(t *testing.T) {
	a := newTestApp()
	a.SetSafeMode(true)
	a.page = "settings"
	mustHave(t, render(a), "安全模式启动：本次已跳过更新检查与数据库")
}
