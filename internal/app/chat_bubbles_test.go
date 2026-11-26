package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
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

func TestMountRiseEasesFromGroundToSeat(t *testing.T) {
	var blend float32
	ground, seat := render.V3{0, 10, 0}, render.V3{0, 11.5, 0}
	first := mountRise(&blend, 1.0/60, ground, seat)
	if first[1] <= 10 || first[1] > 10.1 {
		t.Fatalf("first frame must barely rise, got %v", first)
	}
	var last render.V3
	for range 30 {
		last = mountRise(&blend, 1.0/60, ground, seat)
	}
	if last != seat {
		t.Fatalf("rider must end on the seat, got %v", last)
	}
	blend = 0
	if mountRise(&blend, 0, ground, seat) != ground {
		t.Fatal("no time elapsed must keep the rider at the ground")
	}
}

func TestChatBubbleFollowsRenderedPlayerOrigin(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.wv.pos = render.V3{80, 17, 60}
	at := func(pos render.V3) render.M4 {
		return render.Basis(pos, render.V3{0, 0, -1}, render.V3{0, 1, 0})
	}
	self := &scene.Character{World: at(render.V3{40, 4, 35})}
	a.wv.self = self
	remote := &scene.Character{World: at(render.V3{41, 6, 35})}
	a.wv.actors = map[uint32]*remoteActor{2: {character: remote}}
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, X: 100, Y: 12, Z: 10}
	for id, want := range map[uint32]render.V3{1: {40, 4, 35}, 2: {41, 6, 35}} {
		if got, ok := a.chatBubbleOrigin(id); !ok || got != want {
			t.Fatalf("speaker %d origin %v, want %v", id, got, want)
		}
	}
	remote.World = at(render.V3{42, 7, 36})
	if got, ok := a.chatBubbleOrigin(2); !ok || got != (render.V3{42, 7, 36}) {
		t.Fatalf("remote bubble failed to follow the rendered movement: %v", got)
	}
	delete(a.state.Entities, 2)
	if _, ok := a.chatBubbleOrigin(2); ok {
		t.Fatal("bubble kept a stale speaker")
	}
}
