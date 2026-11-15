package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func TestNativeGroundPickupBillboard(t *testing.T) {
	for _, right := range []render.V3{{1, 0, 0}, {0, 0, 1}, {.6, 0, .8}} {
		normal := (render.V3{0, 1, 0}).Cross(right)
		for _, tc := range []struct {
			x, y float32
			want bool
		}{
			{0, .8, true}, {.29, .99, true}, {-.29, .1, true}, {.31, .5, false}, {0, 1.01, false}, {0, -.01, false},
		} {
			target := right.Scale(tc.x).Add(render.V3{0, tc.y, 0})
			origin := target.Add(normal.Scale(3))
			d, hit := groundPickupIntersection(origin, normal.Scale(-1), right, render.V3{})
			if hit != tc.want || hit && math.Abs(float64(d-3)) > 1e-6 {
				t.Fatalf("right=%v point=%v: hit=%v distance=%g", right, target, hit, d)
			}
		}
		if _, ok := groundPickupIntersection(normal.Scale(3), normal, right, render.V3{}); ok {
			t.Fatal("picked behind camera")
		}
		if _, ok := groundPickupIntersection(normal.Scale(3), right, right, render.V3{}); ok {
			t.Fatal("parallel ray picked item")
		}
	}
}
