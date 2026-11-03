package shortcut

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestParseFormatRoundTrip 存储格式必须能往返：写进去什么，读出来还是什么。
func TestParseFormatRoundTrip(t *testing.T) {
	cases := []string{
		"cmd+1", "cmd+shift+k", "f5", "alt+left", "shift+space",
		"cmd+minus", "super+period", "a", "f12", "pagedown",
	}
	for _, want := range cases {
		b, err := Parse(want)
		if err != nil {
			t.Fatalf("Parse(%q): %v", want, err)
		}
		if b.String != want {
			t.Errorf("往返不一致: %q → %q", want, b.String)
		}
	}
}

// TestParseEmptyIsDisabled 空串是「禁用」，不是错误。
func TestParseEmptyIsDisabled(t *testing.T) {
	b, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	if !b.Zero() {
		t.Fatal("空串应解析为零值（禁用）")
	}
}

// TestParseRejectsGarbage 脏数据要报错（界面会原样展示）。
func TestParseRejectsGarbage(t *testing.T) {
	for _, s := range []string{"nope+2", "cmd+nosuchkey", "cmd+"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) 应报错", s)
		}
	}
}

// TestFormatUsesCmdForPlatformModifier 平台主修饰键统一写成 cmd。
func TestFormatUsesCmdForPlatformModifier(t *testing.T) {
	if got := Format(ui.Cmd, ui.Key2); got != "cmd+2" {
		t.Errorf("Format(Cmd, 2) = %q，期望 cmd+2", got)
	}
	// Cmd+Shift 组合不能丢 Shift。
	if got := Format(ui.Cmd|ui.Shift, ui.KeyK); got != "cmd+shift+k" {
		t.Errorf("Format(Cmd|Shift, K) = %q", got)
	}
}

// TestFromInputRejectsUnknown 不支持的键（如单纯的 KeyUnknown）不产生绑定。
func TestFromInputRejectsUnknown(t *testing.T) {
	if _, ok := FromInput(0, ui.KeyUnknown); ok {
		t.Error("KeyUnknown 不应产生绑定")
	}
}

// TestLabel 展示文本要可读。
func TestLabel(t *testing.T) {
	if got := Label("cmd+shift+k"); got == "" || got == "cmd+shift+k" {
		t.Errorf("Label 未转换成可读文本: %q", got)
	}
	if got := Label(""); got != "未设置" {
		t.Errorf("空绑定的标签应为「未设置」，实际 %q", got)
	}
	if got := Label("bogus+x"); got != "bogus+x" {
		t.Errorf("脏数据应原样展示，实际 %q", got)
	}
}

// TestResolve 用户覆盖优先，未知动作忽略。
func TestResolve(t *testing.T) {
	got := Resolve(map[string]string{
		"nav.task":     "cmd+9",
		"view.refresh": "",
		"no.such":      "cmd+8",
	})
	if got["nav.task"] != "cmd+9" {
		t.Errorf("覆盖未生效: %q", got["nav.task"])
	}
	if _, ok := got["no.such"]; ok {
		t.Error("未知动作不应出现在结果里")
	}
	if _, ok := got["view.refresh"]; !ok {
		t.Error("显式禁用（空串）也应保留在结果里")
	}
	// 未覆盖的用默认。
	if got["nav.home"] != "cmd+1" {
		t.Errorf("默认值丢了: %q", got["nav.home"])
	}
}

// TestDefaultsCoverAllActions 每个动作都要有默认键（否则设置页展示空白）。
func TestDefaultsCoverAllActions(t *testing.T) {
	d := Defaults()
	for _, a := range Actions {
		if a.Def == "" {
			continue
		}
		if _, err := Parse(d[a.ID]); err != nil {
			t.Errorf("动作 %s 的默认键位非法: %v", a.ID, err)
		}
	}
}
