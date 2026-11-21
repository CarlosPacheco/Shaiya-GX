package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestNDoesNotSelectNearbyNPC(t *testing.T) {
	a := &App{state: game.New(), wv: &worldView{}, npcDefs: map[sdata.NPCKey]sdata.NPC{}}
	a.state.Entities[10] = &game.Entity{ID: 10, Kind: game.KindNpc, X: 5}
	for _, key := range []rune{'n', 'N'} {
		if a.updateShop(&platform.Input{Chars: []rune{key}}, 1024, 768) ||
			a.shop.npcID != 0 || a.wv.approachTarget != 0 {
			t.Fatalf("%c activated NPC interaction", key)
		}
	}
}
