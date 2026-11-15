package mapgeometry

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"math"
)

type groundSupport struct {
	surface, feet float32
}

// modelSupport uses the same ellipsoid as movement. At a tread's edge the
// supporting point can be above the feet; a point-height test loses that contact.
// Only authored upward-facing triangles may support the character, never walls.
func (g *Geometry) modelSupport(p Position, drop float32) (groundSupport, bool) {
	radii := collision.PlayerRadii
	center := cv(p)
	center[1] += radii[1]
	delta := collision.Vec{0, -float64(drop), 0}
	origin, velocity := center.Divide(radii), delta.Divide(radii)
	nearest := collision.Contact{}
	g.Visit(collision.SweptBox(center, delta, radii), func(tri collision.Triangle) {
		normal := tri[1].Sub(tri[0]).Cross(tri[2].Sub(tri[0])).Norm()
		if normal[1] < math.Cos(MaxGroundSlope*math.Pi/180) || normal.Dot(center.Sub(tri[0])) < 0 {
			return
		}
		scaled := tri
		for i := range scaled {
			scaled[i] = scaled[i].Divide(radii)
		}
		hit := collision.SweepTriangle(origin, velocity, scaled)
		if !hit.Hit {
			return
		}
		// Native embedded-face sweeps can report the sphere bottom rather than a
		// point on the triangle. Such overlap is not evidence of physical support.
		point := hit.Point.Multiply(radii)
		if math.Abs(normal.Dot(point.Sub(tri[0]))) > .001 {
			return
		}
		at := origin.Add(velocity.Norm().Scale(hit.Distance))
		if at[1] <= hit.Point[1]+.001 {
			return
		}
		if !nearest.Hit || hit.Distance < nearest.Distance {
			nearest = hit
		}
	})
	if !nearest.Hit {
		return groundSupport{}, false
	}
	return groundSupport{surface: float32(nearest.Point[1] * radii[1]), feet: p[1] - float32(nearest.Distance*radii[1]) + FootSkin}, true
}
func (g *Geometry) groundSupport(p Position, drop float32) (groundSupport, bool) {
	support, ok := g.modelSupport(p, drop)
	if g.HasTerrain() && p[0] >= 0 && p[2] >= 0 && p[0] <= float32(g.MapSize) && p[2] <= float32(g.MapSize) {
		h := g.Height(p[0], p[2])
		if p[1] >= h-.01 && p[1]-h <= drop && (!ok || support.feet < h) {
			return groundSupport{surface: h, feet: h}, true
		}
	}
	return support, ok
}
func (g *Geometry) Supported(p Position) bool { _, ok := g.groundSupport(p, .13); return ok }

func (g *Geometry) StandingPosition(p Position) Position {
	if support, ok := g.groundSupport(p, .13); ok && math.Abs(float64(support.feet-p[1])) <= .131 {
		candidate := p
		candidate[1] = support.feet
		if candidate[1] <= p[1]+.0001 || g.clearBody(candidate) {
			p = candidate
		}
	}
	return p
}

// clearBody guards the native edge-slide result against overhead penetration.
// It respects one-sided authored faces and tests the full ellipsoid, not feet.
func (g *Geometry) clearBody(p Position) bool {
	radii := collision.PlayerRadii
	center := cv(p)
	center[1] += radii[1]
	origin := center.Divide(radii)
	clear := true
	g.Visit(collision.SweptBox(center, collision.Vec{}, radii), func(tri collision.Triangle) {
		if !clear {
			return
		}
		for i := range tri {
			tri[i] = tri[i].Divide(radii)
		}
		normal := tri[1].Sub(tri[0]).Cross(tri[2].Sub(tri[0]))
		if normal.Dot(origin.Sub(tri[0])) < -.00001 {
			return
		}
		closest := closestTrianglePoint(origin, tri)
		difference := origin.Sub(closest)
		if difference.Dot(difference) < .998001 {
			clear = false
		}
	})
	return clear
}

func closestTrianglePoint(p collision.Vec, t collision.Triangle) collision.Vec {
	a, b, c := t[0], t[1], t[2]
	ab, ac, ap := b.Sub(a), c.Sub(a), p.Sub(a)
	d1, d2 := ab.Dot(ap), ac.Dot(ap)
	if d1 <= 0 && d2 <= 0 {
		return a
	}
	bp := p.Sub(b)
	d3, d4 := ab.Dot(bp), ac.Dot(bp)
	if d3 >= 0 && d4 <= d3 {
		return b
	}
	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		return a.Add(ab.Scale(d1 / (d1 - d3)))
	}
	cp := p.Sub(c)
	d5, d6 := ab.Dot(cp), ac.Dot(cp)
	if d6 >= 0 && d5 <= d6 {
		return c
	}
	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		return a.Add(ac.Scale(d2 / (d2 - d6)))
	}
	va := d3*d6 - d5*d4
	if va <= 0 && d4-d3 >= 0 && d5-d6 >= 0 {
		return b.Add(c.Sub(b).Scale((d4 - d3) / ((d4 - d3) + (d5 - d6))))
	}
	inv := 1 / (va + vb + vc)
	return a.Add(ab.Scale(vb * inv)).Add(ac.Scale(vc * inv))
}
