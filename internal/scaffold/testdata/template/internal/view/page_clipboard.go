package view

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/kit"
)

// ---- 官方示例：多格式剪贴板与拖放 ----
//
// 移植自 mygo examples/clipboard。它演示的是**一套数据模型同时喂给三个渠道**：
// 系统剪贴板、应用内拖放、以及从别的应用拖进来。
//
// 一份「笔记」同时提供三种表示：
//   - text/plain      任何接收方都能用
//   - text/html       富文本目标（Word、邮件）用它保留格式
//   - 自定义 JSON     只有本应用认识的完整结构（含标题），别的应用看到也不认识
// 自定义格式用 transfer.Lazy 注册：**只有接收方真的索取时才生成**，
// 没人要就不花那次序列化。示例里的「惰性请求次数」就是这件事的计数器。
//
// 与示例的差别：
//   - d.win.Invalidate() 换成 a.requestFrame()（本项目 view 不持有窗口）；
//   - 自定义格式改成自家命名空间 bestuav，不占用 mygo 的；
//   - 拖放的源与目标是同一张卡里的两个盒子，落地时顺手把状态写清楚。

// cbNoteFormat 剪贴板里代表「一条笔记」的自定义格式。
const cbNoteFormat transfer.Format = "application/vnd.bestuav.clipboard-note+json"

// cbNote 一条笔记。自定义格式里存的就是它的 JSON。
type cbNote struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// cbDemo 剪贴板演示的状态。
type cbDemo struct {
	note     cbNote
	filePath string // 「复制文件路径」用的路径
	status   string // 上一次操作的结果
	lazy     int    // 自定义格式被索取的次数（惰性提供者被调用）
	releases int    // 剪贴板释放次数（OnRelease 被调用）
}

// newCbDemo 造一条示例笔记。
func newCbDemo() *cbDemo {
	return &cbDemo{
		note:   cbNote{Title: "可移植的笔记", Body: "同一份内容，剪贴板里同时是纯文本、HTML 和本应用的 JSON。"},
		status: "还没操作过",
	}
}

// data 把当前笔记打包成三种表示。
//
// 关键点：闭包里冻结的是「此刻」的值（frozen），不是 d.note ——
// 否则用户在复制之后再改标题，粘贴方拿到的 JSON 会莫名其妙变成新内容。
func (d *cbDemo) data(redraw func()) transfer.Data {
	frozen := d.note
	return transfer.New(transfer.NewItem(
		transfer.Bytes(transfer.Text, []byte(frozen.Title+"\n"+frozen.Body)),
		transfer.Bytes(transfer.HTML, []byte("<h1>"+html.EscapeString(frozen.Title)+"</h1><p>"+
			html.EscapeString(frozen.Body)+"</p>")),
		transfer.Lazy(cbNoteFormat, func() ([]byte, error) {
			d.lazy++
			redraw()
			return json.Marshal(frozen)
		}),
	))
}

// receive 把外来数据收进来：有自家格式就用它（信息最全），否则退到纯文本。
func (d *cbDemo) receive(data transfer.Data) {
	if f, ok := data.Preferred(cbNoteFormat, transfer.Text); ok {
		b, err := data.Read(f)
		if err != nil {
			d.status = err.Error()
			return
		}
		if f == cbNoteFormat {
			if err := json.Unmarshal(b, &d.note); err != nil {
				d.status = "自定义格式解析失败：" + err.Error()
				return
			}
			d.status = "已接收完整笔记（" + string(f) + "）"
		} else {
			// 纯文本没有标题，只覆盖正文 —— 与示例一致。
			d.note.Body = string(b)
			d.status = "已接收纯文本，只覆盖了正文"
		}
		return
	}
	if files, err := data.Files(); err == nil && len(files) > 0 {
		d.status = "收到文件：" + strings.Join(files, "、")
		return
	}
	d.status = "剪贴板里没有能用的格式"
}

// copyNote 把笔记写进系统剪贴板。自定义格式是惰性的，这里只登记、不序列化。
func (d *cbDemo) copyNote(redraw func()) {
	err := mygo.Clipboard.Write(d.data(redraw), mygo.ClipboardOptions{
		OnRelease: func() { d.releases++; redraw() },
	})
	if err != nil {
		d.status = "复制失败：" + err.Error()
		return
	}
	d.status = "已复制（JSON 只在你索取时才生成）"
}

// paste 按偏好顺序挑一个能用的格式读回来。
func (d *cbDemo) paste() {
	preferred, _ := transfer.PreferredFormat(mygo.Clipboard.Formats(),
		cbNoteFormat, transfer.Text, transfer.FileList, transfer.URIList)
	if preferred == "" {
		d.status = "剪贴板里没有本应用支持的格式"
		return
	}
	data, err := mygo.Clipboard.Read(preferred)
	if err != nil {
		d.status = "读取失败：" + err.Error()
		return
	}
	d.receive(data)
}

