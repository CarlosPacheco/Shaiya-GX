package mapgeometry

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"testing"
)

func TestSpellLineOfSightSolidsFloorsTerrainAndDynamic(t *testing.T) {
	makeGeometry := func() *Geometry { g := &Geometry{}; g.Reindex(); return g }
	a, b := Position{0, 0, 0}, Position{10, 0, 0}
	g := makeGeometry()
	if !g.LineOfSight(a, b) {
		t.Fatal("empty scene blocked")
	}
	for _, tri := range collision.BlockerBox(collision.Vec{4, -1, -2}, collision.Vec{5, 3, 2}, 0) {
		g.Add(tri)
	}
	if g.LineOfSight(a, b) || g.LineOfSight(b, a) {
		t.Fatal("wall is not two-sided")
	}
	g = makeGeometry()
	g.Add(collision.Triangle{{-5, 2, -5}, {5, 2, -5}, {0, 2, 5}})
	if g.LineOfSight(Position{0, 0, 0}, Position{0, 4, 0}) {
		t.Fatal("intervening floor ignored")
	}
	g = makeGeometry()
	g.MapSize = 16
	g.Heights = make([]uint16, 9*9)
	for i := range g.Heights {
		g.Heights[i] = 10000
	}
	g.Heights[2*9+2] = 15000
	if g.LineOfSight(Position{0, 0, 4}, Position{8, 0, 4}) {
		t.Fatal("native terrain hill ignored")
	}
	g = makeGeometry()
	g.Dynamic = []Dynamic{{Base: Identity(), Meshes: []world.CollisionMesh{{Vertices: []world.Vec3{{5, -1, -2}, {5, 3, -2}, {5, 0, 2}}, Faces: []world.Face{{0, 1, 2}}}}}}
	if g.LineOfSight(a, b) {
		t.Fatal("dynamic collider ignored")
	}
	g.Dynamic[0].Base[12] = 20
	if !g.LineOfSight(a, b) {
		t.Fatal("dynamic transform ignored")
	}
}

func TestSpellRayClearsLowObjectsButPreservesTallSolids(t *testing.T) {
	for _, height := range []float64{.5, 1.2, 1.39, 1.41, 3} {
		g := &Geometry{}
		g.Reindex()
		for _, triangle := range collision.BlockerBox(collision.Vec{4, 0, -2}, collision.Vec{5, height, 2}, 0) {
			g.Add(triangle)
		}
		for _, ends := range [][2]Position{{{0, 0, 0}, {10, 0, 0}}, {{10, 0, 0}, {0, 0, 0}}} {
			if got := g.LineOfSight(ends[0], ends[1]); got != (height < 1.4) {
				t.Fatalf("object height %g: visible=%v", height, got)
			}
		}
	}
}

func TestSegmentVisibleUsesCameraHeight(t *testing.T) {
	g := &Geometry{}
	g.Reindex()
	for _, triangle := range collision.BlockerBox(collision.Vec{4, 0, -2}, collision.Vec{5, 1, 2}, 0) {
		g.Add(triangle)
	}
	start, end := Position{0, .5, 0}, Position{10, .5, 0}
	if g.SegmentVisible(start, end) || !g.LineOfSight(start, end) {
		t.Fatal("camera ray should hit the obstacle below the raised spell ray")
	}
}
