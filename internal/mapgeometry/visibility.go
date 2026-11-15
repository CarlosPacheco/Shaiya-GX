package mapgeometry

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"math"
)

// LineOfSight tests actor centers through authored solids, dynamic geometry and
// native terrain fans. It is independent of rendering, LOD and navmesh floors.
func (g *Geometry) LineOfSight(start, end Position) bool {
	for i := range start {
		if math.IsNaN(float64(start[i])) || math.IsInf(float64(start[i]), 0) || math.IsNaN(float64(end[i])) || math.IsInf(float64(end[i]), 0) {
			return false
		}
	}
	a, b := cv(start), cv(end)
	// Native 00438010 adds 0.5 to logical actor origins; those are
	// 0.9 above the feet coordinates used by GX (006bc868).
	const rayHeight = .9 + .5
	a[1] += rayHeight
	b[1] += rayHeight
	direction := b.Sub(a)
	visible := true
	bounds := collision.Box{}
	for i := range a {
		bounds.Min[i] = min(a[i], b[i]) - .001
		bounds.Max[i] = max(a[i], b[i]) + .001
	}
	test := func(t collision.Triangle) {
		if !visible {
			return
		}
		e1, e2 := t[1].Sub(t[0]), t[2].Sub(t[0])
		h := direction.Cross(e2)
		det := e1.Dot(h)
		if math.Abs(det) < 1e-9 {
			return
		}
		inv := 1 / det
		s := a.Sub(t[0])
		u := inv * s.Dot(h)
		if u < 0 || u > 1 {
			return
		}
		q := s.Cross(e1)
		v := inv * direction.Dot(q)
		if v < 0 || u+v > 1 {
			return
		}
		at := inv * e2.Dot(q)
		if at > .00001 && at < .99999 {
			visible = false
		}
	}
	g.Visit(bounds, test)
	if visible {
		g.TerrainVisit(bounds, test)
	}
	return visible
}
