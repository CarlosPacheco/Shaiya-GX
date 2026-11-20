package app

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestNPCTurnsToTheCharacterThatTalksToIt(t *testing.T) {
	a := &App{state: game.New(), frameDT: 1.0 / 60}
	a.state.Self.X, a.state.Self.Z = 10, 0
	e := &game.Entity{ID: 5, Kind: game.KindNpc, NpcType: 1}
	n := &npcActor{facing: render.V3{0, 0, 1}}
	pos := render.V3{0, 0, 0}
	a.turnNPCToTalker(5, e, n, pos)
	if n.facing != (render.V3{0, 0, 1}) {
		t.Fatal("an idle NPC turned")
	}
	a.shop.npcID = 5
	for range 120 {
		a.turnNPCToTalker(5, e, n, pos)
	}
	if math.Abs(float64(n.facing[0])-1) > 1e-3 || math.Abs(float64(n.facing[2])) > 1e-3 {
		t.Fatalf("facing %v, want toward the character", n.facing)
	}
	dead := &npcActor{facing: render.V3{0, 0, 1}}
	a.turnNPCToTalker(5, &game.Entity{ID: 5, Kind: game.KindNpc, NpcType: 12}, dead, pos)
	if dead.facing != (render.V3{0, 0, 1}) || dead.lookAt.Len() != 0 {
		t.Fatal("a dead NPC turned")
	}
}
