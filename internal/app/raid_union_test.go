package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func labels(actions []contextAction) (out []string) {
	for _, a := range actions {
		out = append(out, a.label)
	}
	return out
}

func unionApp(self uint32) *App {
	a := inputApp()
	a.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	a.state.SelfID = self
	a.state.Party.Raid, a.state.Party.Leader, a.state.Party.RaidSubLeader = true, 1, 2
	a.state.Party.Members = []proto.PartyMember{{ID: 1, Name: "L"}, {ID: 2, Name: "S"}, {ID: 3, Name: "M"}}
	a.state.Party.RaidSlots = map[uint16]uint32{0: 1, 1: 2, 2: 3}
	a.party.raidHidden, a.party.unionOpen = true, true
	return a
}

// 0x47ebe0 mode 1: the portrait menu by role.
func TestUnionPortraitMenu(t *testing.T) {
	a := unionApp(1)
	if got := labels(a.portraitActions()); len(got) != 4 || got[0] != "Leave union" || got[1] != "Dismantle union" || got[2] != "Config union" || got[3] != "Hide union" {
		t.Fatalf("leader %v", got)
	}
	a.party.unionOpen = false
	a.state.SelfID = 3
	if got := labels(a.portraitActions()); len(got) != 3 || got[0] != "Leave union" || got[1] != "Config union" || got[2] != "Open union" {
		t.Fatalf("member %v", got)
	}
	a.state.Party.Raid = false
	if got := labels(a.portraitActions()); len(got) != 1 || got[0] != "Create union" {
		t.Fatalf("party %v", got)
	}
}

// 0x47ebe0 mode 2: the slot menu by role.
func TestUnionSlotMenu(t *testing.T) {
	a := unionApp(1)
	for _, c := range []struct {
		self uint32
		slot int
		want string
	}{
		{1, 5, "Move group"},
		{1, 0, "Move group"},
		{1, 1, "Move group|Delegate leader|Remove from raid|Union whisper"},
		{1, 2, "Move group|Delegate leader|Make assistant|Remove from raid|Union whisper"},
		{2, 0, "Move group|Union whisper"},
		{2, 1, "Move group"},
		{3, 0, "Union whisper"},
	} {
		a.state.SelfID = c.self
		got := ""
		for i, l := range labels(a.unionActions(c.slot)) {
			if i > 0 {
				got += "|"
			}
			got += l
		}
		if got != c.want {
			t.Errorf("self %d slot %d: %q want %q", c.self, c.slot, got, c.want)
		}
	}
}

func TestUnionCreateNeedsLootMode(t *testing.T) {
	a := inputApp()
	a.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	a.state.SelfID = 1
	a.state.Party.Leader = 1
	a.state.Party.Members = []proto.PartyMember{{ID: 1}, {ID: 2}}
	a.performContextAction(1, contextAction{label: "Create union"})
	if !a.unionConfigOpen() {
		t.Fatal("create did not open the window")
	}
	if a.applyUnionConfig() {
		t.Fatal("OK without a loot mode must stay open")
	}
	a.partyUI.SetRaidSettings(0, false)
	x, y := a.partyUI.RaidOrigin(1024, 768)
	for _, c := range []platform.Input{{Pressed: true, MouseDown: true, MouseX: x + 35, MouseY: y + 315}, {Clicked: true, MouseX: x + 35, MouseY: y + 315}} {
		c := c
		a.updatePartyPointer(&c, 1024, 768)
	}
	if d, _ := a.partyUI.RaidSettings(); d != 1 {
		t.Fatalf("division %d", d)
	}
}

func TestMoveGroupSwapsSlots(t *testing.T) {
	a := unionApp(1)
	a.performContextAction(1, contextAction{label: "Move group", slot: 2})
	if !a.party.moving || a.party.moveSlot != 2 {
		t.Fatal("move not armed")
	}
	a.finishMove(2) // same slot cancels
	if a.party.moving {
		t.Fatal("move stayed armed")
	}
}

func TestRaidPartyNotices(t *testing.T) {
	a := unionApp(1)
	a.state.Party.Notices = nil
	for _, id := range []int{6516, 6531} {
		a.state.Party.Notices = append(a.state.Party.Notices, struct {
			ID   int
			Name string
		}{id, ""})
	}
	a.drainPartyNotices()
	if len(a.state.Party.Notices) != 0 {
		t.Fatal("notices not drained")
	}
}
