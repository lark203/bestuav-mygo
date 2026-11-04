package view

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/settings"
)

// TestApplyTone 每套色调都换背景/控件面/描边，且正文对比度仍然够。
func TestApplyTone(t *testing.T) {
	for _, def := range toneDefs[1:] { // 跳过「默认」
		t.Run(def.name, func(t *testing.T) {
			light := ui.LightTheme()
			baseSurface := light.Surface
			applyTone(light, def.name)
			if light.Background != def.light.bg || light.Surface != def.light.surface ||
				light.Border != def.light.border {
				t.Errorf("浅色色调未生效: bg=%v surface=%v", light.Background, light.Surface)
			}
			if light.Surface == baseSurface {
				t.Error("色调应改变控件面颜色")
			}
			if !contrastOK(light) {
				t.Errorf("浅色色调下正文对比度不足: text=%v bg=%v", light.Text, light.Background)
			}

			dark := ui.DarkTheme()
			applyTone(dark, def.name)
			if dark.Background != def.dark.bg || dark.Surface != def.dark.surface {
				t.Errorf("深色色调未生效: bg=%v surface=%v", dark.Background, dark.Surface)
			}
			if !dark.Dark {
				t.Error("深色主题标记不应被色调改掉")
			}
			if !contrastOK(dark) {
				t.Errorf("深色色调下正文对比度不足: text=%v bg=%v", dark.Text, dark.Background)
			}
		})
	}
}

// TestApplyToneDefaultKeepsTheme 「默认」色调不覆盖任何颜色。
func TestApplyToneDefaultKeepsTheme(t *testing.T) {
	th := ui.LightTheme()
	want := *th
	applyTone(th, settings.ToneDefault)
	if *th != want {
		t.Error("默认色调不应改变主题")
	}
}

// TestApplyToneIdempotent 每帧调用，重复应用结果一致。
func TestApplyToneIdempotent(t *testing.T) {
	th := ui.LightTheme()
	applyTone(th, settings.ToneGreen)
	first := *th
	applyTone(th, settings.ToneGreen)
	if *th != first {
		t.Error("重复应用色调不应改变主题")
	}
}

// TestToneNamesInSync 视图层与设置层的色调清单一致（防止一边加、一边漏）。
func TestToneNamesInSync(t *testing.T) {
	if len(toneDefs) != len(settings.Tones) {
		t.Fatalf("色调数量不一致: view=%d settings=%d", len(toneDefs), len(settings.Tones))
	}
	for i, def := range toneDefs {
		if settings.Tones[i] != def.name {
			t.Errorf("第 %d 套色调名不一致: %q vs %q", i, def.name, settings.Tones[i])
		}
		if toneIndex(def.name) != i || toneName(i) != def.name {
			t.Errorf("色调索引往返不一致: %q", def.name)
		}
	}
	if toneIndex("nope") != 0 {
		t.Error("未知色调应回落到默认")
	}
}

// TestTonePersists 改色调后写回 settings.json。
func TestTonePersists(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	a := newTestApp()
	tt := render(a)
	a.toneIdx = toneIndex(settings.ToneWarm)
	tt.Frame()
	if a.settings.Tone != settings.ToneWarm {
		t.Fatalf("应写回 %q，实际 %q", settings.ToneWarm, a.settings.Tone)
	}
	a.FlushSettings()
	s, err := settings.Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.Tone != settings.ToneWarm {
		t.Errorf("落盘后应读回 %q，实际 %q", settings.ToneWarm, s.Tone)
	}
}

// TestSettingsShowsTonePicker 设置页「外观」里有色调选择。
func TestSettingsShowsTonePicker(t *testing.T) {
	a := newTestApp()
	a.page = settingsPageID
	mustHave(t, render(a), "色调", "暖白", "护眼绿", "午夜蓝")
}

// contrastOK 正文与背景的感知亮度差足够大（WCAG 对比度简化为亮度差）。
func contrastOK(t *ui.Theme) bool {
	d := luma(t.Text) - luma(t.Background)
	if d < 0 {
		d = -d
	}
	return d >= 100
}
