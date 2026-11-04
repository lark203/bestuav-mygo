package view

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
)

// TestApplyPalette 每套配色都能覆盖强调色系，且浅色/深色下都保持可读。
func TestApplyPalette(t *testing.T) {
	for _, def := range paletteDefs[1:] { // 跳过「跟随系统」
		t.Run(def.name, func(t *testing.T) {
			light := ui.LightTheme()
			applyPalette(light, def.name)
			if light.Accent != def.light {
				t.Errorf("浅色强调色 = %v，期望 %v", light.Accent, def.light)
			}
			if light.Selection.A == 0 || light.Focus.A == 0 {
				t.Error("选中/焦点色应跟着配色走，不能是透明的")
			}
			// 强调色上的文字只能是白或近黑，避免出现同色不可读。
			if !isWhiteOrNearBlack(light.AccentText) {
				t.Errorf("浅色 AccentText 不可用: %v", light.AccentText)
			}

			dark := ui.DarkTheme()
			applyPalette(dark, def.name)
			if dark.Accent != def.dark {
				t.Errorf("深色强调色 = %v，期望 %v", dark.Accent, def.dark)
			}
			// 深色主题里强调色要比浅色主题更亮，才能在深底上读出来。
			if luma(dark.Accent) <= luma(light.Accent) {
				t.Errorf("深色强调色应更亮: light=%d dark=%d", luma(light.Accent), luma(dark.Accent))
			}
			if !isWhiteOrNearBlack(dark.AccentText) {
				t.Errorf("深色 AccentText 不可用: %v", dark.AccentText)
			}
		})
	}
}

// TestApplyPaletteSystemKeepsTheme 「跟随系统」不覆盖任何颜色（保留桌面强调色）。
func TestApplyPaletteSystemKeepsTheme(t *testing.T) {
	th := ui.LightTheme()
	want := th.Accent
	applyPalette(th, settings.PaletteSystem)
	if th.Accent != want {
		t.Errorf("跟随系统不应覆盖强调色: %v → %v", want, th.Accent)
	}
}

// TestApplyPaletteIdempotent 每帧都会调用：重复应用结果必须一致（绝对赋值）。
func TestApplyPaletteIdempotent(t *testing.T) {
	th := ui.LightTheme()
	applyPalette(th, settings.PaletteTeal)
	first := *th
	applyPalette(th, settings.PaletteTeal)
	if *th != first {
		t.Error("重复应用配色不应改变主题")
	}
}

// TestPaletteNamesInSync 视图层与设置层的配色清单必须一致（防止一边加、一边漏）。
func TestPaletteNamesInSync(t *testing.T) {
	if len(paletteDefs) != len(settings.Palettes) {
		t.Fatalf("配色数量不一致: view=%d settings=%d", len(paletteDefs), len(settings.Palettes))
	}
	for i, def := range paletteDefs {
		if settings.Palettes[i] != def.name {
			t.Errorf("第 %d 套配色名不一致: %q vs %q", i, def.name, settings.Palettes[i])
		}
		if paletteIndex(def.name) != i || paletteName(i) != def.name {
			t.Errorf("配色索引往返不一致: %q", def.name)
		}
		if def.label == "" {
			t.Errorf("配色 %q 缺少展示名", def.name)
		}
	}
	if got := paletteIndex("no-such-palette"); got != 0 {
		t.Errorf("未知配色应回落到 0，实际 %d", got)
	}
}

// TestPalettePersists 在设置页改配色后写回 settings.json（重启仍生效）。
func TestPalettePersists(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	a := newTestApp()
	tt := render(a)
	a.paletteIdx = 2 // 青碧
	tt.Frame()
	if a.settings.Palette != settings.PaletteTeal {
		t.Fatalf("应写回 %q，实际 %q", settings.PaletteTeal, a.settings.Palette)
	}
	a.FlushSettings()

	s, err := settings.Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.Palette != settings.PaletteTeal {
		t.Errorf("落盘后应读回 %q，实际 %q", settings.PaletteTeal, s.Palette)
	}
}

// TestSettingsShowsPalettePicker 设置页「外观」里有配色选择。
func TestSettingsShowsPalettePicker(t *testing.T) {
	a := newTestApp()
	a.page = settingsPageID
	mustHave(t, render(a), "配色", "琥珀", "青碧", "靛蓝", "石墨")
}

// isWhiteOrNearBlack 强调色上的文字只允许白或近黑（与 mygo follow 的规则一致）。
func isWhiteOrNearBlack(c ui.Color) bool {
	return luma(c) > 230 || luma(c) < 60
}
