package scene

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func rayFloorWorld() *World {
	idx := collision.NewIndex()
	// Floor at y=0 over 0..100, a wall at x=50 from y=0 to 10.
	idx.Add(collision.Triangle{{0, 0, 0}, {100, 0, 0}, {0, 0, 100}})
	idx.Add(collision.Triangle{{100, 0, 0}, {100, 0, 100}, {0, 0, 100}})
	idx.Add(collision.Triangle{{50, 0, 0}, {50, 10, 0}, {50, 0, 100}})
	idx.Add(collision.Triangle{{50, 10, 0}, {50, 10, 100}, {50, 0, 100}})
	return &World{collision: idx}
}

func TestRayFloorHitsWalkableFloorNotWalls(t *testing.T) {
	w := rayFloorWorld()
	p, ok := w.RayFloor(render.V3{20, 10, 20}, render.V3{0, -1, 0}, 100)
	if !ok || p[1] != 0 || p[0] != 20 || p[2] != 20 {
		t.Fatalf("floor hit %v %v", p, ok)
	}
	// Looking along +x at head height hits the wall first: not a destination.
	if _, ok := w.RayFloor(render.V3{10, 5, 20}, render.V3{1, -.01, 0}, 100); ok {
		t.Fatal("wall accepted as a destination")
	}
	// A ray that misses everything, or is too short, finds nothing.
	if _, ok := w.RayFloor(render.V3{20, 10, 20}, render.V3{0, 1, 0}, 100); ok {
		t.Fatal("upward ray hit the floor")
	}
	if _, ok := w.RayFloor(render.V3{20, 10, 20}, render.V3{0, -1, 0}, 5); ok {
		t.Fatal("hit beyond the maximum distance")
	}
	var empty *World
	if _, ok := empty.RayFloor(render.V3{}, render.V3{0, -1, 0}, 10); ok {
		t.Fatal("nil world")
	}
}
