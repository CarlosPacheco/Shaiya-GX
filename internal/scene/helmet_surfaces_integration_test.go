//go:build integration

package scene

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestOriginalFuryHelmetSurfacesGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const viewport = 384
	win, err := platform.Open(platform.Config{Width: viewport, Height: viewport, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := NewAssets(arc)
	r, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Delete()
	skin, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer skin.Delete()
	batch := &CharacterBatch{}
	defer batch.Delete()

	r.Proj = render.PerspectiveFovLH(.785398, 1, .1, 50)

	r.CharacterAmbient, r.LightDir = .6, render.V3{0, -.3, 1}

	target, err := render.NewWaterTarget(viewport, viewport)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Delete()
	draw := func(actor *Character, crowd bool, filename string) []byte {
		pixels := make([]byte, viewport*viewport*4)
		target.Capture(render.V3{.12, .12, .12}, func() {
			r.Begin()
			if crowd {
				actors := make([]*Character, MinCrowdActors)
				for i := range actors {
					actors[i] = actor
				}
				batch.Draw(skin, r, actors)
				if batch.Stats.Skin == 0 || batch.Stats.Fallback != 0 {
					t.Fatalf("helmet did not reach the batched path: %+v", batch.Stats)
				}
			} else {
				actor.Draw(skin, r)
			}
			r.End()
			gl.ReadPixels(0, 0, viewport, viewport, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
			if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" && filename != "" {
				if err := render.SavePNG(filepath.Join(dir, filename), viewport, viewport); err != nil {
					t.Error(err)
				}
			}
		})
		return pixels
	}
	for _, model := range []struct {
		name      string
		race, job uint8
	}{{"Teneran-Vile", 2, 2}, {"Teneran-DeathEater", 3, 3}, {"Ohgerton", 3, 1}, {"Treffers", 2, 4}} {
		t.Log(model.name)
		actor, err := BuildCharacter(assets, Appearance{Race: model.race, Job: model.job, Helmet: 11, ShowHelmet: true, Upper: 21, Lower: 21, Hand: 21, Foot: 21}, "000_normal")
		if err != nil {
			t.Fatal(err)
		}
		actor.World = render.Identity()
		helmet := actor.parts[len(actor.parts)-1]
		if helmet.tex == nil || !helmet.twoSided {
			t.Fatalf("helmet lacks its texture or two-sided material: %v", assets.Missing)
		}
		for _, view := range []struct {
			name string
			eye  render.V3
		}{
			{"front", render.V3{.35, 1.95, -1.65}},
			{"side", render.V3{1.65, 1.85, .3}},
			{"back", render.V3{.45, 2.15, 1.65}},
		} {
			r.SetCameraView(view.eye, render.V3{0, 1.75, 0}, render.V3{0, 1, 0})
			base := model.name + "-" + view.name
			helmet.twoSided = false
			old := draw(actor, false, base+"-cull.png")
			helmet.twoSided = true
			current := draw(actor, false, base+"-two.png")
			count := 0
			for i := 0; i < len(old); i += 4 {
				if !bytes.Equal(old[i:i+3], current[i:i+3]) {
					count++
				}
			}
			if count == 0 {
				t.Fatalf("%s: reverse helmet surfaces did not appear", base)
			}
			crowd := draw(actor, true, "")
			for i, pixel := range crowd {
				if a, b := int(pixel), int(current[i]); a-b < -1 || a-b > 1 {
					t.Fatalf("batched and direct skin disagree at byte %d: %d vs %d", i, a, b)
				}
			}
		}
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("GL error %#x", code)
		}
	}
}
