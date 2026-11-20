package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// A conversion packet must open the 30-member raid window.
func TestRaidConvertOpensWindow(t *testing.T) {
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
	if !a.state.Party.Raid || !a.unionConfigOpen() {
		t.Fatalf("raid=%v hidden=%v creating=%v", a.state.Party.Raid, a.party.raidHidden, a.party.creating)
	}
	x, y := a.partyUI.RaidOrigin(1024, 768)
	for i := 0; i < 5; i++ {
		a.updatePartyPointer(&platform.Input{MouseX: x + 390, MouseY: y + 410}, 1024, 768)
		a.updatePartyPointer(&platform.Input{MouseX: 5, MouseY: 5}, 1024, 768)
	}
	if !a.unionConfigOpen() {
		t.Fatal("window closed by itself")
	}
}
