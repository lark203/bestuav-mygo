package view

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
)

// ---- 官方示例：索引文本缓冲（大文档编辑）----
//
// 移植自 mygo examples/text-buffer（上游只有 36 行：造一个 10 万行的
// ui.TextBuffer 塞进 TextAreaBuffer）。本页把它做成能说明问题的演示：
//   - 可编辑：中间插入 / 改首行 / 删 100 行 / 撤销，并显示每次编辑的真实耗时；
//   - 可跳转：输入行号把光标移过去（SetTextSelection + LineRange）；
//   - 可实测：一次点击同时跑「rope 插入」与「整串拼接插入」，把倍数摆出来。
//
// 核心区别（也是 rope 这类结构存在的理由）：
//
//	string：改一个字符 = 复制整篇（4 MB 文档改 100 次 = 搬 400 MB）
//	TextBuffer：改一个字符 = 复制路径上的几个块（几微秒，与篇幅无关）

const (
	// 文档规模：与上游一致，10 万行、约 4 MB。
	tbLines = 100000
	// 撤销栈深度：快照共享未改动的块，20 步的代价可以忽略。
	tbUndoDepth = 20
)

// tbDemo 文档演示的状态：缓冲区、撤销栈、以及展示用的几段文字。
type tbDemo struct {
	buf    *ui.TextBuffer
	snaps  []ui.TextSnapshot
	jumpTo string // 行号输入框的内容
	note   string // 上一次操作的结果
	bench  string // 性能对比的结果（展示用）
	// 对比测到的单次耗时，单位是纳秒。用 float 是因为索引缓冲常常快到
	// 整数纳秒除不尽 —— 多轮平均后再取整会变成 0。
	lastRope, lastStr float64
	pending           int           // 待设置的光标位置（rune 下标；-1 表示没有）
	docCost           time.Duration // 建文档（编索引）的耗时
}

// newTbDemo 造一份 10 万行的示例文档，并记下建索引的耗时。
//
// 每行都带 emoji 与双向文字（阿拉伯语、希伯来语），这样字符数 / 字节数 /
// UTF-16 长度三者各不相同，状态行能直接看出「下标单位」这件事。
func newTbDemo() *tbDemo {
	start := time.Now()
	var b strings.Builder
	b.Grow(tbLines * 56)
	for i := range tbLines {
		fmt.Fprintf(&b, "Line %d: editable text, 😀, العربية, עברית\n", i+1)
	}
	// pending 从 0 开始：不清光标的话文本区会一进来就停在文档末尾，
	// 看起来像「已经滚到第 10 万行」，反而不像刚打开的文档。
	d := &tbDemo{buf: ui.NewTextBuffer(b.String()), pending: 0, docCost: time.Since(start)}
	d.note = fmt.Sprintf("已载入 %s 行，建索引耗时 %s", tbNum(d.buf.LineCount()), tbDur(d.docCost))
	return d
}

// pushUndo 编辑前压一个快照。快照不可变且共享未改动的块，所以很便宜。
func (d *tbDemo) pushUndo() {
	d.snaps = append(d.snaps, d.buf.Snapshot())
	if len(d.snaps) > tbUndoDepth {
		d.snaps = d.snaps[1:]
	}
}

// undo 回到上一个快照。Restore 是常数时间（换一个根指针）并推进版本号。
func (d *tbDemo) undo() {
	if len(d.snaps) == 0 {
		d.note = "没有可撤销的编辑"
		return
	}
	s := d.snaps[len(d.snaps)-1]
	d.snaps = d.snaps[:len(d.snaps)-1]
	d.buf.Restore(s)
	d.note = fmt.Sprintf("已撤销，回到版本 %d（还剩 %d 步）", d.buf.Version(), len(d.snaps))
}

// insertMiddle 在文档正中间插一行，并把光标移过去。
func (d *tbDemo) insertMiddle() {
	s := d.buf.Snapshot()
	line := s.LineCount() / 2
	a, _ := s.LineRange(line)
	d.pushUndo()
	start := time.Now()
	d.buf.Replace(a, a, ">> inserted in the middle <<\n")
	d.note = fmt.Sprintf("在第 %s 行前插入一行，耗时 %s", tbNum(line+1), tbDur(time.Since(start)))
	d.pending = a
}

