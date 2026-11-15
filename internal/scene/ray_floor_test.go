package scene

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func rayFloorWorld() *World {
	idx := collision.NewIndex()
	// Floor at y=0 over 0..100, a wall at x=50 from y=0 to 10.
	idx.Add(collision.Triangle{{0, 0, 0}, {0, 0, 100}, {100, 0, 0}})
	idx.Add(collision.Triangle{{100, 0, 0}, {0, 0, 100}, {100, 0, 100}})
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

func TestRayFloorRejectsCeilingUndersidesAndBackFaces(t *testing.T) {
	index := collision.NewIndex()
	// A downward-facing ceiling underside, above a separate valid floor.
	index.Add(collision.Triangle{{0, 5, 0}, {100, 5, 0}, {0, 5, 100}})
	index.Add(collision.Triangle{{0, 0, 0}, {0, 0, 100}, {100, 0, 0}})
	w := &World{collision: index}
	if p, ok := w.RayFloor(render.V3{20, 2, 20}, render.V3{0, 1, 0}, 10); ok {
		t.Fatalf("ceiling underside accepted as walkable: %v", p)
	}
	// Even an upward-facing platform cannot be selected through its underside.
	index = collision.NewIndex()
	index.Add(collision.Triangle{{0, 5, 0}, {0, 5, 100}, {100, 5, 0}})
	w.collision = index
	if p, ok := w.RayFloor(render.V3{20, 2, 20}, render.V3{0, 1, 0}, 10); ok {
		t.Fatalf("platform backface accepted as walkable: %v", p)
	}
}

func TestRayFloorKeepsOcclusionAndFloorLayers(t *testing.T) {
	w := rayFloorWorld()
	// The floor behind the wall is in range, but must not be picked through it.
	if p, ok := w.RayFloor(render.V3{10, 5, 20}, render.V3{1, -.1, 0}, 100); ok {
		t.Fatal("picked floor through a wall", p)
	}
	index := collision.NewIndex()
	for _, y := range []float64{0, 5} {
		index.Add(collision.Triangle{{0, y, 0}, {0, y, 100}, {100, y, 0}})
	}
	w.collision = index
	if p, ok := w.RayFloor(render.V3{20, 10, 20}, render.V3{0, -2, 0}, 100); !ok || p != (render.V3{20, 5, 20}) {
		t.Fatal("picked wrong floor layer", p, ok)
	}
	for _, slope := range []float64{.5, 3} {
		index = collision.NewIndex()
		index.Add(collision.Triangle{{0, 0, 0}, {0, 0, 100}, {100, 100 * slope, 0}})
		w.collision = index
		_, ok := w.RayFloor(render.V3{10, 40, 10}, render.V3{0, -1, 0}, 100)
		if ok != (slope == .5) {
			t.Fatal("ramp slope classification", slope, ok)
		}
	}
	for _, query := range []struct {
		origin, direction render.V3
		distance          float32
	}{
		{render.V3{}, render.V3{}, 100},
		{render.V3{float32(math.NaN()), 0, 0}, render.V3{0, -1, 0}, 100},
		{render.V3{}, render.V3{0, -1, 0}, float32(math.Inf(1))},
	} {
		if _, ok := w.RayFloor(query.origin, query.direction, query.distance); ok {
			t.Fatal("invalid ray accepted", query)
		}
	}
}
