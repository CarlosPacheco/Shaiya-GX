package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// A conversion packet shows the 30-cell party2 raid window, not the dialog.
func TestRaidConvertShowsRaidWindow(t *testing.T) {
	a := inputApp()
	a.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	a.state.SelfID = 1
	a.state.Party.Leader = 1
	a.state.Party.Members = []proto.PartyMember{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}
	w := proto.NewWriter(proto.RaidConvert)
	w.U8(1)
	w.U8(0)
	w.U32(1)
	w.U32(0)
	w.U32(1)
	a.applyWorldPacket(w.Bytes())
	if !a.state.Party.Raid || a.unionConfigOpen() || !a.raidGridShown() {
		t.Fatalf("raid=%v hidden=%v creating=%v", a.state.Party.Raid, a.party.raidHidden, a.party.creating)
	}
	// A click on a cell selects its member.
	gx, gy := a.partyUI.RaidGridOrigin(1024, 768)
	a.updatePartyPointer(&platform.Input{Pressed: true, MouseDown: true, MouseX: gx + 5 + 98*0, MouseY: gy + 40}, 1024, 768)
	a.updatePartyPointer(&platform.Input{Clicked: true, MouseX: gx + 5, MouseY: gy + 40}, 1024, 768)
	if a.state.Target != 2 {
		t.Fatalf("target %d", a.state.Target)
	}
}
