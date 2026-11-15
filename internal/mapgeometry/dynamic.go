package mapgeometry

import (
	"math"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
)

var physicsEpoch = time.Now()

func PhysicsSeconds() float64                            { return time.Since(physicsEpoch).Seconds() }
func (g *Geometry) SetDynamicClock(clock func() float64) { g.clock = clock }
func (d Dynamic) Transform(elapsed float64) Matrix {
	a := norm(d.Axis)
	if a == ([3]float32{}) || d.Speed == 0 {
		return d.Base
	}
	t := math.Mod(elapsed*float64(d.Speed)*30, 2*math.Pi)
	c, s := float32(math.Cos(t)), float32(math.Sin(t))
	x, y, z := a[0], a[1], a[2]
	v := 1 - c
	rotation := Matrix{c + x*x*v, x*y*v + z*s, x*z*v - y*s, 0, y*x*v - z*s, c + y*y*v, y*z*v + x*s, 0, z*x*v + y*s, z*y*v - x*s, c + z*z*v, 0, 0, 0, 0, 1}
	var out Matrix
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				out[i*4+j] += rotation[i*4+k] * d.Base[k*4+j]
			}
		}
	}
	return out
}

// prepareBounds retains immutable local bounds; queries still use the current
// synchronized transform, so rotation can never leave a stale collision box.
func (d *Dynamic) prepareBounds() {
	for _, mesh := range d.Meshes {
		for _, vertex := range mesh.Vertices {
			if d.bounds == nil {
				d.bounds = &world.Box{Min: vertex, Max: vertex}
			}
			for a := range 3 {
				d.bounds.Min[a] = min(d.bounds.Min[a], vertex[a])
				d.bounds.Max[a] = max(d.bounds.Max[a], vertex[a])
			}
		}
	}
}

// Visit rejects distant instances before transforming any mesh faces. Bounds
// only select candidates; the original authored triangles decide all contacts.
func (d Dynamic) Visit(query collision.Box, elapsed float64, emit func(collision.Triangle)) {
	matrix := d.Transform(elapsed)
	if d.bounds != nil {
		var bounds collision.Box
		for corner := 0; corner < 8; corner++ {
			p := d.bounds.Min
			for a := range 3 {
				if corner&(1<<a) != 0 {
					p[a] = d.bounds.Max[a]
				}
			}
			point := Point(p, matrix)
			if corner == 0 {
				bounds = collision.Box{Min: point, Max: point}
			}
			for a := range 3 {
				bounds.Min[a] = min(bounds.Min[a], point[a])
				bounds.Max[a] = max(bounds.Max[a], point[a])
			}
		}
		// Outward padding covers float32 matrix rounding, including large maps.
		for a := range 3 {
			margin := .001 + max(math.Abs(bounds.Min[a]), math.Abs(bounds.Max[a]))*1e-6
			bounds.Min[a] -= margin
			bounds.Max[a] += margin
		}
		if !query.Overlaps(bounds) {
			return
		}
	}
	Visit(d.Meshes, matrix, func(t collision.Triangle) {
		if query.Overlaps(t.Bounds()) {
			emit(t)
		}
	})
}
