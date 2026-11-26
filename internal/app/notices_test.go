package app

import (
	"fmt"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestScreenNewsPreservesChatCapsQueueAndExpires(t *testing.T) {
	a := inputApp()
	a.msg = text.SysMsg{463: "Destroyed <t>"}
	for i := range 7 {
		a.clock = float64(i)
		a.systemMessage(8, 463, map[string]string{"t": fmt.Sprint(i)})
	}
	if len(a.state.Chat) != 7 || len(a.screenNotices) != 5 || a.screenNotices[0].text != "Destroyed 2" {
		t.Fatal(a.state.Chat, a.screenNotices)
	}
	a.clock = 12
	// With no font this still ages the queue without a GL context.
	a.drawScreenNotices(1024, 768)
	if len(a.screenNotices) != 5 {
		t.Fatal("native expiry must be strictly after 10s")
	}
	a.clock = 12.01
	a.drawScreenNotices(1024, 768)
	if len(a.screenNotices) != 4 {
		t.Fatal("oldest notice did not expire")
	}
	a.clock = 17
	a.drawScreenNotices(1024, 768)
	if len(a.screenNotices) != 0 || len(a.state.Chat) != 7 {
		t.Fatal("notice expiry modified chat history")
	}
	a.systemMessage(7, 463, map[string]string{"t": "local"})
	if len(a.screenNotices) != 0 || len(a.state.Chat) != 8 {
		t.Fatal("local altar notice became news")
	}
}

func TestServerNoticeUsesAltarNewsPresentation(t *testing.T) {
	a := inputApp()
	w := proto.NewWriter(proto.GXServerNotice)
	w.Text("GX Automatic Notice - Test")
	if _, err := a.state.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.confirmedSystemChat(w.Bytes(), 0)
	if len(a.state.Chat) != 1 || a.state.Chat[0].Style != 8 || !game.ChatSystemPane(a.state.Chat[0].Style) ||
		len(a.screenNotices) != 1 || a.screenNotices[0].text != a.state.Chat[0].Text {
		t.Fatalf("announcement missing altar presentation: chat=%v screen=%v", a.state.Chat, a.screenNotices)
	}
}

func TestMobSpeechUsesVioletNewsAndUpperChat(t *testing.T) {
	const body = "You have crossed a line."
	raw := proto.NewWriter(proto.MobSpeechText)
	raw.U8(1)
	raw.U32(42)
	raw.Text(body)
	indexed := proto.NewWriter(proto.MobSpeechIndex)
	indexed.U32(42)
	indexed.U16(6401)

	for _, packet := range []*proto.Writer{raw, indexed} {
		a := inputApp()
		a.msg = text.SysMsg{6401: body}
		if _, err := a.state.Apply(packet.Bytes()); err != nil {
			t.Fatal(err)
		}
		a.confirmedSystemChat(packet.Bytes(), 0)
		if len(a.screenNotices) != 1 || a.screenNotices[0].text != body || a.screenNotices[0].style != 30 {
			t.Fatalf("unexpected boss news: %v", a.screenNotices)
		}
		if len(a.state.Chat) != 1 {
			t.Fatalf("expected one boss chat entry: %v", a.state.Chat)
		}
		entry := chatEntry(a.msg, a.state.Chat[0])
		if entry.Text != body || !entry.System || entry.Color != ui.RGBColor(0xff80ff) {
			t.Fatalf("unexpected boss chat entry: %+v", entry)
		}
	}
}
