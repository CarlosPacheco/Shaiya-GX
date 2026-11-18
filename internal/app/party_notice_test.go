package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Party refusals are native SysMsg lines in the chat log, never a screen overlay.
func TestPartyRefusalsGoToChatAsSysMsg(t *testing.T) {
	a := inputApp()
	a.partyMessage, _ = ui.NewMessage(inputTextures{})
	a.msg = text.SysMsg{206: "<p> rejected the party's invitation", 347: "The character doesn't exist.", 6540: "You don't have invitation authority."}
	a.party.inviteName = "Faraway"
	last := func() string {
		if n := len(a.state.Chat); n > 0 {
			return a.state.Chat[n-1].Text
		}
		return ""
	}
	a.state.Party.ReplySerial, a.state.Party.Reply = 1, proto.PartyReply{Result: 4, ID: 99}
	a.updatePartyInvite(&platform.Input{}, 1024, 768)
	if last() != "The character doesn't exist." || a.combatNotice != "" {
		t.Fatalf("unavailable reply: chat %q overlay %q", last(), a.combatNotice)
	}
	// A decline from another map has no entity: the typed name is used.
	a.state.Party.ReplySerial, a.state.Party.Reply = 2, proto.PartyReply{Result: 0, ID: 99}
	a.updatePartyInvite(&platform.Input{}, 1024, 768)
	if last() != "Faraway rejected the party's invitation" || a.combatNotice != "" {
		t.Fatalf("decline: chat %q overlay %q", last(), a.combatNotice)
	}
	a.state.Party.RaidErrorSerial, a.state.Party.RaidError = 1, 2
	a.updatePartyInvite(&platform.Input{}, 1024, 768)
	if last() != "You don't have invitation authority." || a.combatNotice != "" {
		t.Fatalf("raid refusal: chat %q overlay %q", last(), a.combatNotice)
	}
	for _, style := range []uint8{3} {
		if nativeNoticeStyle(style) {
			t.Fatal("party notices must stay out of the screen overlay")
		}
	}
}
