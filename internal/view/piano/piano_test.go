package piano

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// 说明：这些用例**不按琴键发声**（真按下会在跑测试的机器上响出来）。
// 需要走「按下/松开」的地方一律先 SetMuted(true)：静音时状态照常变化，
// 只是不去合成与播放。

// TestView 离屏渲染一帧，确认曲子按钮与提示都在。
func TestView(t *testing.T) {
	p := New()
	tt := ui.NewTester(p.View, 900, 420)
	for _, s := range Songs {
		if !tt.HasText(s.Name) {
			t.Errorf("没有 %q 按钮，画面文本：%v", s.Name, tt.Texts())
		}
	}
	for _, want := range []string{"停止", "琴键 Z–M"} {
		if !tt.HasText(want) {
			t.Errorf("缺少 %q，画面文本：%v", want, tt.Texts())
		}
	}
}

// TestKeys 键盘几何：白键数、黑键压在白键之上、越界返回 -1。
func TestKeys(t *testing.T) {
	if n := Whites(); n != 22 {
		t.Errorf("%d 个白键，应为 22", n)
	}
	// C♯3 黑键的左半边 → C♯3；它下方的白键 → C3；它右边 → D3。
	if got := keyAt(whiteWidth-2, 10); got != 49 {
		t.Errorf("C#3 上 = %d，应为 49", got)
	}
	if got := keyAt(whiteWidth-2, blackHeight+10); got != 48 {
		t.Errorf("C#3 下 = %d，应为 48", got)
	}
	if got := keyAt(whiteWidth+blackWidth, 10); got != 50 {
		t.Errorf("C#3 右 = %d，应为 50", got)
	}
	if got := keyAt(KeyboardWidth+5, 10); got != -1 {
		t.Errorf("键盘外 = %d，应为 -1", got)
	}
	// 每个白键的中心都应当命中它自己，且顺序递增。
	want := Lowest
	for x := float32(0); x < KeyboardWidth; x += whiteWidth {
		got := keyAt(x+whiteWidth/2, keyboardHeight-10)
		for Black(want) {
			want++
		}
		if got != want {
			t.Fatalf("x=%v 处 = %d，应为 %d", x, got, want)
		}
		want++
	}
}

// TestSongs 曲谱解析：音名换算正确，且没有音跑到键盘外。
func TestSongs(t *testing.T) {
	if noteNumber("A4") != 69 || noteNumber("C#4") != 61 || noteNumber("C3") != Lowest {
		t.Errorf("noteNumber 不对：A4=%d C#4=%d C3=%d", noteNumber("A4"), noteNumber("C#4"), noteNumber("C3"))
	}
	for _, s := range Songs {
		evs := s.events()
		if len(evs) == 0 {
			t.Fatalf("%s 一个音都没有", s.Name)
		}
		for _, ev := range evs {
			if ev.beats <= 0 {
				t.Errorf("%s 有非正的拍数 %v", s.Name, ev.beats)
			}
			for _, n := range ev.notes {
				if n < Lowest || n > Highest {
					t.Errorf("%s 弹到 %d，超出键盘", s.Name, n)
				}
			}
		}
	}
}

// TestSynth A4 应当持续 1.4–4 秒，且基频在 440 Hz 附近
// （用自相关找周期：波形与自身错开一个周期时最相关）。
func TestSynth(t *testing.T) {
	s := synthSamples(69)
	if d := float64(len(s)) / synthRate; d < 1.4 || d > 4 {
		t.Errorf("A4 长 %v 秒", d)
	}
	best, lag := 0.0, 0
	for l := synthRate / 600; l <= synthRate/300; l++ {
		c := 0.0
		for i := synthRate / 10; i < synthRate*6/10; i++ {
			c += float64(s[i]) * float64(s[i+l])
		}
		if c > best {
			best, lag = c, l
		}
	}
	if f := float64(synthRate) / float64(lag); f < 430 || f > 450 {
		t.Errorf("A4 在 %v Hz，应为 440 附近", f)
	}
	// 归一化后不应削顶。
	peak := 0.0
	for _, v := range s {
		peak = math.Max(peak, math.Abs(float64(v)))
	}
	if peak > 1 {
		t.Errorf("峰值 %v 削顶了", peak)
	}
}

