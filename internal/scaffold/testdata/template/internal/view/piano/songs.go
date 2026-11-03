package piano

import (
	"strconv"
	"strings"
	"time"
)

// Song 是一首公有领域的曲子：音名写成「E5:0.5」（音名+八度 : 拍数），
// 和弦写成「C4+E4+G4:1」，休止写成「r:1」。
type Song struct {
	Name  string
	BPM   float64
	Notes string
}

// Songs 是可以自动演奏的曲子。
var Songs = []*Song{
	{"小星星", 100, `
		C4+C3:1 C4:1 G4+E3:1 G4:1 A4+F3:1 A4:1 G4+E3:2
		F4+D3:1 F4:1 E4+C3:1 E4:1 D4+G3:1 D4:1 C4+C3:2
		G4+E3:1 G4:1 F4+D3:1 F4:1 E4+C3:1 E4:1 D4+G3:2
		G4+E3:1 G4:1 F4+D3:1 F4:1 E4+C3:1 E4:1 D4+G3:2
		C4+C3:1 C4:1 G4+E3:1 G4:1 A4+F3:1 A4:1 G4+E3:2
		F4+D3:1 F4:1 E4+C3:1 E4:1 D4+G3:1 D4:1 C4+C3:2`},
	{"欢乐颂", 120, `
		E4+C3:1 E4:1 F4:1 G4:1 G4+G3:1 F4:1 E4:1 D4:1
		C4+A3:1 C4:1 D4:1 E4:1 E4+G3:1.5 D4:0.5 D4:2
		E4+C3:1 E4:1 F4:1 G4:1 G4+G3:1 F4:1 E4:1 D4:1
		C4+A3:1 C4:1 D4:1 E4:1 D4+G3:1.5 C4:0.5 C4+C3:2`},
	{"致爱丽丝", 66, `
		E5:0.25 D#5:0.25 E5:0.25 D#5:0.25 E5:0.25 B4:0.25 D5:0.25 C5:0.25
		A4+A3:0.5 r:0.25 C4:0.25 E4:0.25 A4:0.25
		B4+E3:0.5 r:0.25 E4:0.25 G#4:0.25 B4:0.25
		C5+A3:0.5 r:0.25 E4:0.25 E5:0.25 D#5:0.25
		E5:0.25 D#5:0.25 E5:0.25 B4:0.25 D5:0.25 C5:0.25
		A4+A3:0.5 r:0.25 C4:0.25 E4:0.25 A4:0.25
		B4+E3:0.5 r:0.25 E4:0.25 C5:0.25 B4:0.25
		A4+A3:1.5`},
	{"生日歌", 100, `
		G4:0.75 G4:0.25 A4+F3:1 G4:1 C5+E3:1 B4+G3:2
		G4:0.75 G4:0.25 A4+G3:1 G4:1 D5+F3:1 C5+E3:2
		G4:0.75 G4:0.25 G5+E3:1 E5:1 C5+A3:1 B4:1 A4+F3:1
		F5:0.75 F5:0.25 E5+C3:1 C5:1 D5+G3:1 C5+C3:2`},
}

// event 是曲子里的一拍：同时按下的几个音，持续多少拍。
type event struct {
	notes []int
	beats float64
}

// events 把曲子的文本解析成一串事件。写错的拍数或音名会 panic ——
// 曲子是写死在源码里的，错了就应当在启动时暴露，而不是静默跑歪。
func (s *Song) events() []event {
	var evs []event
	for _, tok := range strings.Fields(s.Notes) {
		name, length, _ := strings.Cut(tok, ":")
		beats, err := strconv.ParseFloat(length, 64)
		if err != nil {
			panic("piano: 拍数写错了：" + tok)
		}
		var notes []int
		if name != "r" {
			for _, n := range strings.Split(name, "+") {
				notes = append(notes, noteNumber(n))
			}
		}
		evs = append(evs, event{notes, beats})
	}
	return evs
}

// noteNumber 把 C4、G#3 这样的音名换成 MIDI 音高编号。
func noteNumber(name string) int {
	semi := map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}[name[0]]
	rest := name[1:]
	if strings.HasPrefix(rest, "#") {
		semi++
		rest = rest[1:]
	}
	octave, err := strconv.Atoi(rest)
	if err != nil {
		panic("piano: 音名写错了：" + name)
	}
	return (octave+1)*12 + semi
}

// Play 演奏 s：每个事件都从曲子开头起算时间，这样速度不会被累计误差拖慢。
func (p *Piano) Play(s *Song) {
	p.Stop()
	p.song = s
	go func() {
		beat := time.Duration(float64(time.Minute) / s.BPM)
		start := time.Now()
		at := time.Duration(0)
		for _, ev := range s.events() {
			length := time.Duration(ev.beats * float64(beat))
			if !p.at(s, start.Add(at), func() { p.songNotes(ev.notes, true) }) {
				return
			}
			// 像真人那样：按住到这一拍的九成才松手。
			if !p.at(s, start.Add(at+length*9/10), func() { p.songNotes(ev.notes, false) }) {
				return
			}
			at += length
		}
		p.onMain(func() {
			if p.song == s {
				p.song = nil
			}
		})
	}()
}

// at 在主线程于时刻 t 跑 fn；如果这期间曲子换了（被停止或改放别的），就不跑。
func (p *Piano) at(s *Song, t time.Time, fn func()) bool {
	time.Sleep(time.Until(t))
	done := make(chan bool, 1)
	p.onMain(func() {
		if p.song != s {
			done <- false
			return
		}
		fn()
		p.redrawNow()
		done <- true
	})
	return <-done
}

// songNotes 按下或松开曲子当前这一拍的音。
func (p *Piano) songNotes(notes []int, down bool) {
	for _, n := range notes {
		if down {
			p.down(n)
			p.songHeld = append(p.songHeld, n)
		} else {
			p.up(n)
			for i, h := range p.songHeld {
				if h == n {
					p.songHeld = append(p.songHeld[:i], p.songHeld[i+1:]...)
					break
				}
			}
		}
	}
}

// Stop 停掉正在放的曲子，并松开它按着的键。
func (p *Piano) Stop() {
	p.song = nil
	for _, n := range p.songHeld {
		p.up(n)
	}
	p.songHeld = nil
}
