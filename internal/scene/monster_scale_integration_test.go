//go:build integration

package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"os"
	"runtime"
	"testing"
)

func TestOriginalMonsterScalesGL(t *testing.T) {
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
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := NewAssets(arc)
	for _, tt := range []struct {
		id    uint16
		scale float32
	}{{108, .6}, {21, 1}, {3, 1.3}, {1, 1.6}, {8, 2}, {1627, 5}, {356, .4}, {2294, 1}} {
		m, err := assets.BuildMonster(tt.id)
		if err != nil {
			t.Fatal(err)
		}
		world := render.Basis(render.V3{10, 20, 30}, render.V3{0, 0, -1}, render.V3{0, 1, 0})
		m.SetTransform(world)
		m.Update(0)
		lo, hi, ok := m.WorldBounds()
		if !ok {
			t.Fatalf("mob %d has no bounds", tt.id)
		}
		anchor, ok := m.NameAnchor()
		if !ok {
			t.Fatalf("mob %d has no anchor", tt.id)
		}
		m.World = world // Independent unscaled posed model reference.
		baseLo, baseHi, ok := m.WorldBounds()
		if !ok {
			t.Fatal("no base bounds")
		}
		baseAnchor, _ := m.NameAnchor()
		for axis := range 3 {
			want := (baseHi[axis]-baseLo[axis]-.02)*tt.scale + .02
			if math.Abs(float64((hi[axis]-lo[axis])-want)) > 1e-3 {
				t.Fatalf("mob %d bounds axis %d not scaled", tt.id, axis)
			}
		}
		wantY := 20 + (baseAnchor[1]-20-.15)*tt.scale + .15
		if math.Abs(float64(anchor[1]-wantY)) > 1e-3 {
			t.Fatalf("mob %d anchor not scaled", tt.id)
		}
		t.Logf("mob=%d scale=%g height=%g", tt.id, tt.scale, hi[1]-lo[1])
	}
}
