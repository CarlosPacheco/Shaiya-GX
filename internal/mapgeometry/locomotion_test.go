package mapgeometry

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"math"
	"testing"
)

func testFloor(g *Geometry, y, x0, x1, z0, z1 float64) {
	g.Add(collision.Triangle{{x0, y, z0}, {x0, y, z1}, {x1, y, z0}})
	g.Add(collision.Triangle{{x1, y, z1}, {x1, y, z0}, {x0, y, z1}})
}
func TestGroundObstaclesAndNoHorizontalWallClimb(t *testing.T) {
	g := &Geometry{}
	g.Reindex()
	testFloor(g, 0, 0, 30, 0, 30)
	for _, tri := range collision.BlockerBox(collision.Vec{10, 0, 0}, collision.Vec{11, 10, 30}, 0) {
		g.Add(tri)
	}
	p := Position{8, FootSkin, 5}
	for range 200 {
		p = g.GroundMove(p, Position{.1, 0, 0}, false)
	}
	if p[0] > 9.8 || p[1] > .13 {
		t.Fatalf("climbed or crossed wall: %v", p)
	}
	p = Position{8, 1.2, 5}
	for range 200 {
		p = g.GroundMove(p, Position{.1, 0, 0}, true)
	}
	if p[0] > 9.8 || p[1] > 1.21 {
		t.Fatalf("airborne horizontal contact added height: %v", p)
	}
}
func TestTerrainRampRemainsTraversableBeforeSlopePressure(t *testing.T) {
	for _, slope := range []float32{.4, 2} {
		g := &Geometry{MapSize: 32, Heights: make([]uint16, 17*17)}
		g.Reindex()
		for z := 0; z < 17; z++ {
			for x := 0; x < 17; x++ {
				height := max(float32(0), float32(x*2)-8) * slope
				g.Heights[z*17+x] = uint16(10000 + height*50)
			}
		}
		p := Position{4, FootSkin, 8}
		for range 100 {
			p = g.GroundMove(p, Position{.1, 0, 0}, false)
		}
		if slope < 1 && p[0] < 12 {
			t.Fatalf("valid ramp blocked: %v", p)
		}
		if p[0] < 12 {
			t.Fatalf("terrain became an invisible wall: %v", p)
		}
	}
}
func TestStepAndLayeredEntry(t *testing.T) {
	g := &Geometry{}
	g.Reindex()
	testFloor(g, 0, 0, 20, 0, 20)
	for _, tri := range collision.BlockerBox(collision.Vec{5, 0, 0}, collision.Vec{10, .3, 20}, 0) {
		g.Add(tri)
	}
	p := Position{4, FootSkin, 5}
	for range 30 {
		p = g.GroundMove(p, Position{.1, 0, 0}, false)
	}
	if p[0] < 6 || math.Abs(float64(p[1]-.35)) > .05 {
		t.Fatalf("legal step blocked: %v", p)
	}
	testFloor(g, 5, 0, 20, 0, 20)
	upper, ok := g.Entry(Position{3, 5.1, 3})
	if !ok || math.Abs(float64(upper[1]-5.05)) > .01 {
		t.Fatalf("upper floor lost: %v %v", upper, ok)
	}
	lower, ok := g.Entry(Position{3, .1, 3})
	if !ok || lower[1] > .1 {
		t.Fatalf("snapped to highest floor: %v %v", lower, ok)
	}
}