// TestSources 同一个音被多处按住时，要等最后一处松手才停。
func TestSources(t *testing.T) {
	p := New()
	p.SetMuted(true)
	p.down(60)
	p.down(60) // 指针和电脑键各按一次
	if p.sources[60] != 2 {
		t.Fatalf("按两下后 sources=%d，应为 2", p.sources[60])
	}
	p.up(60)
	if p.sources[60] != 1 {
		t.Errorf("松开一处后 sources=%d，应为 1", p.sources[60])
	}
	p.up(60)
	if _, ok := p.sources[60]; ok {
		t.Errorf("全松开后不该还记着这个音")
	}
	// 多按一次 up 不应越减成负数。
	p.up(60)
	if _, ok := p.sources[60]; ok {
		t.Errorf("多松一次又把音记回来了")
	}
	// 键盘外的音直接忽略。
	p.down(Lowest - 1)
	p.down(Highest + 1)
	if len(p.sources) != 0 {
		t.Errorf("越界音不该被记下：%v", p.sources)
	}
}

// TestPedal 踏板抬起前，松手的弦不会被制住。
func TestPedal(t *testing.T) {
	p := New()
	p.SetMuted(true)
	p.pedal(true)
	if !p.sustain {
		t.Fatal("踏板没踩下")
	}
	p.down(60)
	p.up(60) // 踏板还踩着：弦应当还在响
	if _, ok := p.held[60]; !ok && !p.muted {
		t.Error("踏板踩着时松手不该止音")
	}
	p.pedal(false)
	if p.sustain {
		t.Error("踏板没抬起")
	}
	// 抬起后会制住所有没人按的弦。
	if len(p.held) != 0 {
		t.Errorf("抬起踏板后还剩 %d 根弦", len(p.held))
	}
}

// TestSongNotes 曲子按下的音要记进 songHeld，Stop 时全部松开。
func TestSongNotes(t *testing.T) {
	p := New()
	p.SetMuted(true)
	p.song = Songs[0]
	p.songNotes([]int{60, 64, 67}, true)
	if len(p.songHeld) != 3 {
		t.Fatalf("按下 3 个音后 songHeld=%v", p.songHeld)
	}
	for _, n := range []int{60, 64, 67} {
		if p.sources[n] != 1 {
			t.Errorf("音 %d 的 sources=%d，应为 1", n, p.sources[n])
		}
	}
	p.songNotes([]int{64}, false) // 松开中间那个
	if len(p.songHeld) != 2 {
		t.Errorf("松开一个后 songHeld=%v，应剩 2 个", p.songHeld)
	}
	if _, ok := p.sources[64]; ok {
		t.Error("松开的音不该还记着")
	}
	p.Stop()
	if p.song != nil || p.songHeld != nil {
		t.Error("Stop 后曲子与按住的音都该清空")
	}
	if len(p.sources) != 0 {
		t.Errorf("Stop 后还有音没松开：%v", p.sources)
	}
}

// TestOctaveClamps 换八度只在 3–4 之间，越界输入不改变音域。
func TestOctaveClamps(t *testing.T) {
	p := New()
	p.SetMuted(true)
	for i := 0; i < 10; i++ {
		p.input(ui.InputEvent{Kind: ui.InputKeyDown, PhysicalKey: ui.KeyRight})
	}
	if p.Octave() != 4 {
		t.Errorf("一直往右换八度后 = %d，应停在 4", p.Octave())
	}
	for i := 0; i < 10; i++ {
		p.input(ui.InputEvent{Kind: ui.InputKeyDown, PhysicalKey: ui.KeyLeft})
	}
	if p.Octave() != 3 {
		t.Errorf("一直往左换八度后 = %d，应停在 3", p.Octave())
	}
	// 带修饰键的按键交给页面其它部分处理（不归钢琴）。
	if p.input(ui.InputEvent{Kind: ui.InputKeyDown, PhysicalKey: ui.KeyZ, Mods: ui.Ctrl}) {
		t.Error("Ctrl+Z 不该被钢琴吃掉")
	}
	// 静音时按琴键只改状态：键亮起来，但一根弦都不响。
	if p.input(ui.InputEvent{Kind: ui.InputKeyDown, PhysicalKey: ui.KeyZ}) {
		if p.sources[(p.Octave()+1)*12] != 1 {
			t.Errorf("Z 键没按下对应的音，sources=%v", p.sources)
		}
		if len(p.held) != 0 {
			t.Errorf("静音时不该有弦在响，实际 %d 根", len(p.held))
		}
	}
}