// replaceFirst 把首行换成另一句话。
func (d *tbDemo) replaceFirst() {
	s := d.buf.Snapshot()
	a, z := s.LineRange(0)
	d.pushUndo()
	start := time.Now()
	d.buf.Replace(a, z, "first line replaced at "+time.Now().Format("15:04:05.000"))
	d.note = fmt.Sprintf("替换第 1 行，耗时 %s", tbDur(time.Since(start)))
	d.pending = a
}

// deleteHundred 删掉中间连续的 100 行。
func (d *tbDemo) deleteHundred() {
	s := d.buf.Snapshot()
	from := s.LineCount() / 2
	a, _ := s.LineRange(from)
	_, z := s.LineRange(from + 100)
	d.pushUndo()
	start := time.Now()
	d.buf.Replace(a, z, "")
	now := d.buf.Snapshot()
	d.note = fmt.Sprintf("删除第 %s–%s 行（剩 %s 行），耗时 %s",
		tbNum(from+1), tbNum(from+100), tbNum(now.LineCount()), tbDur(time.Since(start)))
	d.pending = a
}

// jump 把光标移到输入的行号。行号按 1 起算，越界会被 LineRange 钳到两端。
func (d *tbDemo) jump() {
	n, err := strconv.Atoi(strings.TrimSpace(d.jumpTo))
	if err != nil || n < 1 {
		d.note = "行号得是 ≥ 1 的整数"
		return
	}
	s := d.buf.Snapshot()
	a, _ := s.LineRange(n - 1)
	d.pending = a
	total := s.LineCount()
	if n > total {
		n = total
	}
	d.note = fmt.Sprintf("光标移到第 %s 行（共 %s 行）", tbNum(n), tbNum(total))
}

// benchmark 同时测两种插入：索引缓冲 vs 整串复制，各跑多轮取平均。
//
// 两个测量细节（都是实测踩出来的）：
//   - 整串那一路用 append 到预分配的缓冲，而不是 `raw[:cut] + ins + raw[cut:]`：
//     后者的结果只被用来取长度时，编译器会把整个拼接消掉，测出来 0 ns。
//   - 单次插入只有几微秒，轮数太少时读数会被调度抖动淹没（曾测出 0），
//     所以多跑几轮取平均。
func (d *tbDemo) benchmark() {
	const (
		ins      = ">> benchmark <<\n"
		ropeRuns = 500 // 索引缓冲：单次只有几微秒，多跑几轮取平均
		strRuns  = 20  // 整串复制：单次就要搬 4 MB，20 轮够了
	)

	s := d.buf.Snapshot()
	a, _ := s.LineRange(s.LineCount() / 2)
	d.pushUndo()
	start := time.Now()
	for range ropeRuns {
		d.buf.Replace(a, a, ins)
	}
	ropeNs := float64(time.Since(start).Nanoseconds()) / ropeRuns

	raw := d.buf.String()
	cut := strings.LastIndexByte(raw[:len(raw)/2], '\n') + 1
	dst := make([]byte, 0, len(raw)+len(ins))
	start = time.Now()
	for range strRuns {
		dst = dst[:0]
		dst = append(dst, raw[:cut]...)
		dst = append(dst, ins...)
		dst = append(dst, raw[cut:]...)
	}
	strNs := float64(time.Since(start).Nanoseconds()) / strRuns
	if len(dst) != len(raw)+len(ins) {
		d.note = "对比失败：复制长度不对"
		return
	}

	d.note = fmt.Sprintf("对比完成：往 %s 的文档中间插一行，各跑 %d / %d 轮取平均（插入的 %d 行可一次撤销）",
		tbBytes(len(raw)), ropeRuns, strRuns, ropeRuns)
	// 倍数要防除零：极端情况下单次插入可能小于 1ns。
	ratio := "> 1000"
	if ropeNs > 0 {
		ratio = tbNum(int(strNs / ropeNs))
	}
	d.lastRope, d.lastStr = ropeNs, strNs
	d.bench = fmt.Sprintf("索引缓冲 %s　·　整串复制 %s　·　快 %s 倍",
		tbNs(ropeNs), tbNs(strNs), ratio)
}

