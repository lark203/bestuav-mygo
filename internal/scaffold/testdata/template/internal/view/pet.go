package view

import (
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/pet"
	"bestuav-mygo/internal/settings"
)

// 伴生桌宠：一个无边框、透明、置顶、不占任务栏的小窗口，只画一只简笔小猫/小狗。
//
// 低占用约束（实现时不许破坏）：
//   - 没有持续动画循环：静止时不请求帧，靠 c.After(20s) 做一次低频「心跳」；
//   - 只有矢量元素（无图片/无贴图），窗口一百多像素见方；
//   - 隐藏/关闭窗口后不残留定时器（窗口销毁即停）。

// petBaseSize 幼崽阶段的窗口逻辑尺寸（按阶段缩放）。
const (
	petBaseW = 150
	petBaseH = 210
)

// petSize 当前阶段对应的窗口尺寸。
func petSize(stage int) (int, int) {
	s := pet.Scale(stage)
	return int(float64(petBaseW) * s), int(float64(petBaseH) * s)
}

// petTick 累积陪伴时长、驱动久坐提醒、同步窗口。
//
// 主窗口每帧与桌宠窗口每 20s 都会调用；delta 以 petTickAt 为基准，重复调用不会多算。
func (a *App) petTick(now time.Time) {
	a.petSyncWindow()
	if !a.settings.PetEnabled {
		return
	}
	if a.petTickAt.IsZero() {
		a.petTickAt = now
		if a.petLastBreak.IsZero() {
			a.petLastBreak = now
		}
	}
	delta := now.Sub(a.petTickAt)
	a.petTickAt = now
	// 睡眠/长时间挂起后不把整段空闲算成陪伴时长（心跳是 20 秒，5 分钟已很宽松）。
	if delta > 0 && delta < 5*time.Minute {
		xp, stage := pet.AddXP(a.petXP, delta.Minutes())
		a.petXP = xp
		if stage != a.petStage {
			a.petStage = stage
			a.petResize()
			a.toast("桌宠长大了一点：" + pet.StageName(stage))
		}
		if a.petXP-a.petSavedXP >= 1 {
			a.petSavedXP = a.petXP
			a.settings.PetXP = a.petXP
			a.persist()
		}
	}
	// 位置：窗口移动没有事件，1 秒轮询一次（比每帧读平台窗口便宜）。
	if a.petWin != nil && now.Sub(a.petPollAt) >= time.Second {
		a.petPollAt = now
		x, y := a.petWin.Position()
		if x != a.settings.PetX || y != a.settings.PetY {
			a.settings.PetX, a.settings.PetY = x, y
			a.persist()
		}
	}
	if pet.ReminderDue(a.petLastBreak, now, a.settings.PetReminderMinutes) {
		a.petLastBreak = now
		a.petRemind()
	}
}

// petSyncWindow 根据开关创建/销毁桌宠窗口。
func (a *App) petSyncWindow() {
	if a.settings.PetEnabled == a.petApplied {
		return
	}
	// 主窗口还没就绪（启动早期 / 测试）：下一帧再试，不要把「已应用」提前置位。
	if a.settings.PetEnabled && a.win == nil {
		return
	}
	a.petApplied = a.settings.PetEnabled
	if a.settings.PetEnabled {
		a.openPetWindow()
	} else {
		a.closePetWindow()
	}
	a.persist()
}

// openPetWindow 创建桌宠窗口（需要主窗口已就绪；托盘模式下也允许）。
func (a *App) openPetWindow() {
	if a.petWin != nil || a.win == nil {
		return
	}
	w, h := petSize(a.petStage)
	opts := mygo.WindowOptions{
		Title:             "桌宠",
		Width:             w,
		Height:            h,
		Frameless:         true,
		Transparent:       true,
		AlwaysOnTop:       true,
		SkipTaskbar:       true,
		DisableResize:     true,
		DisableMaximize:   true,
		DisableMinimize:   true,
		DisableFullScreen: true,
		DisableShadow:     true,
		// 桌宠没有标题栏也没有关闭按钮；键盘/系统关闭也禁掉，避免误关后
		// 设置里仍显示「已开启」。
		DisableClose: true,
		Content:      ui.View(a.petView),
	}
	if a.settings.PetX >= 0 && a.settings.PetY >= 0 {
		opts.X, opts.Y = a.settings.PetX, a.settings.PetY
	}
	win := mygo.NewWindow(opts)
	if win == nil {
		a.settings.PetEnabled = false
		a.petApplied = false
		a.toast("桌宠窗口创建失败")
		return
	}
	a.petWin = win
	win.OnClosed(func() {
		// 只有应用退出或设置里关掉桌宠时才会走到这里；不改设置（退出时改会丢开关）。
		a.petWin = nil
		a.petApplied = false
		a.requestFrame()
	})
	a.persist()
}

// closePetWindow 关闭桌宠窗口。
func (a *App) closePetWindow() {
	if a.petWin == nil {
		return
	}
	a.petWin.Close()
	a.petWin = nil
}

