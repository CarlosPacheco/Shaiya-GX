//go:build integration

package app

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestOriginalDungeonMinimapsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and imported dungeon minimaps")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	if _, ok := arc.Lookup("DungeonMiniMaps.INI"); !ok {
		t.Skip("requires imported Core minimaps")
	}
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state = game.New()
	a.questDataLoaded = true
	for _, test := range []struct {
		id  uint16
		pos render.V3
	}{{3, render.V3{230, 13.286, 35}}, {15, render.V3{136.27, 6.69, 229.12186}}, {50, render.V3{50, 0, 50}}} {
		w, err := scene.LoadMap(a.assets, test.id)
		if err != nil {
			t.Fatal(err)
		}
		a.wv = &worldView{mapID: test.id, world: w, pos: test.pos, facing: render.V3{0, 0, 1}}
		a.state.MapID = test.id
		v := a.mapState()
		v.full = true
		a.loadMapImage(v)
		p := a.mapProjection(v)
		if test.id == 50 {
			if v.image != nil {
				t.Fatal("invented absent guild-interior map")
			}
		} else {
			if v.image == nil || v.image.W != 512 || v.image.H != 512 {
				t.Fatal("dungeon map did not load", test.id)
			}
			u, z, ok := p.Position(test.pos[0], test.pos[2])
			if !ok {
				t.Fatal("player missing from calibrated map", test.id, u, z)
			}
			ru, rv, ok := a.mapTeleportCoordinates(test.id, ui.UV{U1: 1, V1: 1}, u, z)
			size := a.minimapWorldSize(test.id)
			if !ok || math.Abs(float64(ru*size-test.pos[0])) > .0001 || math.Abs(float64((1-rv)*size-test.pos[2])) > .0001 {
				t.Fatal("map click moved to another coordinate", test.id, ru, rv)
			}
			t.Logf("map %d image 512x512; image=(%.6f,%.6f), world=(%.3f,%.3f), request=(%.6f,%.6f)", test.id, u, z, test.pos[0], test.pos[2], ru, rv)
		}
		for _, full := range []bool{false, true} {
			gl.Viewport(0, 0, 1024, 768)
			gl.ClearColor(0, 0, 0, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT)
			a.sprites.Begin(1024, 768)
			a.drawMinimap(1024, 768, full)
			a.sprites.End()
			if out := os.Getenv("GX_MINIMAP_SHOTS"); out != "" {
				if err := os.MkdirAll(out, 0700); err != nil {
					t.Fatal(err)
				}
				if err := render.SavePNG(filepath.Join(out, fmt.Sprintf("map-%d-full-%v.png", test.id, full)), 1024, 768); err != nil {
					t.Fatal(err)
				}
			}
		}
		if v.image != nil {
			v.image.Delete()
			v.image = nil
		}
		w.Close()
		a.wv = nil
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error: %#x", code)
	}
}