// textBufferPage 大文档编辑演示页。
func (a *App) textBufferPage(c *ui.Context, t *ui.Theme) {
	if a.tb == nil {
		a.tb = newTbDemo()
	}
	d := a.tb

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "大文档编辑（索引缓冲）",
			"10 万行文本存在持久化分块树里：改一个字符只复制路径上的几个块，不整篇复制", func() {
				ui.Text(c, fmt.Sprintf("%s 行 · %s", tbNum(d.buf.LineCount()), tbBytes(d.buf.ByteLen()))).
					FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			})

		// 工具条：编辑操作都在这里，每个按钮都把真实耗时写进下面的状态行。
		ui.Row(c).Gap(kit.CardGap).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "中间插一行").Tooltip("在文档正中间插入一行并记录耗时").Clicked() {
				d.insertMiddle()
			}
			if ui.Button(c, "改首行").Clicked() {
				d.replaceFirst()
			}
			if ui.Button(c, "删 100 行").Clicked() {
				d.deleteHundred()
			}
			if ui.Button(c, "撤销").Tooltip("回到上一个快照（Restore 是常数时间）").Clicked() {
				d.undo()
			}
			ui.TextInput(c, &d.jumpTo).Placeholder("行号").Width(96).Label("跳转到行")
			if ui.Button(c, "跳转").Clicked() {
				d.jump()
			}
			ui.Spacer(c).Grow(1)
			if ui.Button(c, "重建文档").Clicked() {
				a.tb = newTbDemo()
			}
		})

		// 文本区：固定高度 + 撑满宽度（不加 Fill()，否则高度会被父容器推回最小）。
		ta := ui.TextAreaBuffer(c, d.buf).Height(360).FillWidth().Font("monospace")
		if d.pending >= 0 {
			ta = ta.SetTextSelection(d.pending, d.pending)
			d.pending = -1
		}
		cur, _ := ta.TextSelection()

		// 状态行：篇幅、版本号、光标位置、撤销栈深度。
		s := d.buf.Snapshot()
		ui.Text(c, fmt.Sprintf("%s 行 · %s 字符 · %s · 版本 %d · 光标在第 %s 行 · 撤销栈 %d 步",
			tbNum(s.LineCount()), tbNum(s.Len()), tbBytes(s.ByteLen()),
			s.Version(), tbNum(s.LineAt(cur)+1), len(d.snaps))).
			FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		ui.Text(c, d.note).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "为什么不用 string", "", func() {
			if ui.Button(c, "跑一次对比").Clicked() {
				d.benchmark()
			}
		})
		if d.bench != "" {
			kit.Pill(c, t, d.bench, t.Accent)
		} else {
			ui.Text(c, "点右侧按钮：往 4 MB 文档中间插一行，一边用索引缓冲、一边用整串复制。").
				FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		}
		kit.KV(c, t, "小改成本", "string 是 O(篇幅)：改 1 个字符也要复制整篇；索引缓冲是 O(log 行数)")
		kit.KV(c, t, "快照与撤销", "Snapshot 不可变且共享块，Restore 常数时间 —— 撤销栈几乎不花内存")
		kit.KV(c, t, "和控件的关系", "TextAreaBuffer 只保留视口内的段落，不会把 10 万行都排版一遍")
		kit.KV(c, t, "什么时候不必用", "几 KB 的文本直接用 string + TextArea 就好，索引反而是负担")
	})
}

// tbDur 把耗时写成人能读的形式（ns / µs / ms 三档）。
func tbDur(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%d ns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%.1f µs", float64(d.Nanoseconds())/1000)
	default:
		return fmt.Sprintf("%.2f ms", float64(d.Nanoseconds())/1e6)
	}
}

// tbNs 把纳秒（可以是小数）写成人能读的形式。
//
// 和 tbDur 的区别：这条链路上的数值常常小于 1µs，用 time.Duration
// 换算会先把小数截掉（200 轮平均后可能变成 0），所以直接用 float。
func tbNs(ns float64) string {
	switch {
	case ns < 1000:
		return fmt.Sprintf("%.1f ns", ns)
	case ns < 1e6:
		return fmt.Sprintf("%.1f µs", ns/1e3)
	default:
		return fmt.Sprintf("%.2f ms", ns/1e6)
	}
}

// tbBytes 把字节数写成 KB / MB。
func tbBytes(n int) string {
	const kb = 1024
	switch {
	case n < kb:
		return fmt.Sprintf("%d B", n)
	case n < kb*kb:
		return fmt.Sprintf("%.1f KB", float64(n)/kb)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(kb*kb))
	}
}

// tbNum 加千分位，方便看清 10 万这个量级。
func tbNum(n int) string {
	if n < 0 {
		return "-" + tbNum(-n)
	}
	s := strconv.Itoa(n)
	var b strings.Builder
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
