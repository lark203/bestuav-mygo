package view

import (
	"os"
	"strings"
	"testing"

	"bestuav-mygo/internal/settings"
)

// TestSettingsSaveDebounced 钉住「改设置不立刻写盘、防抖后才写、退出前能补写」。
//
// 背景：滑块拖动时每个像素都会 persist 一次，直接 Save 会在一秒内写几十次
// 临时文件 + rename（机械盘/杀软上能看出卡顿）。这里用临时配置目录验证：
// persist 之后文件还不存在，FlushSettings 之后才落盘。
func TestSettingsSaveDebounced(t *testing.T) {
	// 把 per-user 配置目录指到临时目录（Windows 读 AppData，Linux/macOS 读 XDG）。
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	a := newTestApp()
	path, err := settings.Path()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("测试开始时不应已有配置文件: %v", err)
	}

	a.settings.Theme = "dark"
	a.persist()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("persist 之后不应立即写盘（应等防抖窗口）")
	}

	a.FlushSettings()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FlushSettings 后应已写盘: %v", err)
	}
	if !strings.Contains(string(data), `"dark"`) {
		t.Errorf("落盘内容未包含新主题: %s", data)
	}

	// 没有待写内容时重复 flush 不应报错，也不应改写文件。
	a.FlushSettings()
}

// TestSettingsFlushOnQuitKeepsLastChange 覆盖「改完立刻退出」：FlushSettings 必须
// 把最后一次修改写进去（main 的 OnBeforeQuit 会调用它）。
func TestSettingsFlushOnQuitKeepsLastChange(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	a := newTestApp()
	a.settings.FontSize = 18
	a.persist()
	a.settings.FontSize = 19 // 同一批里的最后一次修改
	a.FlushSettings()

	path, _ := settings.Path()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"fontSize": 19`) {
		t.Errorf("应保存最后一次修改（19），实际: %s", data)
	}
}
