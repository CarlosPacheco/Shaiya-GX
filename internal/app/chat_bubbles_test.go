package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestOrdinaryChatBubbleOnlyConfirmedEchoAndReplacement(t *testing.T) {
	a := inputApp()
	a.clock = 10
	a.state.AddChat(game.ChatLine{Op: proto.ChatParty, SenderID: 7, Text: "group"})
	a.confirmedChatBubble(proto.ChatParty)
	if len(a.chatBubbles) != 0 {
		t.Fatal("party produced ordinary bubble")
	}
	a.state.AddChat(game.ChatLine{Op: proto.ChatNormal, SenderID: 7, Text: "first"})
	a.confirmedChatBubble(proto.ChatNormal)
	a.clock = 12
	a.state.AddChat(game.ChatLine{Op: proto.ChatNormal, SenderID: 7, Text: "replaced 😀"})
	a.confirmedChatBubble(proto.ChatNormal)
	if len(a.chatBubbles) != 1 || a.chatBubbles[0].text != "replaced 😀" || a.chatBubbles[0].until != 17 {
		t.Fatal("bubble did not replace/reset", a.chatBubbles)
	}
}

func TestChatProtectionRunsBeforeHistoryAndBubbles(t *testing.T) {
	a := inputApp()
	a.clock = 10
	packet := func(id uint32, text string, items proto.ChatItems) []byte {
		w := proto.NewWriter(proto.ChatNormal)
		w.U32(id)
		w.Text(text)
		return proto.WithChatItems(w.Bytes(), items)
	}
	a.applyWorldPacket(packet(7, "Vendo armadura!", proto.ChatItems{}))
	if len(a.chatBubbles) != 1 || a.chatBubbles[0].until != 15 {
		t.Fatalf("ordinary chat did not create a bubble: %+v", a.chatBubbles)
	}
	a.clock = 12
	a.applyWorldPacket(packet(7, "VENDO ARMADURA", proto.ChatItems{}))
	a.state.Social.Blocked = map[uint32]proto.SocialContact{8: {ID: 8, Name: "Muted"}}
	a.applyWorldPacket(packet(8, "Ignored", proto.ChatItems{}))
	if len(a.state.Chat) != 1 || chatEntry(a.msg, a.state.Chat[0]).Repetitions != 2 || len(a.chatBubbles) != 1 || a.chatBubbles[0].until != 15 {
		t.Fatalf("repeat or blocked sender refreshed presentation: %+v / %+v", a.state.Chat, a.chatBubbles)
	}
	items := proto.ChatItems{{Type: 1, TypeID: 2, Quality: 100, Count: 1}}
	a.applyWorldPacket(packet(9, "Armadura y https://example.org/client.exe", items))
	if len(a.state.Chat) != 2 || a.state.Chat[1].Text != "Armadura y [link blocked]" || a.state.Chat[1].Items != items || len(a.chatBubbles) != 1 {
		t.Fatalf("unexpected unsafe-link presentation: %+v / %+v", a.state.Chat, a.chatBubbles)
	}
}

func TestChatBubblesSkipConcealedSpeakersIncludingSelf(t *testing.T) {
	a := inputApp()
	a.clock = 10
	a.state.SelfID = 1
	a.state.Entities[7] = &game.Entity{ID: 7, Concealed: true}
	for _, id := range []uint32{1, 7} {
		a.state.Self.Concealed = id == 1
		a.state.AddChat(game.ChatLine{Op: proto.ChatNormal, SenderID: id, Text: "psst"})
		a.confirmedChatBubble(proto.ChatNormal)
		if len(a.chatBubbles) != 0 {
			t.Fatalf("concealed speaker %d produced a bubble", id)
		}
	}
	a.state.Self.Concealed = false
	a.state.Entities[7].Concealed = false
	a.state.AddChat(game.ChatLine{Op: proto.ChatNormal, SenderID: 7, Text: "hi"})
	a.confirmedChatBubble(proto.ChatNormal)
	if len(a.chatBubbles) != 1 {
		t.Fatal("visible speaker lost its bubble")
	}
}
