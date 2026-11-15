//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Map 15 has no terrain heightmap; a click on its floor must still select a
// destination on the authored dungeon floor.
func TestDungeonClickMoveSelectsFloorGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and game resources")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const W, H = 1600, 900
	win, e := platform.Open(platform.Config{Width: W, Height: H, Hidden: true})
	if e != nil {
		t.Fatal(e)
	}
	defer win.Close()
	if e = render.Init(false); e != nil {
		t.Fatal(e)
	}
	arc, e := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if e != nil {
		t.Fatal(e)
	}
	defer arc.Close()
	a, e := New(Config{}, arc)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	a.state = game.New()
	a.state.Phase = game.PhaseInWorld
	a.state.MapID, a.state.SelfID = 15, 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: 0, Class: 0, Height: 2}
	a.state.Self.X, a.state.Self.Y, a.state.Self.Z = 175, 7.6, 26
	a.screen = screenWorld
	a.enterWorld()
	a.frameDT = 1.0 / 60
	for range 30 {
		a.Draw(W, H)
	}
	if a.wv.world.HasTerrain() {
		t.Fatal("map 15 should be a dungeon")
	}
	a.moveToGround(&platform.Input{MouseX: W / 2, MouseY: H/2 + 120, Clicked: true}, W, H)
	d := a.wv.destination
	if !a.wv.travel || d[1] > a.wv.pos[1]+.5 || d.Sub(a.wv.pos).Len() > 40 {
		t.Fatalf("destination %v travel %v from %v", d, a.wv.travel, a.wv.pos)
	}
}
