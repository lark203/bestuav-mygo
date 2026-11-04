package view

import (
	"strings"
	"testing"

	"bestuav-mygo/internal/profile"
	"bestuav-mygo/internal/settings"
)

// TestRepoSection 设置页有「仓库同步」区块与三个平台。
func TestRepoSection(t *testing.T) {
	a := newTestApp()
	a.page = settingsPageID
	a.setSettingsGroup(settingsGroupAbout)
	mustHave(t, render(a), "仓库同步", "平台", "实例地址", "所有者 / 仓库",
		"分支 / 路径", "访问令牌", "测试连接", "上传当前配置", "从仓库下载并导入")
}

// TestRepoDefaultPath 默认路径带设备名，避免多台机器互相覆盖。
func TestRepoDefaultPath(t *testing.T) {
	p := repoDefaultPath()
	if !strings.HasPrefix(p, "profiles/") || !strings.HasSuffix(p, ".mygo-profile.zip") {
		t.Errorf("默认路径不对: %q", p)
	}
	if !strings.Contains(p, strings.ReplaceAll(profile.DeviceName(), " ", "-")) {
		t.Errorf("默认路径应含设备名: %q", p)
	}
}

// TestRepoConfigDefaults 默认平台 GitHub、分支 main、路径按设备名生成。
func TestRepoConfigDefaults(t *testing.T) {
	a := newApp(settings.Default())
	cfg := a.repoConfig()
	if cfg.Kind != "github" || cfg.Branch != "main" || cfg.Path == "" {
		t.Errorf("默认仓库配置不对: %+v", cfg)
	}
	if cfg.Owner != "" || cfg.Token != "" {
		t.Error("默认不应带所有者/令牌")
	}
}
