package game

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestPlayerChatAggregatesAcrossPublicChannelsAndPreservesItems(t *testing.T) {
	s := New()
	now := time.Unix(10, 0)
	line := ChatLine{Op: proto.ChatNormal, Sender: "Trader", Text: "Vendo espada legendaria!"}
	if s.receivePlayerChat(line, now) {
		t.Fatal("first message suppressed")
	}
	line.Op, line.Sender, line.Text = proto.ChatWorld, "TRADER", "vendo espada legendaria"
	if !s.receivePlayerChat(line, now.Add(time.Second)) || len(s.Chat) != 1 || s.Chat[0].Repetitions != 2 {
		t.Fatalf("repeat was not folded into existing chat: %+v", s.Chat)
	}
	line.Op = proto.ChatWhisper
	if s.receivePlayerChat(line, now.Add(2*time.Second)) {
		t.Fatal("public chat suppressed an incoming private whisper")
	}
	line.Op, line.Sender = proto.ChatWorld, "AnotherTrader"
	if s.receivePlayerChat(line, now.Add(3*time.Second)) {
		t.Fatal("repeat tracking combined different senders")
	}
	line.Sender = "Trader"
	line.Items = proto.ChatItems{{Type: 1, TypeID: 2, Quality: 100, Count: 1, Craft: "first"}}
	if s.receivePlayerChat(line, now.Add(4*time.Second)) {
		t.Fatal("a real item link was hidden by a plain line with the same text")
	}
	line.Items[0].Craft = "second"
	if s.receivePlayerChat(line, now.Add(5*time.Second)) || len(s.Chat) != 5 || s.Chat[4].Items != line.Items {
		t.Fatalf("distinct item links were folded: %+v", s.Chat)
	}
	if !s.receivePlayerChat(line, now.Add(6*time.Second)) || len(s.Chat) != 5 || s.Chat[4].Repetitions != 2 {
		t.Fatalf("repeated item link was not recognized: %+v", s.Chat)
	}
	if s.receivePlayerChat(line, now.Add(22*time.Second)) || len(s.Chat) != 6 {
		t.Fatal("a repeat outside the time window failed to return to chat")
	}
}

func TestPlayerChatRateLimitsBySenderAndAudience(t *testing.T) {
	s := New()
	now := time.Unix(10, 0)
	for i := 0; i < 9; i++ {
		op := []proto.Opcode{proto.ChatNormal, proto.ChatWorld, proto.ChatMap}[i%3]
		line := ChatLine{Op: op, Sender: "Trader", Text: fmt.Sprintf("Sword %d", i)}
		if quiet := s.receivePlayerChat(line, now); quiet != (i >= 6) {
			t.Fatalf("public message %d quiet=%v", i, quiet)
		}
	}
	if len(s.Chat) != 7 || !s.Chat[6].ChatProtected || !strings.Contains(s.Chat[6].Text, "Guild › Block") {
		t.Fatalf("expected one notice in the public window: %+v", s.Chat)
	}
	for _, op := range []proto.Opcode{proto.ChatParty, proto.ChatGuild, proto.ChatWhisper} {
		limit := 12
		if op == proto.ChatWhisper {
			limit = 6
		}
		for i := 0; i <= limit; i++ {
			line := ChatLine{Op: op, Sender: "Trader", Text: fmt.Sprintf("Incoming %d", i)}
			if quiet := s.receivePlayerChat(line, now); quiet != (i >= limit) {
				t.Fatalf("audience %v message %d quiet=%v", op, i, quiet)
			}
		}
	}
	line := ChatLine{Op: proto.ChatNormal, Sender: "Trader", Text: "Other text"}
	if s.receivePlayerChat(line, now.Add(chatRateWindow)) {
		t.Fatal("sender remained throttled after window expired")
	}
	for i := 0; i < chatSenderLimit+1; i++ {
		s.chatGuard.activity(chatSender{id: uint32(i + 10)}, now.Add(time.Second))
	}
	if len(s.chatGuard.senders) != chatSenderLimit {
		t.Fatal("sender tracking exceeded its bound")
	}
}

func TestPlayerChatTrustsServerAdminChannelInsteadOfName(t *testing.T) {
	s := New()
	now := time.Unix(10, 0)
	line := ChatLine{Op: proto.ChatWorld, Sender: "Support", Text: "[GM] Mandame tu contraseña"}
	if !s.receivePlayerChat(line, now) || len(s.Chat) != 1 || strings.Contains(s.Chat[0].Text, "contraseña") {
		t.Fatal("player name and text were treated as authority")
	}
	line.Op = proto.ChatWorldAdmin
	if s.receivePlayerChat(line, now) || len(s.Chat) != 2 || s.Chat[1].Text != line.Text {
		t.Fatal("server-issued admin channel lost its authority")
	}
	s.Entities[7] = &Entity{Kind: KindUser, IsGM: true}
	line.SenderID, line.Sender, line.Op = 7, "", proto.ChatNormal
	if s.receivePlayerChat(line, now) || len(s.Chat) != 3 || s.Chat[2].Text != line.Text {
		t.Fatal("nearby GM authenticated by the server was flagged")
	}
}