// petResize 阶段变化后调整窗口大小（保持当前位置）。
func (a *App) petResize() {
	if a.petWin == nil {
		return
	}
	w, h := petSize(a.petStage)
	a.petWin.SetSize(w, h)
}

// petView 桌宠窗口的内容：一只简笔宠物 + 可选气泡。
func (a *App) petView(c *ui.Context) {
	now := time.Now()
	a.petTick(now)
	t := c.Theme()

	bubble := now.Before(a.petBubbleUntil)
	// 低频心跳：每 20 秒重绘一次，用来累计经验与检查久坐提醒——不是动画循环。
	c.After(20 * time.Second)
	if bubble {
		// 气泡消失时再重绘一帧，不必靠轮询。
		c.After(time.Until(a.petBubbleUntil))
	}

	root := ui.Box(c).Fill().DragWindow().Tooltip("单击摸摸头 · 双击打开主窗口")
	if root.Clicked() {
		a.petPoke()
	}
	if root.DoubleClicked() {
		a.petOpenMain()
	}
	root.Children(func() {
		ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			if bubble {
				ui.Box(c).Radius(10).Padding(8, 5).Background(t.Accent).
					Children(func() {
						ui.Text(c, a.petBubbleText).FontSize(11).TextColor(t.AccentText).
							SingleLine()
					})
			}
			a.drawPet(c, t, float32(pet.Scale(a.petStage)))
		})
	})
}

// drawPet 用矢量元素拼一只小猫/小狗；颜色固定（桌面透出时也看得清）。
func (a *App) drawPet(c *ui.Context, t *ui.Theme, scale float32) {
	skin := pet.NormalizeSkin(a.settings.PetSkin)
	cat := skin == settings.PetSkinCat

	fur, furDark, belly, dark, inner := "#f0b45f", "#d99a45", "#fbe3c0", "#3a2b1c", "#f0a5a5"
	if !cat {
		fur, furDark, belly, dark, inner = "#b98a5f", "#9a6f47", "#efdcc4", "#33261a", "#d99b8a"
	}
	head := 58 * scale
	eye := 7 * scale
	nose := 7 * scale

	// 耳朵：猫是两只菱形（转 45°），狗是两只垂耳。
	ui.Row(c).Gap(head * 0.42).Children(func() {
		if cat {
			ui.Box(c).Width(18*scale).Height(18*scale).Radius(4*scale).
				Rotate(45).Background(ui.Hex(furDark)).Margin(0, 0, -6*scale, 0)
			ui.Box(c).Width(18*scale).Height(18*scale).Radius(4*scale).
				Rotate(45).Background(ui.Hex(furDark)).Margin(0, 0, -6*scale, 0)
		} else {
			ui.Box(c).Width(13*scale).Height(26*scale).Radius(7*scale).
				Rotate(40).Background(ui.Hex(furDark)).Margin(0, 0, -14*scale, 0)
			ui.Box(c).Width(13*scale).Height(26*scale).Radius(7*scale).
				Rotate(-40).Background(ui.Hex(furDark)).Margin(0, 0, -14*scale, 0)
		}
	})

	// 头：圆角方块（Radius 大于半径即圆），里面是眼睛与鼻子。
	ui.Box(c).Width(head).Height(head*0.94).Radius(head).Background(ui.Hex(fur)).
		Border(1.5, ui.Hex(furDark)).AlignItems(ui.Center).Justify(ui.Center).
		Children(func() {
			ui.Row(c).Gap(head * 0.2).Children(func() {
				ui.Box(c).Width(eye).Height(eye * 1.2).Radius(eye).Background(ui.Hex(dark))
				ui.Box(c).Width(eye).Height(eye * 1.2).Radius(eye).Background(ui.Hex(dark))
			})
			ui.Box(c).Width(nose).Height(nose*0.7).Radius(nose).Background(ui.Hex(inner)).
				Margin(4*scale, 0, 0, 0)
		})

	// 项圈：跟着界面强调色，和主题有一点呼应。
	ui.Box(c).Width(head*0.55).Height(7*scale).Radius(4*scale).
		Background(t.Accent).Margin(-2*scale, 0, 0, 0)

	// 身体 + 肚子 + 四只脚。
	ui.Box(c).Width(head*1.1).Height(head*0.9).Radius(head).
		Background(ui.Hex(fur)).
		Border(1.5, ui.Hex(furDark)).AlignItems(ui.Center).Justify(ui.Center).
		Children(func() {
			ui.Box(c).Width(head*0.6).Height(head*0.5).Radius(head).
				Background(ui.Hex(belly)).Margin(0, 0, 0, 0)
		})
	ui.Row(c).Gap(head*0.35).Margin(-6*scale, 0, 0, 0).Children(func() {
		for i := 0; i < 2; i++ {
			ui.Box(c).Width(16 * scale).Height(10 * scale).Radius(6 * scale).
				Background(ui.Hex(furDark))
		}
	})
}

