package view

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/pet"
	"bestuav-mygo/internal/settings"
)

// TestPetSection 设置页有「桌宠」区块与全部入口。
func TestPetSection(t *testing.T) {
	a := newTestApp()
	a.page = settingsPageID
	mustHave(t, render(a), "桌宠", "显示桌宠", "皮肤", "久坐提醒", "成长", "回到屏幕中央")
}

// TestPetGrowthAccrues 陪伴时长按 tick 间隔累计，跨门槛时进入下一阶段。
func TestPetGrowthAccrues(t *testing.T) {
	a := newTestApp()
	a.settings.PetEnabled = true
	now := time.Now()
	a.petTick(now) // 只记基准，不累计
	if a.petXP != 0 {
		t.Fatalf("首次 tick 不应累计，实际 %v", a.petXP)
	}
	a.petTick(now.Add(2 * time.Minute))
	if a.petXP < 1.9 || a.petXP > 2.1 {
		t.Fatalf("2 分钟后应累计约 2 分钟，实际 %v", a.petXP)
	}
	// 睡眠/挂起造成的大间隔不应计入。
	a.petTick(now.Add(3 * time.Hour))
	if a.petXP > 3 {
		t.Fatalf("大间隔不应累计，实际 %v", a.petXP)
	}
}

// TestPetStageUp 到达门槛后升阶段，并把经验写回设置。
func TestPetStageUp(t *testing.T) {
	a := newTestApp()
	a.settings.PetEnabled = true
	a.petXP = 119.5
	a.petSavedXP = 119.5
	a.petStage = 0
	now := time.Now()
	a.petTick(now) // 基准
	a.petTick(now.Add(time.Minute))
	if a.petStage != 1 {
		t.Fatalf("跨过 120 分钟应升到少年，实际阶段 %d（xp=%v）", a.petStage, a.petXP)
	}
	if a.settings.PetXP <= 0 {
		t.Error("经验应写回设置")
	}
}

// TestPetReminder 久坐到点冒泡；关闭桌宠时不提醒。
func TestPetReminder(t *testing.T) {
	a := newTestApp()
	a.settings.PetEnabled = true
	a.settings.PetReminderMinutes = 45
	now := time.Now()
	a.petLastBreak = now.Add(-46 * time.Minute)
	a.petTick(now)
	if a.petBubbleText == "" || !a.petBubbleUntil.After(now) {
		t.Fatalf("应触发久坐提醒，text=%q until=%v", a.petBubbleText, a.petBubbleUntil)
	}
	if !a.petLastBreak.After(now.Add(-time.Minute)) {
		t.Error("提醒后应重置计时")
	}

	b := newTestApp()
	b.settings.PetEnabled = false
	b.petLastBreak = now.Add(-46 * time.Minute)
	b.petTick(now)
	if b.petBubbleText != "" {
		t.Error("关闭桌宠后不应提醒")
	}
}

// TestPetViewRenders 桌宠窗口内容能正常构建，气泡文字会画出来。
func TestPetViewRenders(t *testing.T) {
	for _, skin := range settings.PetSkins {
		a := newTestApp()
		a.settings.PetSkin = skin
		a.petBubbleText = "喵～"
		a.petBubbleUntil = time.Now().Add(time.Minute)
		tt := ui.NewTester(a.petView, 150, 210)
		tt.Frame()
		if !tt.HasText("喵～") {
			t.Errorf("%s 皮肤的气泡没有画出来", skin)
		}
	}
}

// TestPetNoWindowWithoutMain 主窗口未就绪时不创建桌宠（避免测试/启动早期误开窗）。
func TestPetNoWindowWithoutMain(t *testing.T) {
	a := newTestApp()
	a.settings.PetEnabled = true
	a.petTick(time.Now())
	if a.petWin != nil {
		t.Fatal("没有主窗口时不应创建桌宠窗口")
	}
	if a.petApplied {
		t.Fatal("窗口未就绪时不应把开关标记为已应用")
	}
	_ = pet.Stage(0)
}
