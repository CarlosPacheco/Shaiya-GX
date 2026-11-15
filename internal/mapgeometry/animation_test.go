package mapgeometry

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"os"
	"testing"
)

func TestMap15RollersAreVisualAndDungeonFloorIsPreserved(t *testing.T) {
	directory := os.Getenv("SHAIYA_DATA")
	if directory == "" {
		t.Skip("requires original map15 assets")
	}
	arc, err := archive.OpenDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("world/15.wld")
	if err != nil {
		t.Fatal(err)
	}
	layout, err := world.ParseWLD(raw)
	if err != nil {
		t.Fatal(err)
	}
	g, err := Load(arc.ReadFile, layout)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Dynamic) != 0 {
		t.Fatal("animation became a physical collider")
	}
	for _, z := range []float32{229.12186, 235.59166, 251.43314, 256.26212} {
		start, ok := g.Spawn(Position{136.27, 6.6, z - 4})
		if !ok || start[1] < 5 || start[1] > 8 {
			t.Fatal("native dungeon floor missing", start, ok)
		}
		end, hit := g.MoveFeet(start, Position{0, 0, 8})
		if hit || end[2] != start[2]+8 {
			t.Fatal("roller still blocks the corridor", z, start, end, hit)
		}
	}
}
