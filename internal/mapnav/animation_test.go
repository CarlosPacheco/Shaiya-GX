package mapnav

import (
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"os"
	"testing"
)

func TestCachedMap15RollersHaveNoPhysicalCollision(t *testing.T) {
	dir := os.Getenv("GX_TEST_GEOMETRY")
	if dir == "" {
		t.Skip("requires regenerated local map15 physics")
	}
	n, err := Load(dir, 15)
	if err != nil {
		t.Fatal(err)
	}
	g, ok := n.PlayerGeometry()
	if !ok {
		t.Fatal("player physics missing")
	}
	for _, z := range []float32{229.12186, 235.59166, 251.43314, 256.26212} {
		p, ok := g.Spawn(mapgeometry.Position{136.27, 6.6, z - 4})
		if !ok {
			t.Fatal("dungeon floor missing")
		}
		end, hit := g.MoveFeet(p, mapgeometry.Position{0, 0, 8})
		if hit || end[2] != p[2]+8 {
			t.Fatal("stale cached roller collider", p, end, hit)
		}
	}
}
