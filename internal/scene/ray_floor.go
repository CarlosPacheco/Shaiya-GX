package scene

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// minFloorNormalY accepts surfaces up to about 60 degrees as a destination;
// steeper hits (walls, pillars) are not ground to walk to.
const minFloorNormalY = .5

// RayFloor intersects a ray with the map's collision surfaces (dungeon floors,
// ramps, model platforms) and returns the nearest hit when it is walkable.
// Terrain and model triangles compete by distance along the same ray.
func (w *World) RayFloor(origin, direction render.V3, maxDistance float32) (render.V3, bool) {
	if w == nil || w.collision == nil || maxDistance <= 0 || math.IsNaN(float64(maxDistance)) || math.IsInf(float64(maxDistance), 0) {
		return render.V3{}, false
	}
	o, d := collisionVec(origin), collisionVec(direction).Norm()
	if !o.Finite() || !d.Finite() || d.Len() == 0 {
		return render.V3{}, false
	}
	limit := float64(maxDistance)
	best, walkable, hit := limit, false, false
	test := func(t collision.Triangle) {
		distance, normal, ok := rayTriangle(o, d, t)
		up := normal[1] >= minFloorNormalY && normal.Dot(d) < 0
		if ok && (distance < best || hit && distance == best && up) {
			best, hit, walkable = distance, true, up
		}
	}
	// Keep each query narrow along long diagonal rays. Stop as soon as the
	// closest hit falls inside the section just examined.
	for start := float64(0); start < limit; start += 16 {
		stop := min(start+16, limit)
		from, to := o.Add(d.Scale(start)), o.Add(d.Scale(stop))
		var box collision.Box
		for axis := range 3 {
			box.Min[axis] = min(from[axis], to[axis]) - .01
			box.Max[axis] = max(from[axis], to[axis]) + .01
		}
		w.visitCollisions(box, test)
		if w.physics != nil {
			w.physics.TerrainVisit(box, test)
		}
		if hit && best <= stop {
			break
		}
	}
	if !hit || !walkable {
		return render.V3{}, false
	}
	return renderVec(o.Add(d.Scale(best))), true
}

// rayTriangle is a two-sided Moller-Trumbore test; it also returns the
// signed unit normal. Backfaces still occlude, but are not walkable picks.
func rayTriangle(o, d collision.Vec, t collision.Triangle) (float64, collision.Vec, bool) {
	e1, e2 := t[1].Sub(t[0]), t[2].Sub(t[0])
	p := d.Cross(e2)
	det := e1.Dot(p)
	if math.Abs(det) < 1e-9 {
		return 0, collision.Vec{}, false
	}
	inv := 1 / det
	s := o.Sub(t[0])
	u := s.Dot(p) * inv
	if u < 0 || u > 1 {
		return 0, collision.Vec{}, false
	}
	q := s.Cross(e1)
	v := d.Dot(q) * inv
	if v < 0 || u+v > 1 {
		return 0, collision.Vec{}, false
	}
	distance := e2.Dot(q) * inv
	if distance <= 1e-4 {
		return 0, collision.Vec{}, false
	}
	n := e1.Cross(e2)
	length := n.Len()
	if length == 0 {
		return 0, collision.Vec{}, false
	}
	return distance, n.Scale(1 / length), true
}
