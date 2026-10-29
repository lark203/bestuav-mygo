package view

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/egoist/mygo/transfer"
)

// 这一组不碰系统剪贴板（离屏测试里 mygo.App 没在跑，Clipboard 会 panic），
// 只验数据模型与接收逻辑 —— 剪贴板往返留给真机。

// cbTestDemo 造一个带固定笔记的演示状态，redraw 用空函数。
func cbTestDemo() *cbDemo { return newCbDemo() }

// TestCbDataFormats 一份数据要同时提供三种表示，内容各自对得上。
func TestCbDataFormats(t *testing.T) {
	d := cbTestDemo()
	data := d.data(func() {})

	got := map[transfer.Format]bool{}
	for _, f := range data.Formats() {
		got[f] = true
	}
	for _, want := range []transfer.Format{transfer.Text, transfer.HTML, cbNoteFormat} {
		if !got[want] {
			t.Errorf("缺少表示 %q，实际 %v", want, data.Formats())
		}
	}

	// 纯文本：标题 + 换行 + 正文。
	text, err := data.Read(transfer.Text)
	if err != nil {
		t.Fatal(err)
	}
	if string(text) != d.note.Title+"\n"+d.note.Body {
		t.Errorf("text = %q", string(text))
	}

	// HTML：标题进 h1，并且要转义过（尖括号不能原样进标签）。
	raw, err := data.Read(transfer.HTML)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(raw), "<h1>") || !strings.Contains(string(raw), "<p>") {
		t.Errorf("html = %q", string(raw))
	}

	// 自定义：能原样解析回结构体。
	blob, err := data.Read(cbNoteFormat)
	if err != nil {
		t.Fatal(err)
	}
	var back cbNote
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("JSON 解析失败：%v", err)
	}
	if back != d.note {
		t.Errorf("JSON 往返后 = %+v，期望 %+v", back, d.note)
	}
}

// TestCbHTMLEscapes 标题里的尖括号不能直接进标签，否则粘贴方能注入结构。
func TestCbHTMLEscapes(t *testing.T) {
	d := cbTestDemo()
	d.note = cbNote{Title: `<script>alert(1)</script>`, Body: `a & b`}
	raw, err := d.data(func() {}).Read(transfer.HTML)
	if err != nil {
		t.Fatal(err)
	}
	html := string(raw)
	if strings.Contains(html, "<script>") {
		t.Errorf("标题没转义：%q", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") || !strings.Contains(html, "a &amp; b") {
		t.Errorf("转义结果不对：%q", html)
	}
}

// TestCbLazyOnlyOnDemand 自定义格式是惰性的：不索取就不序列化。
func TestCbLazyOnlyOnDemand(t *testing.T) {
	d := cbTestDemo()
	calls := 0
	data := d.data(func() { calls++ })

	if d.lazy != 0 {
		t.Fatalf("构造数据时不该调用提供者，实际 %d 次", d.lazy)
	}
	// 读别的表示也不该触发。
	if _, err := data.Read(transfer.Text); err != nil {
		t.Fatal(err)
	}
	if d.lazy != 0 {
		t.Fatalf("读纯文本触发了提供者，实际 %d 次", d.lazy)
	}
	// 索取才生成，并顺带请求一次重绘（计数要显示出来）。
	if _, err := data.Read(cbNoteFormat); err != nil {
		t.Fatal(err)
	}
	if d.lazy != 1 {
		t.Errorf("索取后提供者调用 %d 次，期望 1", d.lazy)
	}
	if calls != 1 {
		t.Errorf("重绘请求 %d 次，期望 1", calls)
	}
	// 再读一次：结果应被缓存，不再调用提供者。
	if _, err := data.Read(cbNoteFormat); err != nil {
		t.Fatal(err)
	}
	if d.lazy != 1 {
		t.Errorf("重复索取应命中缓存，提供者共调用 %d 次", d.lazy)
	}
}

// TestCbSnapshotFrozen 复制之后改笔记，已经交出去的数据不该跟着变。
func TestCbSnapshotFrozen(t *testing.T) {
	d := cbTestDemo()
	data := d.data(func() {})
	old := d.note.Title

	d.note.Title = "改过的标题"
	blob, err := data.Read(cbNoteFormat)
	if err != nil {
		t.Fatal(err)
	}
	var back cbNote
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatal(err)
	}
	if back.Title != old {
		t.Errorf("已复制的数据被后续修改影响了：标题 = %q，期望 %q", back.Title, old)
	}
}

// TestCbReceivePrefersOwnFormat 同时有纯文本与自定义格式时，优先用自定义（信息更全）。
func TestCbReceivePrefersOwnFormat(t *testing.T) {
	d := cbTestDemo()
	blob, _ := json.Marshal(cbNote{Title: "外来标题", Body: "外来正文"})
	incoming := transfer.New(transfer.NewItem(
		transfer.Bytes(transfer.Text, []byte("只有正文")),
		transfer.Bytes(cbNoteFormat, blob),
	))
	d.receive(incoming)

	if d.note.Title != "外来标题" || d.note.Body != "外来正文" {
		t.Errorf("收到 %+v，期望两段都被替换", d.note)
	}
	if !strings.Contains(d.status, "完整笔记") {
		t.Errorf("状态 = %q", d.status)
	}
}

// TestCbReceivePlainTextOnly 只有纯文本时，只覆盖正文，保留原标题。
func TestCbReceivePlainTextOnly(t *testing.T) {
	d := cbTestDemo()
	title := d.note.Title
	d.receive(transfer.TextData("别处复制来的一段话"))

	if d.note.Body != "别处复制来的一段话" {
		t.Errorf("正文 = %q", d.note.Body)
	}
	if d.note.Title != title {
		t.Errorf("纯文本不该覆盖标题，现在 = %q", d.note.Title)
	}
	if !strings.Contains(d.status, "纯文本") {
		t.Errorf("状态 = %q", d.status)
	}
}

// TestCbReceiveFiles 收到文件列表时只报出来，不当成笔记。
func TestCbReceiveFiles(t *testing.T) {
	d := cbTestDemo()
	d.receive(transfer.New(transfer.NewItem(
		transfer.Bytes(transfer.FileList, []byte("file:///C:/tmp/a.txt\r\n")),
	)))
	if !strings.Contains(d.status, "文件") {
		t.Errorf("状态 = %q，期望提到文件", d.status)
	}
}

// TestCbReceiveUnsupported 收到不认识的格式要给出明确提示，而不是静默。
func TestCbReceiveUnsupported(t *testing.T) {
	d := cbTestDemo()
	before := d.note
	d.receive(transfer.New(transfer.NewItem(
		transfer.Bytes(transfer.PNG, []byte{0x89, 'P', 'N', 'G'}),
	)))
	if d.note != before {
		t.Errorf("不认识的格式不该改笔记，现在 %+v", d.note)
	}
	if !strings.Contains(d.status, "没有能用的格式") {
		t.Errorf("状态 = %q", d.status)
	}
}

// TestCbCopyFileNeedsPath 空路径要拦下，不能把空字符串写进剪贴板。
func TestCbCopyFileNeedsPath(t *testing.T) {
	d := cbTestDemo()
	d.filePath = "   "
	d.copyFile()
	if !strings.Contains(d.status, "路径") {
		t.Errorf("状态 = %q，期望提示填路径", d.status)
	}
}
