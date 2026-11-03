package piano

import (
	"math"
	"math/rand/v2"
	"sync"

	"github.com/egoist/mygo/audio"
)

// synthRate 是合成用的采样率：够高到泛音不混叠，够低到算得快。
const synthRate = 32000

// sounds 是每个琴键合成结果的缓存，第一次弹到才算。
var sounds [Highest + 1]struct {
	once sync.Once
	s    *audio.Sound
}

// sound 取 note 这个键的声音，第一次调用时合成。
func sound(note int) *audio.Sound {
	e := &sounds[note]
	e.once.Do(func() { e.s = audio.NewSound(synthRate, 1, synthSamples(note)) })
	return e.s
}

// synthSamples 合成一个像钢琴的音色：泛音比整数倍略高（像刚性弦），
// 两根相差一音分的弦同时发声，先快后慢地衰减（音越高越快），
// 再叠上琴槌敲击的一记短噪声与 2 ms 的起振。
func synthSamples(note int) []float32 {
	f := 440 * math.Pow(2, float64(note-69)/12)
	// 低音弦余韵更长。
	dur := 1.4 + 2.6*float64(Highest-note)/float64(Highest-Lowest)
	n := int(dur * synthRate)
	out := make([]float64, n)
	stiffness := 0.00008 * math.Pow(2, float64(note-Lowest)/12)
	for k := 1; k <= 18; k++ {
		fk := float64(k) * f * math.Sqrt(1+stiffness*float64(k*k))
		if fk > synthRate*0.45 {
			break
		}
		amp := 1 / math.Pow(float64(k), 1.1)
		fast := 0.3 * math.Pow(261.6/f, 0.5) / (1 + 0.3*float64(k-1))
		slow := fast * 7
		for _, detune := range []float64{1, 1.0006} {
			// 用旋转代替逐点 sin，用逐点乘代替 exp：18 个泛音 × 十几万采样，
			// 每个采样都调 sin/exp 会明显变慢。
			w := 2 * math.Pi * fk * detune / synthRate
			cw, sw := math.Cos(w), math.Sin(w)
			s, c := 0.0, 1.0
			e1, e2 := 0.65, 0.35
			d1, d2 := math.Exp(-1/(fast*synthRate)), math.Exp(-1/(slow*synthRate))
			for i := range out {
				out[i] += amp * (e1 + e2) * s
				s, c = s*cw+c*sw, c*cw-s*sw
				e1 *= d1
				e2 *= d2
			}
		}
	}
	// 琴槌：一小段低通噪声，加 2 ms 的起振。
	rng := rand.New(rand.NewPCG(uint64(note), 1))
	lp := 0.0
	for i := range min(n, synthRate/20) {
		t := float64(i) / synthRate
		lp += 0.3 * (rng.Float64()*2 - 1 - lp)
		out[i] += 0.4 * lp * math.Exp(-t/0.006)
		out[i] *= min(t/0.002, 1)
	}
	// 末尾 50 ms 淡出，并把每个键的响度归一化到一致。
	peak := 0.0
	for i, v := range out {
		if r := n - i; r < synthRate/20 {
			out[i] *= float64(r) / (synthRate / 20)
		}
		peak = max(peak, math.Abs(v))
	}
	samples := make([]float32, n)
	for i, v := range out {
		samples[i] = float32(0.22 * v / peak)
	}
	return samples
}
