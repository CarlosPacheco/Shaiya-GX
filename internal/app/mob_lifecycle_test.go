package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"testing"
)

func TestCorpseUsesOpaqueHoldAndLightweightFade(t *testing.T) {
	for _, tc := range []struct {
		age  float64
		fade float32
	}{{0, 0}, {3.5, 0}, {4.25, .5}, {5, 1}, {100, 1}} {
		if got := corpseFade(tc.age); got != tc.fade {
			t.Fatal(tc, got)
		}
	}
}

func TestRepeatedDeathDoesNotRestartFadeAndRespawnClearsIt(t *testing.T) {
	m := mobActor{}
	if m.updateCorpse(true, 10) != 0 || m.updateCorpse(true, 14.25) != .5 || m.updateCorpse(true, 20) != 1 {
		t.Fatal("death updates restarted the lifetime")
	}
	if m.updateCorpse(false, 21) != 0 || m.updateCorpse(true, 25) != 0 || m.updateCorpse(true, 29.25) != .5 {
		t.Fatal("respawn did not start a new lifecycle")
	}
}

func TestSpawnAnchorFreezesPacketCoordinatesDespiteMobMovementOrRemoval(t *testing.T) {
	a := inputApp()
	e := &game.Entity{ID: 9, Kind: game.KindMob, X: 100, Y: 20, Z: 30, MobHeightKnown: true}
	a.state.Entities[9] = e
	actor := &scene.CreatureActor{Character: &scene.Character{World: render.Basis(render.V3{99, 20, 30}, render.V3{1, 0, 0}, render.V3{0, 1, 0})}}
	a.wv.mobs = map[uint32]*mobActor{9: {entity: e, actor: actor, position: remotePosition{initialized: true}, pos: render.V3{99, 20, 30}}}
	matrix := a.mobSpawnTransform(proto.MobSpawn{ID: 9, X: 10, Y: 20, Z: 30, HeightKnown: true})
	impact := skillImpact{fixedWorld: &matrix}
	expected := render.Identity()
	expected[12], expected[13], expected[14] = 10, 20, 30
	for _, remove := range []bool{false, true} {
		e.X, e.Z = 500, 600
		actor.World[12] = 400
		a.wv.mobs[9].pos = render.V3{400, 20, 30}
		if remove {
			delete(a.state.Entities, 9)
		}
		if got, ok := a.impactTransform(impact); !ok || got != expected {
			t.Fatal("spawn followed live monster", got, ok)
		}
	}
}