// copyFile 把路径作为「文件列表」写进剪贴板（粘贴到资源管理器就是那个文件）。
func (d *cbDemo) copyFile() {
	p := strings.TrimSpace(d.filePath)
	if p == "" {
		d.status = "先填一个存在的文件路径"
		return
	}
	if err := mygo.Clipboard.WriteFiles(p); err != nil {
		d.status = "复制文件失败：" + err.Error()
		return
	}
	d.status = "已把文件写进剪贴板：" + p
}

// flush 让系统接管剪贴板内容（之后即使应用退出，粘贴也还在）。
func (d *cbDemo) flush() {
	if err := mygo.Clipboard.Flush(); err != nil {
		d.status = "持久化失败：" + err.Error()
		return
	}
	d.status = "已交给系统托管，惰性资源已释放"
}

// clipboardPage 多格式剪贴板演示页。
func (a *App) clipboardPage(c *ui.Context, t *ui.Theme) {
	if a.cb == nil {
		a.cb = newCbDemo()
	}
	d := a.cb
	redraw := a.requestFrame

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "多格式剪贴板与拖放",
			"同一份笔记同时是纯文本、HTML 和本应用的 JSON；自定义格式只有被索取时才生成", func() {
				ui.Text(c, fmt.Sprintf("惰性请求 %d 次 · 剪贴板释放 %d 次", d.lazy, d.releases)).
					FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			})

		ui.Row(c).Gap(kit.CardGap).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, &d.note.Title).Placeholder("标题").Width(320).Label("笔记标题")
			ui.Spacer(c).Grow(1)
			if ui.Button(c, "复制笔记").Tooltip("写进系统剪贴板（三种表示）").Clicked() {
				d.copyNote(redraw)
			}
			if ui.Button(c, "粘贴").Tooltip("按偏好挑格式读回来").Clicked() {
				d.paste()
			}
			if ui.Button(c, "交给系统").Tooltip("Flush：让系统接管内容，之后应用退出也能粘贴").Clicked() {
				d.flush()
			}
		})
		ui.TextArea(c, &d.note.Body).Height(80).Label("笔记正文")

		// 拖放：左边拖出去、右边放进来。两边用的是同一套 transfer.Data。
		opts := transfer.DropOptions{Formats: []transfer.Format{
			cbNoteFormat, transfer.Text, transfer.FileList, transfer.URIList,
		}}
		kit.RowGrid(c, 2, func() {
			ui.Box(c).Padding(14).Border(1, t.Border).Radius(kit.CardRadius).
				DragDataFrom(func() transfer.Data { return d.data(redraw) }).
				Children(func() {
					ui.Text(c, "拖我").Bold()
					ui.Text(c, "按住拖到右边，或拖到别的应用").
						FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				})
			zone := ui.Box(c).Padding(14).Border(1, t.Border).Radius(kit.CardRadius)
			if _, over := ui.DataDragOver(zone, opts); over {
				zone.Border(2, t.Accent)
			}
			zone.Children(func() {
				ui.Text(c, "放这里").Bold()
				ui.Text(c, "笔记 / 文本 / 文件都收").
					FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			})
			if drop, ok := ui.DropData(zone, opts); ok {
				d.receive(drop.Data)
			}
		})

		ui.Row(c).Gap(kit.CardGap).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, &d.filePath).Placeholder("文件的绝对路径").Grow(1).Label("文件路径")
			if ui.Button(c, "复制文件").Clicked() {
				d.copyFile()
			}
		})
		ui.Text(c, d.status).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Wrap()
	})

	kit.Card(c, t, func() {
		kit.CardHeader(c, t, "为什么一份数据要三种表示", "", nil)
		kit.KV(c, t, "text/plain", "谁都能读：记事本、终端、输入框。信息最少的兜底")
		kit.KV(c, t, "text/html", "富文本目标（Word / 邮件）用它保留标题和段落结构")
		kit.KV(c, t, "自定义 JSON", "只有本应用认识：带上标题等纯文本装不下的字段")
		kit.KV(c, t, "Lazy 的意义", "自定义格式只在被索取时序列化，复制动作本身不产生这份开销")
		kit.KV(c, t, "与拖放的关系", "拖拽、页内放下、系统剪贴板共用同一个 transfer.Data，不用写两套")
		kit.KV(c, t, "注意", "复制时冻结的是当时的快照：之后再改标题，不会影响已经复制出去的内容")
	})
}

// closeClipboard 释放演示状态。
//
// 剪贴板内容归系统所有（Flush 之后更是如此），这里只丢掉本地的笔记与计数 ——
// 刻意不碰剪贴板：用户可能正要去别的应用里粘贴刚才复制的东西。
func (a *App) closeClipboard() { a.cb = nil }