// petPoke 单击宠物：有气泡时当作「知道了」，否则冒个泡回应。
func (a *App) petPoke() {
	now := time.Now()
	if now.Before(a.petBubbleUntil) {
		a.petBubbleUntil = time.Time{}
		a.petLastBreak = now
		a.petBubbleText = ""
	} else {
		if pet.NormalizeSkin(a.settings.PetSkin) == settings.PetSkinDog {
			a.petBubbleText = "汪～"
		} else {
			a.petBubbleText = "喵～"
		}
		a.petBubbleUntil = now.Add(1200 * time.Millisecond)
	}
	if a.petWin != nil {
		a.petWin.Invalidate()
	}
	a.requestFrame()
}

// petOpenMain 双击宠物：唤回主窗口。
func (a *App) petOpenMain() {
	if a.win == nil {
		return
	}
	a.win.Show()
	a.win.Focus()
}

// petRemind 久坐提醒：气泡 + （主窗口不可见时）系统通知。
func (a *App) petRemind() {
	a.petBubbleText = "起来活动一下～"
	a.petBubbleUntil = time.Now().Add(15 * time.Second)
	if a.petWin != nil {
		a.petWin.Invalidate()
	}
	if a.win != nil && !a.win.IsVisible() && mygo.NotificationsSupported() {
		n := mygo.NewNotification(mygo.NotificationOptions{
			ID:    "pet-break",
			Title: "该休息一下了",
			Body:  fmt.Sprintf("已经坐了 %d 分钟，起来走走、看看远处。", a.settings.PetReminderMinutes),
		})
		_ = n.Show()
	}
	a.requestFrame()
}

// petSection 设置页「桌宠」区块。
func (a *App) petSection(c *ui.Context, t *ui.Theme) {
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Switch(c, &a.settings.PetEnabled).Shrink(0)
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, "显示桌宠").Bold()
			ui.Text(c, "无边框、透明、置顶的小窗，不占任务栏；静止时不重绘，只有矢量图形。").
				TextColor(t.TextMuted).FontSize(t.FontSize - 2)
		})
	})

	// 皮肤：两档。
	skinIdx := 0
	if pet.NormalizeSkin(a.settings.PetSkin) == settings.PetSkinDog {
		skinIdx = 1
	}
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "皮肤").Width(72).Shrink(0).TextColor(t.TextMuted)
		ui.Segmented(c, &skinIdx, "小猫", "小狗")
	})
	if got := []string{settings.PetSkinCat, settings.PetSkinDog}[skinIdx]; got != a.settings.PetSkin {
		a.settings.PetSkin = got
		a.persist()
		if a.petWin != nil {
			a.petWin.Invalidate()
		}
	}

	// 久坐提醒：0 = 关闭。
	remind := float64(a.settings.PetReminderMinutes)
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "久坐提醒").Width(72).Shrink(0).TextColor(t.TextMuted)
		ui.Stepper(c, &remind, 0, settings.MaxPetReminderMinutes, 5)
		hint := fmt.Sprintf("每 %d 分钟提醒一次", int(remind))
		if int(remind) == 0 {
			hint = "关闭提醒"
		}
		ui.Text(c, hint).TextColor(t.TextMuted).FontSize(t.FontSize - 2)
	})
	if got := int(remind); got != a.settings.PetReminderMinutes {
		a.settings.PetReminderMinutes = got
		a.persist()
	}

	// 成长：只展示，不提供「一键喂大」——陪伴时长才有意义。
	stage := pet.Stage(a.petXP)
	next, ok := pet.NextThreshold(stage)
	progress := pet.Progress(a.petXP, stage)
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "成长").Width(72).Shrink(0).TextColor(t.TextMuted)
		ui.Text(c, pet.StageName(stage)).Bold()
		// 进度条：不用 Meter（它的参数是值域，不是 0–1），两段 Row 更直观。
		ui.Row(c).Gap(0).Width(160).Height(6).Children(func() {
			if progress > 0 {
				ui.Box(c).Width(160 * float32(progress)).Height(6).Radius(3).Background(t.Accent)
			}
			if progress < 1 {
				ui.Box(c).Width(160 * float32(1-progress)).Height(6).Radius(3).
					Background(t.SurfacePressed)
			}
		})
		if ok {
			ui.Text(c, fmt.Sprintf("%d / %d 分钟", int(a.petXP), int(next))).
				TextColor(t.TextMuted).FontSize(t.FontSize - 2)
		} else {
			ui.Text(c, "已成年（不再长大）").TextColor(t.TextMuted).FontSize(t.FontSize - 2)
		}
	})
	ui.Text(c, "成长只看陪伴时长（前台每 20 秒结算一次），换机器后随之同步；"+
		"关掉桌宠不会丢进度。").
		TextColor(t.TextMuted).FontSize(t.FontSize - 2)

	ui.Row(c).Children(func() {
		if ui.Button(c, "回到屏幕中央").Clicked() && a.petWin != nil {
			a.petWin.Center()
		}
	})
}
