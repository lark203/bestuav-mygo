package view

import (
	"testing"

	"bestuav-mygo/internal/settings"
)

// TestSettingsShowsProfileSection 设置页「数据与诊断」有导出/导入与同步文件夹入口。
func TestSettingsShowsProfileSection(t *testing.T) {
	a := newTestApp()
	a.page = settingsPageID
	a.setSettingsGroup(settingsGroupAbout)
	mustHave(t, render(a), "配置与数据", "导出设置…", "导出设置与数据…", "导入…", "同步文件夹")
}

// TestSyncFolderEmptyHint 未设置同步文件夹时给出引导，而不是空区块。
func TestSyncFolderEmptyHint(t *testing.T) {
	a := newTestApp()
	a.settings.SyncFolder = ""
	a.page = settingsPageID
	a.setSettingsGroup(settingsGroupAbout)
	mustHave(t, render(a), "未设置同步文件夹")
}

// TestSyncFolderEmptyDir 设置了一个空目录时说明「还没有配置包」。
func TestSyncFolderEmptyDir(t *testing.T) {
	a := newTestApp()
	a.settings.SyncFolder = t.TempDir()
	a.refreshSyncBundles()
	a.page = settingsPageID
	a.setSettingsGroup(settingsGroupAbout)
	mustHave(t, render(a), "文件夹里还没有配置包")
}

// TestProfileTextHelpers 说明文案与大小格式。
func TestProfileTextHelpers(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"settings"}, "仅设置"},
		{[]string{"settings", "data"}, "设置 + 数据"},
		{[]string{"settings", "data", "background"}, "设置 + 数据 + 背景图"},
	}
	for _, c := range cases {
		if got := includesText(c.in); got != c.want {
			t.Errorf("includesText(%v) = %q，期望 %q", c.in, got, c.want)
		}
	}
	if got := formatBytes(900); got != "900 B" {
		t.Errorf("formatBytes(900) = %q", got)
	}
	if got := formatBytes(2 * 1024 * 1024); got != "2.0 MB" {
		t.Errorf("formatBytes(2MB) = %q", got)
	}
}

// TestProfileSettingsDefaults 新增的桌宠/同步字段有合理默认值。
func TestProfileSettingsDefaults(t *testing.T) {
	s := settings.Default()
	if s.Tone != settings.ToneDefault || s.PetSkin != settings.PetSkinCat {
		t.Errorf("默认值不对: tone=%q skin=%q", s.Tone, s.PetSkin)
	}
	if s.PetReminderMinutes != settings.DefaultPetReminderMinutes {
		t.Errorf("默认提醒间隔 = %d", s.PetReminderMinutes)
	}
	if s.PetX != -1 || s.PetY != -1 {
		t.Errorf("桌宠位置哨兵应为 -1: %d,%d", s.PetX, s.PetY)
	}
}
