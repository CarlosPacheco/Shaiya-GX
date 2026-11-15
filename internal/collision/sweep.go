package collision

import "math"

// Radii and skin are the float constants in EP5 0x4383d0/0x437b60.
var PlayerRadii = Vec{float64(float32(.3)), float64(float32(.9)), float64(float32(.3))}

const Skin = float64(float32(.05))

type Contact struct {
	Hit      bool
	Distance float64
	Point    Vec
}

// lowestRoot follows 0x5a0080: strictly positive, strictly before maxRoot.
func lowestRoot(a, b, c, maxRoot float64) (float64, bool) {
	if math.Abs(a) < 1e-20 {
		if math.Abs(b) < 1e-20 {
			return 0, false
		}
		r := -c / b
		return r, r > 0 && r < maxRoot
	}
	determinant := b*b - 4*a*c
	if determinant < 0 {
		return 0, false
	}
	root := math.Sqrt(determinant)
	r1, r2 := (-b-root)/(2*a), (-b+root)/(2*a)
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	if r1 > 0 && r1 < maxRoot {
		return r1, true
	}
	return r2, r2 > 0 && r2 < maxRoot
}
func inside(p Vec, t Triangle) bool {
	a, b := t[1].Sub(t[0]), t[2].Sub(t[0])
	v := p.Sub(t[0])
	aa, ab, bb := a.Dot(a), a.Dot(b), b.Dot(b)
	denominator := aa*bb - ab*ab
	if denominator <= 0 {
		return false
	}
	u := (bb*v.Dot(a) - ab*v.Dot(b)) / denominator
	w := (aa*v.Dot(b) - ab*v.Dot(a)) / denominator
	return u >= 0 && w >= 0 && u+w <= 1
}

// SweepTriangle is the one-sided unit-sphere sweep from original 0x5a0ef0.
// Faces, then all three vertices and edges, share one nearest-hit interval.
func SweepTriangle(origin, velocity Vec, t Triangle) Contact {
	if !t.Valid() || !origin.Finite() || !velocity.Finite() {
		return Contact{}
	}
	speed := velocity.Len()
	if speed < 1e-15 {
		return Contact{}
	}
	normal := t[1].Sub(t[0]).Cross(t[2].Sub(t[0])).Norm()
	normalVelocity := normal.Dot(velocity)
	if normalVelocity > 0 {
		return Contact{}
	}
	distance := normal.Dot(origin.Sub(t[0]))
	embedded := false
	t0 := float64(0)
	if normalVelocity == 0 {
		if math.Abs(distance) >= 1 {
			return Contact{}
		}
		embedded = true
	} else {
		t0 = (-1 - distance) / normalVelocity
		t1 := (1 - distance) / normalVelocity
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		if t0 > 1 || t1 < 0 {
			return Contact{}
		}
		t0 = min(1, max(0, t0))
	}
	if !embedded {
		p := origin.Sub(normal).Add(velocity.Scale(t0))
		if inside(p, t) {
			return Contact{true, t0 * speed, p}
		}
	}
	best := float64(1)
	point := Vec{}
	hit := false
	speedSquared := velocity.Dot(velocity)
	for _, vertex := range t {
		difference := origin.Sub(vertex)
		if root, ok := lowestRoot(speedSquared, 2*velocity.Dot(difference), difference.Dot(difference)-1, best); ok {
			best, point, hit = root, vertex, true
		}
	}
	for i, a := range t {
		edge := t[(i+1)%3].Sub(a)
		base := a.Sub(origin)
		lengthSquared := edge.Dot(edge)
		if lengthSquared == 0 {
			continue
		}
		edgeVelocity, edgeBase := edge.Dot(velocity), edge.Dot(base)
		qa := lengthSquared*(-speedSquared) + edgeVelocity*edgeVelocity
		qb := lengthSquared*2*velocity.Dot(base) - 2*edgeVelocity*edgeBase
		qc := lengthSquared*(1-base.Dot(base)) + edgeBase*edgeBase
		if root, ok := lowestRoot(qa, qb, qc, best); ok {
			fraction := (edgeVelocity*root - edgeBase) / lengthSquared
			if fraction >= 0 && fraction <= 1 {
				best, point, hit = root, a.Add(edge.Scale(fraction)), true
			}
		}
	}
	return Contact{hit, best * speed, point}
}

// Visit enumerates original world-space triangles intersecting a broad-phase box.
type Visit func(Box, func(Triangle))

func slide(position, velocity, radii Vec, visit Visit, depth int, recurse bool) (Vec, bool) {
	if depth > 5 || velocity.Len() < 1e-12 {
		return position, false
	}
	nearest := Contact{}
	visit(SweptBox(position.Multiply(radii), velocity.Multiply(radii), radii), func(t Triangle) {
		for i := range t {
			t[i] = t[i].Divide(radii)
		}
		contact := SweepTriangle(position, velocity, t)
		if contact.Hit && (!nearest.Hit || contact.Distance < nearest.Distance) {
			nearest = contact
		}
	})
	if !nearest.Hit {
		return position.Add(velocity), false
	}
	destination := position.Add(velocity)
	next := position
	if nearest.Distance >= Skin {
		direction := velocity.Norm()
		next = position.Add(direction.Scale(nearest.Distance - Skin))
		nearest.Point = nearest.Point.Sub(direction.Scale(Skin))
	}
	normal := next.Sub(nearest.Point).Norm()
	projected := destination.Sub(normal.Scale(destination.Sub(nearest.Point).Dot(normal)))
	remainder := projected.Sub(nearest.Point)
	if recurse && remainder.Len() >= Skin {
		p, _ := slide(next, remainder, radii, visit, depth+1, recurse)
		return p, true
	}
	return next, true
}

// Move reproduces 0x4383d0: horizontal sliding (never downward), followed by
// a non-recursive vertical sweep. Positions are ellipsoid centres, not feet.
func Move(position, velocity Vec, visit Visit) (Vec, bool) {
	if !position.Finite() || !velocity.Finite() || visit == nil {
		return position, false
	}
	radii := PlayerRadii
	p := position.Divide(radii)
	delta := velocity.Divide(radii)
	horizontal := Vec{delta[0], 0, delta[2]}
	after, wall := slide(p, horizontal, radii, visit, 0, true)
	if after[1] < p[1] {
		after = p
	}
	after, floor := slide(after, Vec{0, delta[1], 0}, radii, visit, 0, false)
	return after.Multiply(radii), wall || floor
}
