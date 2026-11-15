//go:build integration

package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"os"
	"runtime"
	"testing"
)

func TestOriginalLadderObjectsRemainVisibleAndNonSolid(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("original assets and GL")
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
	assets := NewAssets(arc)
	for _, file := range []string{"world/1.wld", "world/19.wld", "world/55.wld"} {
		raw, err := arc.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		layout, err := world.ParseWLD(raw)
		if err != nil {
			t.Fatal(err)
		}
		only := &world.WLD{Objects: layout.Objects}
		visual := &World{WLD: only}
		visual.addObjects(assets)
		if len(visual.Objects) != len(layout.Objects.Instances) {
			t.Fatal(file, "ladder instances lost", len(visual.Objects), len(layout.Objects.Instances))
		}
		for _, obj := range visual.Objects {
			for _, part := range obj.Model.Parts {
				if part.Mesh.CullBack {
					t.Fatal(file, "ladder disappears from reverse side")
				}
			}
		}
		g, err := mapgeometry.Load(arc.ReadFile, only)
		if err != nil {
			t.Fatal(err)
		}
		if len(g.Triangles) != 0 {
			t.Fatal("ladders unexpectedly became solid")
		}
	}
}
