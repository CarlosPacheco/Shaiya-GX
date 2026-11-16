//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"os"
	"runtime"
	"testing"
)

func TestSpawnWorldAnchorAndStationaryCorpseOriginalActorGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.assets = scene.NewAssets(arc)
	a.clock = 10
	a.frameDT = 1.0 / 60
	e := &game.Entity{ID: 9, Kind: game.KindMob, TypeID: 2009, X: 10, Y: 20, Z: 30, MobHeightKnown: true, MobMoveDurationMS: 1000}
	a.state.Entities[9] = e
	a.prepareMonsters()
	cached := a.wv.mobs[9]
	if cached == nil || cached.actor == nil {
		t.Fatal("missing original monster")
	}
	e.X = 12
	e.MoveSerial++
	a.clock = 10.1
	a.prepareMonsters()
	a.clock = 10.4
	a.prepareMonsters()
	if cached.pos[0] <= 10 || cached.pos[0] >= 12 {
		t.Fatal("fixture did not start interpolating", cached.pos)
	}
	deathWorld := cached.actor.World
	deathPos := cached.pos
	e.Dead = true
	a.clock = 10.45
	a.prepareMonsters()
	if cached.actor.World != deathWorld || cached.position.duration != 0 {
		t.Fatal("death advanced pending movement")
	}
	for frame := 0; frame < 360; frame++ {
		a.clock += 1.0 / 60
		e.X += .01
		e.Y += .01
		e.Z += .01
		e.MoveSerial++
		a.prepareMonsters()
		if cached.actor.World != deathWorld || cached.pos != deathPos {
			t.Fatal("corpse drifted after death", frame, cached.pos, deathPos)
		}
	}
	if cached.actor.CorpseFade != 1 {
		t.Fatal("normal disappearance did not complete", cached.actor.CorpseFade)
	}
	heldTime := cached.actor.Time
	cached.actor.PlayOnce(2)
	cached.actor.PlayOnce(7)
	cached.actor.PlayOnce(5)
	a.clock += .1
	a.prepareMonsters()
	if cached.actor.Time != heldTime {
		t.Fatal("dead animation did not hold final pose")
	}
	e.Dead = false
	e.X, e.Y, e.Z = 55, 20, 33
	e.PositionResetSerial++
	a.clock += .1
	a.prepareMonsters()
	if cached.pos != (render.V3{55, 20, 33}) || cached.actor.CorpseFade != 0 {
		t.Fatal("respawn retained corpse anchor", cached.pos)
	}
	w := proto.NewWriter(proto.GXMobEnter)
	w.U32(9)
	w.Bool(true)
	w.U16(2009)
	w.F32(55)
	w.F32(20)
	w.F32(33)
	a.confirmedEffects(w.Bytes())
	if len(a.wv.impacts) != 1 {
		t.Fatal("missing spawn effect")
	}
	p := a.wv.impacts[0]
	before, ok := a.impactTransform(p)
	if !ok || before[12] != 55 || before[13] != 20 || before[14] != 33 {
		t.Fatal("spawn did not use packet coordinates", before)
	}
	e.X = 90
	e.MoveSerial++
	a.clock += 1
	a.prepareMonsters()
	after, ok := a.impactTransform(p)
	if !ok || after != before {
		t.Fatal("spawn effect followed moving monster", before, after)
	}
	delete(a.state.Entities, 9)
	if after, ok := a.impactTransform(p); !ok || after != before {
		t.Fatal("spawn anchor disappeared with entity")
	}
}
