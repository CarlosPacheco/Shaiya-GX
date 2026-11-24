//go:build integration

package scene

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestNormalizedRendCreaturesVisibleGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and normalized original creature assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 320, Height: 320, Hidden: true, Debug: true})
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
	assets := NewAssets(arc)
	renderer, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer renderer.Delete()
	skin, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer skin.Delete()
	renderer.Ambient = 1
	renderer.Proj = render.PerspectiveFovLH(math.Pi/4, 1, .05, 1000)
	// All original monster IDs referencing MON visuals5/40/199–206, not only
	// Cloron Ogre2045 and Guard2046. Each retains its MON material assignment.
	ids := []uint16{14, 32, 144, 566, 620, 625, 673, 924, 981, 1340, 1350, 2040, 2041, 2045, 2046, 2054, 2262, 2327, 2662, 2663, 2664, 2665, 2817, 2826}
	for _, id := range ids {
		mob, err := assets.BuildMonster(id)
		if err != nil && strings.Contains(err.Error(), "out of range") {
			t.Logf("mob%d is not in this client's Monster.SData; skipped", id) // a smaller catalog than the one the list was written for
			continue
		}
		if err != nil {
			t.Fatalf("mob%d: %v", id, err)
		}
		mob.World = render.Identity()
		minimum := 320 * 320
		authored, poses := 0, 0
		for action := range 9 {
			if !strings.EqualFold(filepath.Ext(mob.visual.Animations[action]), ".ani") {
				continue
			}
			authored++
			if err := mob.SetAction(action); err != nil {
				t.Fatalf("mob%d action%d: %v", id, action, err)
			}
			mob.Update(.12)
			if action != 0 && action != 2 && action != 5 && action != 6 {
				continue
			}
			poses++
			lo, hi, ok := mob.WorldBounds()
			if !ok {
				t.Fatalf("mob%d has no visible bounds", id)
			}
			center := lo.Add(hi).Scale(.5)
			radius := max(1, hi.Sub(lo).Len()*.5)
			renderer.View = render.LookAtLH(center.Add(render.V3{0, radius * .1, -radius * 2.8}), center, render.V3{0, 1, 0})
			gl.Viewport(0, 0, 320, 320)
			gl.DepthMask(true)
			gl.ClearColor(0, 0, 0, 1)
			gl.ClearDepth(1)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			renderer.Begin()
			mob.Draw(skin, renderer)
			renderer.End()
			pixels := make([]byte, 320*320*4)
			gl.ReadPixels(0, 0, 320, 320, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
			visible := 0
			for i := 0; i < len(pixels); i += 4 {
				if pixels[i] != 0 || pixels[i+1] != 0 || pixels[i+2] != 0 {
					visible++
				}
			}
			if visible < 100 {
				t.Fatalf("mob%d action%d invisible: %d pixels", id, action, visible)
			}
			minimum = min(minimum, visible)
			if action == 6 && (id == 2045 || id == 2046) {
				if out := os.Getenv("GX_CREATURE_SHOTS"); out != "" {
					if err := os.MkdirAll(out, 0700); err != nil {
						t.Fatal(err)
					}
					if err := render.SavePNG(filepath.Join(out, fmt.Sprintf("mob-%d.png", id)), 320, 320); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		if poses < 3 {
			t.Fatalf("mob%d did not exercise enough authored poses: %d", id, poses)
		}
		t.Logf("mob%d %s: %d authored actions load; %d poses visible, minimum%d pixels", id, mob.Name, authored, poses, minimum)
	}
	if len(assets.Missing) != 0 {
		t.Fatalf("missing creature resources: %v", assets.Missing)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
