package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
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

func TestPlayerMenuFromTargetUsesNativeDistance(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: 0}
	a.state.Target = 2
	target := &game.Entity{ID: 2, Kind: game.KindUser, Y: 11, Shape: &proto.UserShape{ID: 2, Race: 1, Name: "Ally"}}
	a.state.Entities[2] = target
	pointer := platform.Input{MouseX: 270, MouseY: 20, RightPressed: true}
	if a.updatePlayerContext(&pointer, 1024, 768) || a.playerContext.open {
		t.Fatal("a three-dimensional distance above ten opened the menu")
	}
	target.Y = 10
	if !a.updatePlayerContext(&pointer, 1024, 768) || !a.playerContext.open {
		t.Fatal("a distance of ten should open the menu")
	}
	target.Y = 11
	a.updatePlayerContext(&platform.Input{}, 1024, 768)
	if a.playerContext.open {
		t.Fatal("the menu stayed open after leaving range")
	}
}
