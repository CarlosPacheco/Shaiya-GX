//go:build integration

package bots

import (
	"math/rand/v2"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestLocalBotsRealAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires OpenGL and client data")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "Local PvP bots", Width: 320, Height: 200, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	world, err := scene.LoadMap(scene.NewAssets(arc), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer world.Close()
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	skin, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer skin.Delete()

	origin := render.V3{996.458, 0, 787.152}
	origin[1] = world.Height(origin[0], origin[2])
	forward := render.V3{0, 0, 1}
	a := New(arc, world, origin, forward, 24)
	a.rng = rand.New(rand.NewPCG(23, 67))
	defer a.Close()
	s.SetCameraView(origin.Add(render.V3{0, 6, -16}), origin.Add(render.V3{0, 1, 12}), up)
	s.Proj = render.PerspectiveFovLH(.9, 1.6, .1, 180)
	gl.Viewport(0, 0, 320, 200)
	var hit, death, revival, fx, draw bool
	killed := make(map[int]bool)
	for frame := range 720 {
		a.Update(1.0/30, true)
		for i, f := range a.actors {
			hit = hit || f.hp < maxHP
			death = death || f.hp == 0
			if f.hp == 0 {
				killed[i] = true
			} else if killed[i] && f.hp == maxHP {
				revival = true
			}
		}
		fx = fx || len(a.effects) > 0
		if frame%10 == 0 {
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			render.ResetDrawCalls()
			a.Draw(s, skin, true)
			draw = draw || render.DrawCalls() > 0
		}
	}
	if a.Count() != 24 || !hit || !death || !revival || !fx || !draw {
		t.Fatalf("count=%d hit=%v death=%v revival=%v fx=%v draw=%v", a.Count(), hit, death, revival, fx, draw)
	}
	looks := map[scene.Appearance]bool{}
	for _, f := range a.actors {
		if f.avatar == nil || f.loadout.look.WeaponType == 0 || !f.avatar.HasCloak() {
			t.Fatalf("%s lacks a weapon or cloth, appearance=%+v", f.name, f.loadout.look)
		}
		looks[f.loadout.look] = true
	}
	if len(looks) < 12 {
		t.Fatalf("only %d distinct outfits", len(looks))
	}
	a.Close()
	before := render.GPUBytes()
	a = New(arc, world, origin, forward, MaxCount)
	defer a.Close()
	a.rng = rand.New(rand.NewPCG(41, 3))
	for range 120 {
		if a.Update(0, false) {
			break
		}
	}
	if a.Count() != MaxCount {
		t.Fatalf("spawned %d of %d", a.Count(), MaxCount)
	}
	for _, f := range a.actors {
		if f.avatar == nil || !f.avatar.HasCloak() || f.loadout.look.WeaponType == 0 {
			t.Fatalf("max-count spawn: %s has an incomplete avatar", f.name)
		}
	}
	a.Update(1.0/30, true)
	beforeDraw := int64(render.GPUBytes())
	s.SetCameraView(origin.Add(render.V3{0, 34, -25}), origin.Add(render.V3{0, 1, 45}), up)
	render.ResetDrawCalls()
	a.Draw(s, skin, true)
	if calls := render.DrawCalls(); calls < MaxCount {
		t.Fatalf("expected a populated render at maximum count, got %d draw calls", calls)
	}
	rendererDelta := int64(render.GPUBytes()) - beforeDraw
	a.Close()
	if got := int64(render.GPUBytes()) - rendererDelta; got != int64(before) {
		t.Fatalf("bot resources: %d bytes before arena, %d after", before, got)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("OpenGL error %#x", code)
	}
}
