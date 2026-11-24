//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/movespeed"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestMonsterVisualMovementOnOriginalTerrainGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "monster motion", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := scene.NewAssets(arc)
	w, err := scene.LoadMap(assets, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	s := game.New()
	s.Entities[42] = &game.Entity{ID: 42, Kind: game.KindMob, TypeID: 2009, X: 500, Z: 600}
	a := &App{assets: assets, state: s, wv: &worldView{world: w}}
	a.prepareMonsters()
	mob := a.wv.mobs[42]
	if mob == nil || mob.actor == nil {
		t.Fatal("original monster failed to load")
	}
	if !mob.actor.PlayOnce(7) {
		t.Fatal("original monster failed to start its damage reaction")
	}
	move := proto.NewWriter(proto.MobMove)
	move.U32(42)
	move.U8(1)
	move.F32(504)
	move.F32(600)
	if _, err := s.Apply(move.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.clock, a.frameDT = 1, .02
	a.prepareMonsters()
	if mob.pos[0] != 500 {
		t.Fatal("monster snapped to packet", mob.pos)
	}
	if mob.actor.Time <= 0 {
		t.Fatal("chase playback did not start on the first frame")
	}
	beforeHit := mob.actor.Time
	a.reactToMobHit(mob)
	if mob.actor.Time != beforeHit {
		t.Fatal("hit interrupted active chase playback")
	}
	def := mob.actor.Definition
	duration := 4 / movespeed.MobUnitsPerSecond(int(def.ChaseStep), def.ChaseTimeMS, 2)
	a.clock = 1 + duration/2
	a.prepareMonsters()
	if mob.pos[0] != 502 || mob.pos[1] != w.Height(502, 600) || mob.actor.World[12] != 502 {
		t.Fatalf("model did not follow interpolated terrain position: %v", mob.pos)
	}
	if got := a.visualEntityPosition(s.Entities[42]); got != mob.pos {
		t.Fatal("effects diverged from model", got, mob.pos)
	}
	a.clock = 1 + duration
	a.prepareMonsters()
	if mob.pos[0] != 504 || s.Entities[42].X != 504 {
		t.Fatal("monster did not reach server position")
	}
	mob.actor.PlayOnce(2)
	mob.actor.PlayOnce(7)
	if !mob.actor.Attacking() {
		t.Fatal("damage reaction interrupted the confirmed attack")
	}
	// Re-entry can reuse an ID; the new entity must not inherit stale motion.
	s.Entities[42] = &game.Entity{ID: 42, Kind: game.KindMob, TypeID: 2009, X: 700, Z: 700}
	a.prepareMonsters()
	if got := a.wv.mobs[42].pos; got[0] != 700 || got[2] != 700 {
		t.Fatal("re-entered monster inherited old position", got)
	}
}
