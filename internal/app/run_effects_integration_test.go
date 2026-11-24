//go:build integration

package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestOriginalRunTrailGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 512, Height: 512, Hidden: true, Debug: true})
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
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	v := a.wv
	v.moving = true
	a.state.Self.Concealed = false
	a.state.Self.Speed = &proto.CharacterSpeed{ID: a.state.SelfID, Move: 3}
	capture := func() []byte {
		gl.Viewport(0, 0, 512, 512)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.scene3d.View = render.LookAtLH(v.pos.Add(render.V3{3, 2, -5}), v.pos.Add(render.V3{0, .6, 0}), render.V3{0, 1, 0})
		a.scene3d.Proj = render.PerspectiveFovLH(.8, 1, .1, 100)
		a.scene3d.Begin()
		a.drawRunEffects()
		pixels := make([]byte, 512*512*4)
		gl.ReadPixels(0, 0, 512, 512, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	baseline := capture()
	for _, faction := range []uint8{0, 1} {
		a.state.Faction.Faction = faction
		v.runEffects = nil
		v.runEffectAt = nil
		for range 60 {
			a.clock += 1.0 / 60
			a.frameDT = 1.0 / 60
			a.updateRunEffects()
		}
		if len(v.runEffects) < 3 || len(v.runEffects) > 11 {
			t.Fatal("original cadence/lifetime", len(v.runEffects))
		}
		for _, e := range v.runEffects {
			if e.instance.Origin != v.pos {
				t.Fatal("Crun detached from native feet origin", e.instance.Origin, v.pos)
			}
			if e.instance.Skipped != 0 {
				t.Fatal("unsupported Crun emitters", e.instance.Skipped)
			}
		}
		if bytes.Equal(baseline, capture()) {
			t.Fatal("original Crun produced no pixels", faction)
		}
		if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
			if err := render.SavePNG(filepath.Join(dir, "crun-"+string(rune('0'+faction))+".png"), 512, 512); err != nil {
				t.Fatal(err)
			}
		}
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		a.frameDT = 0
		for i := 0; i < 8 || !a.SceneReady(); i++ {
			if i >= 240 {
				t.Fatal("world residency did not complete")
			}
			gl.Viewport(0, 0, 512, 512)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			a.drawWorldView(512, 512)
		}
		if err := render.SavePNG(filepath.Join(dir, "crun-world.png"), 512, 512); err != nil {
			t.Fatal(err)
		}
	}
	// Remote users use the same confirmed state and emit at their world position.
	a.state.Entities[991090] = &game.Entity{ID: 991090, Kind: game.KindUser, Motion: 1, Shape: &proto.UserShape{Faction: 1}, Speed: &proto.CharacterSpeed{Move: 4}, X: v.pos[0] + 2, Y: v.pos[1], Z: v.pos[2]}
	if v.actors == nil {
		v.actors = map[uint32]*remoteActor{}
	}
	v.actors[991090] = &remoteActor{character: v.self, facing: v.facing}
	a.clock += .2
	a.updateRunEffects()
	found := false
	for _, e := range v.runEffects {
		if e.source == 991090 {
			found = true
		}
	}
	if !found {
		t.Fatal("remote haste missing")
	}
	a.state.Entities[991090].Concealed = true
	v.moving = false
	for range 300 {
		a.clock += 1.0 / 60
		a.frameDT = 1.0 / 60
		a.updateRunEffects()
	}
	if len(v.runEffects) != 0 {
		t.Fatal("stopped/hidden trail did not expire", len(v.runEffects))
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("Crun GL error %#x", err)
	}
}

func TestSonicNativeItemData(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires client data")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("Item/Item.SData")
	if err != nil {
		t.Fatal(err)
	}
	defs, err := sdata.ParseItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[uint8]uint8{106: 1, 107: 2} {
		if d := defs[sdata.ItemKey{Type: 30, TypeID: id}]; d == nil || d.Speed != want {
			t.Fatalf("client Sonic %d speed mismatch: %+v", id, d)
		}
	}
}
