package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// The user menu is never offered on a player of the other faction.
func TestPlayerMenuIsNotOfferedOnTheOtherFaction(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Me", Race: 0} // races 0-1 are one faction
	a.state.Guild.ID, a.state.Guild.Rank = 42, 2
	ally := &game.Entity{ID: 2, Kind: game.KindUser, Shape: &proto.UserShape{ID: 2, Race: 1, Name: "Ally"}}
	foe := &game.Entity{ID: 3, Kind: game.KindUser, Shape: &proto.UserShape{ID: 3, Race: 2, Name: "Foe"}}
	a.state.Entities[2], a.state.Entities[3] = ally, foe
	if len(a.contextActions(2)) == 0 {
		t.Fatal("no menu on an ally")
	}
	if got := a.contextActions(3); len(got) != 0 {
		t.Fatalf("a menu was offered on a player of the other faction: %v", got)
	}
}
