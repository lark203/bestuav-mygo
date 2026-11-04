package view

import (
	"testing"

	"bestuav-mygo/internal/settings"
)

// TestSettingsShowsProfileSection 设置页「数据与诊断」有导出/导入入口。
func TestSettingsShowsProfileSection(t *testing.T) {
	a := newTestApp()
	a.page = settingsPageID
	a.setSettingsGroup(settingsGroupAbout)
	mustHave(t, render(a), "配置与数据", "导出设置…", "导出设置与数据…", "导入…")
}

// TestProfileTextHelpers 大小格式。
func TestProfileTextHelpers(t *testing.T) {
	if got := formatBytes(900); got != "900 B" {
		t.Errorf("formatBytes(900) = %q", got)
	}
	if got := formatBytes(2 * 1024 * 1024); got != "2.0 MB" {
		t.Errorf("formatBytes(2MB) = %q", got)
	}
}

// TestProfileSettingsDefaults 新增字段有合理默认值。
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
