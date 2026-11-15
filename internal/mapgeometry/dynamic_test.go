package mapgeometry

import (
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
)

func TestDynamicBoundsPreserveRotatingContacts(t *testing.T) {
	for _, axis := range []world.Vec3{{0, 1, 0}, {1, 0, 0}, {1, 2, 3}} {
		d := Dynamic{Meshes: []world.CollisionMesh{{
			Vertices: []world.Vec3{{-5, -2, -1}, {5, -2, -1}, {5, 2, 1}, {-5, 2, 1}},
			Faces:    []world.Face{{0, 1, 2}, {0, 2, 3}},
		}}, Base: Basis(world.Vec3{8000, 30, -2000}, world.Vec3{1, 0, 1}, world.Vec3{0, 1, 0}), Axis: axis, Speed: .7}
		d.prepareBounds()
		for _, elapsed := range []float64{0, .01, .1, 1, 10, 100} {
			matrix := d.Transform(elapsed)
			queries := []collision.Box{{Min: collision.Vec{-1, -1, -1}, Max: collision.Vec{1, 1, 1}}}
			Visit(d.Meshes, matrix, func(tri collision.Triangle) {
				queries = append(queries, tri.Bounds())
				for _, v := range tri {
					queries = append(queries, collision.Box{Min: v, Max: v})
				}
			})
			for _, query := range queries {
				var want, got []collision.Triangle
				Visit(d.Meshes, matrix, func(tri collision.Triangle) {
					if query.Overlaps(tri.Bounds()) {
						want = append(want, tri)
					}
				})
				d.Visit(query, elapsed, func(tri collision.Triangle) { got = append(got, tri) })
				if !slices.Equal(got, want) {
					t.Fatalf("lost transformed contacts axis=%v time=%g bounds=%v", axis, elapsed, query)
				}
			}
		}
	}
}
