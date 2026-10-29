package view

import (
	"math/rand/v2"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"bestuav-mygo/internal/logging"
	"bestuav-mygo/internal/ui/kit"
)

// ---- 官方示例：Markdown 聊天（流式渲染）----
//
// 移植自 mygo examples/markdown-chat：回复按「词 / 句 / 行 / 段」逐块吐出，
// 用 markdown.Stream 渲染未完成的部分、markdown.View 渲染完成的部分，
// 列表跟随末尾滚动（用户往上滚时暂停跟随）。
//
// 与示例的差别：示例直接持有 *mygo.Window 调 Update；我们拿不到窗口对象，
// 改用注入的重绘触发器（a.requestFrame），后台协程统一走 mygo.RunOnMain。

// chatMessage 一条消息：用户的在右侧气泡，回复的按 Markdown 渲染。
type chatMessage struct {
	id   int
	mine bool
	text string
	done bool
}

var (
	chatRevealNames = []string{"按词", "按句", "按行", "按段", "整段"}
	chatRevealUnits = []markdown.RevealUnit{
		markdown.ByWord, markdown.BySentence, markdown.ByLine,
		markdown.ByParagraph, markdown.AtOnce,
	}
)

// markdownChatPage Markdown 聊天页。
func (a *App) markdownChatPage(c *ui.Context, t *ui.Theme) {
	a.chatList.Key = func(i int) any { return a.chatMsgs[i].id }
	a.chatList.FollowEnd = true
	a.chatList.FollowAnimation = &ui.Smooth // 回复变长时平滑上移
	a.chatMD.Reveal = &markdown.Reveal{Unit: chatRevealUnits[a.chatReveal]}

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "Markdown 聊天",
			"官方 examples/markdown-chat 移植：代码块 / 表格 / 公式 / 任务列表，可按不同粒度流式呈现", func() {
				ui.Segmented(c, &a.chatReveal, chatRevealNames...).Label("流式单位")
			})

		// 固定高度的会话区：页面整体是可滚动的，而输入框与列表需要确定的边界。
		ui.Column(c).Height(430).Children(func() {
			ui.List(c, &a.chatList, len(a.chatMsgs), func(i int) {
				a.chatMessage(c, t, a.chatMsgs[i])
			}).Grow(1).Gap(kit.PageGap).Justify(ui.End).Children(func() {
				if len(a.chatMsgs) == 0 {
					ui.Text(c, "随便问点什么：回复是内置文本，按模型的方式逐块吐出来。").
						TextColor(t.TextMuted).Margin(ui.Auto)
				}
			})

			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				in := ui.TextInput(c, &a.chatInput).Placeholder("输入消息，回车发送").Grow(1)
				streaming := len(a.chatMsgs) > 0 && !a.chatMsgs[len(a.chatMsgs)-1].done
				switch {
				case streaming:
					if ui.Button(c, "停止").Clicked() {
						a.chatMsgs[len(a.chatMsgs)-1].done = true
					}
				case (ui.PrimaryButton(c, "发送").
					Disabled(strings.TrimSpace(a.chatInput) == "").Clicked() || in.Submitted()) &&
					strings.TrimSpace(a.chatInput) != "":
					a.chatSend()
				}
			})
		})
	})
}

// chatMessage 渲染一条消息。
func (a *App) chatMessage(c *ui.Context, t *ui.Theme, m *chatMessage) {
	if m.mine {
		ui.Row(c).Children(func() {
			ui.Spacer(c)
			ui.Text(c, m.text).Selectable().Padding(8, 12).Radius(14).
				Background(t.Surface).MaxWidth(520)
		})
		return
	}
	ui.Column(c).MaxWidth(760).Children(func() {
		if m.done {
			markdown.View(c, m.text, &a.chatMD)
		} else {
			markdown.Stream(c, m.text, &a.chatMD)
		}
	})
}

// chatSend 发送输入，并流式产出下一条回复。
func (a *App) chatSend() {
	text := strings.TrimSpace(a.chatInput)
	a.chatNext++
	a.chatMsgs = append(a.chatMsgs, &chatMessage{id: a.chatNext, mine: true, text: text, done: true})
	a.chatInput = ""

	a.chatNext++
	reply := &chatMessage{id: a.chatNext}
	a.chatMsgs = append(a.chatMsgs, reply)

	src := chatReplies[a.chatReplies%len(chatReplies)]
	a.chatReplies++

	// 单测 / 无窗口环境：没有重绘触发器，直接一次性给出完整回复。
	if a.invalidate == nil {
		reply.text, reply.done = src, true
		return
	}

	go func() {
		defer logging.Recover("markdown-chat")
		for i := 0; i < len(src); {
			// 每次吐几个字节，节奏不均匀（模拟模型输出）。
			n := min(len(src)-i, 2+rand.IntN(7))
			for n < len(src)-i && src[i+n]&0xC0 == 0x80 {
				n++ // 不要把 UTF-8 字符切两半
			}
			i += n
			chunk := src[:i]
			stopped := false
			mygo.RunOnMain(func() {
				if reply.done {
					stopped = true
					return
				}
				reply.text = chunk
				a.requestFrame()
			})
			if stopped {
				return
			}
			time.Sleep(time.Duration(8+rand.IntN(30)) * time.Millisecond)
		}
		mygo.RunOnMain(func() {
			reply.done = true
			a.requestFrame()
		})
	}()
}

// chatReplies 内置回复：分别覆盖代码块、表格、公式与任务列表。
var chatReplies = []string{
	"下面是一个可并发使用的极简 **LRU 缓存**：\n\n```go\n" +
		"type LRU struct {\n\tmu    sync.Mutex\n\tcap   int\n\titems map[string]*list.Element\n\torder *list.List\n}\n\n" +
		"func (c *LRU) Get(key string) (any, bool) {\n\tc.mu.Lock()\n\tdefer c.mu.Unlock()\n" +
		"\tif e, ok := c.items[key]; ok {\n\t\tc.order.MoveToFront(e)\n\t\treturn e.Value.(*entry).value, true\n\t}\n\treturn nil, false\n}\n```\n\n" +
		"几点说明：\n\n1. `container/list` 让「最近使用」的维护是 O(1)。\n2. 互斥锁让 `Get` 也变成了写操作（它要移动节点）。\n3. 读多写多时可以按 key 哈希**分片**。\n",

	"## 几种缓存策略对比\n\n| 策略 | 读延迟 | 脏数据 | 复杂度 |\n|---|--:|:-:|---|\n" +
		"| 不用缓存 | ~41 ms | 无 | 极低 |\n| 旁路缓存 | ~0.2 ms | 中 | 低 |\n| 写穿透 | ~0.2 ms | 低 | 中 |\n\n" +
		"> **建议**：先用*旁路缓存*，等脏数据真的造成问题再上写穿透。\n\n" +
		"想先定位瓶颈，可以看看 [Go 的性能分析工具](https://go.dev/blog/pprof)。",

	"圆的面积是 $$A = \\pi r^2$$，周长是 $$C = 2\\pi r$$。\n\n" +
		"- [x] 量出半径\n- [x] 平方\n- [ ] 乘以 π\n\n" +
		"除非你需要~~近似值~~精确值——那交给 `math.Pi`，它有 15 位有效数字。",
}
