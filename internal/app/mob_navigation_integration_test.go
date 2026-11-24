//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestMobNavigationDungeonAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_MOB_NAVIGATION") != "1" {
		t.Skip("requires isolated map-3 fixture")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "dungeon mob navigation probe", Width: 1024, Height: 768, Hidden: true})
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
	st, w, _, wait := testworld.Enter(t)
	countMobs := func() int {
		n := 0
		for _, e := range st.Entities {
			if e.Kind == game.KindMob {
				n++
			}
		}
		return n
	}
	wait(func() bool { return countMobs() >= 5 })
	if st.MapID != 3 {
		t.Fatalf("expected map 3, got %d", st.MapID)
	}
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, w, screenWorld
	a.enterWorld()
	if a.wv.world == nil || a.wv.world.HasTerrain() {
		t.Fatal("map 3 did not load as dungeon")
	}
	verifyPlayerAuthorityNetwork(t, a, wait)
	a.wv.eye = a.wv.pos.Add(render.V3{7, 5, -10})
	a.wv.cameraReady = false
	a.prepareMonsters()
	grounded := 0
	for id, e := range st.Entities {
		if e.Kind != game.KindMob {
			continue
		}
		if !e.MobHeightKnown || math.Abs(float64(e.Y)) < .1 {
			t.Fatalf("mob %d missing server height: %+v", id, e)
		}
		actor := a.wv.mobs[id]
		if actor == nil || actor.actor == nil {
			continue
		}
		if math.Abs(float64(actor.pos[1]-e.Y)) > .5 {
			t.Fatalf("client replaced server floor for mob %d: rendered %v server %g", id, actor.pos, e.Y)
		}
		grounded++
	}
	if grounded < 5 {
		t.Fatalf("too few rendered mobs: %d", grounded)
	}
	gl.Viewport(0, 0, 1024, 768)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.Draw(1024, 768)
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("GL error %#x", err)
	}
	if directory := os.Getenv("SHAIYA_TEST_ARTIFACTS"); directory != "" {
		if err = render.SavePNG(filepath.Join(directory, "dungeon-authoritative-mobs.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("map 3: %d visible mobs retain authoritative height on original dungeon surfaces", grounded)
}

// The isolated server must reject client-authored height and keep simulating
// a jump when the client stops sending. No production bypass is used.
func verifyPlayerAuthorityNetwork(t *testing.T, a *App, wait func(func() bool)) {
	t.Helper()
	st := a.state
	original := a.wv.pos
	request := func(x, y, z float32) {
		w := proto.NewWriter(proto.CharacterMove)
		w.U8(1)
		w.U16(0)
		w.F32(x)
		w.F32(y)
		w.F32(z)
		if err := a.world.Send(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	request(original[0], original[1]+100, original[2])
	wait(func() bool { return st.LastPlayerCorrection.ID == st.SelfID })
	if math.Abs(float64(st.Self.Y-original[1])) > .15 {
		t.Fatal("forged Y accepted", st.Self.Y, original)
	}
	if st.LastPlayerCorrection.Jumping {
		t.Fatal("forged height granted jump")
	}
	if err := a.world.Send(proto.MotionPacket(proto.MotionJump)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	request(st.Self.X, original[1]+100, st.Self.Z)
	wait(func() bool { return st.LastPlayerCorrection.Jumping })
	if st.LastPlayerCorrection.Y > original[1]+1.6 || st.LastPlayerCorrection.Y <= original[1] {
		t.Fatal("invalid server jump", st.LastPlayerCorrection)
	}
	wait(func() bool { return !st.LastPlayerCorrection.Jumping })
	if math.Abs(float64(st.Self.Y-original[1])) > .15 {
		t.Fatal("idle client did not land", st.LastPlayerCorrection)
	}
	a.pumpWorld()
	t.Log("forged Y rejected; supported jump uses server arc and lands without movement input")
}
