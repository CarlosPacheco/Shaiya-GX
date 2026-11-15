// Package collision implements the original EP5 ellipsoid/triangle movement
// queries without graphics, assets, networking or game-rule dependencies.
package collision

import "math"

type Vec [3]float64

func (a Vec) Add(b Vec) Vec       { return Vec{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a Vec) Sub(b Vec) Vec       { return Vec{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func (a Vec) Scale(s float64) Vec { return Vec{a[0] * s, a[1] * s, a[2] * s} }
func (a Vec) Dot(b Vec) float64   { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func (a Vec) Cross(b Vec) Vec {
	return Vec{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
func (a Vec) Len() float64 { return math.Sqrt(a.Dot(a)) }
func (a Vec) Norm() Vec {
	if length := a.Len(); length > 0 {
		return a.Scale(1 / length)
	}
	return Vec{}
}
func (a Vec) Divide(b Vec) Vec   { return Vec{a[0] / b[0], a[1] / b[1], a[2] / b[2]} }
func (a Vec) Multiply(b Vec) Vec { return Vec{a[0] * b[0], a[1] * b[1], a[2] * b[2]} }
func (a Vec) Finite() bool {
	for _, v := range a {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

type Triangle [3]Vec
type Box struct{ Min, Max Vec }

func (t Triangle) Bounds() Box {
	b := Box{t[0], t[0]}
	for _, p := range t[1:] {
		for a := range 3 {
			b.Min[a] = min(b.Min[a], p[a])
			b.Max[a] = max(b.Max[a], p[a])
		}
	}
	return b
}
func (b Box) Overlaps(c Box) bool {
	for a := range 3 {
		if b.Min[a] > c.Max[a] || c.Min[a] > b.Max[a] {
			return false
		}
	}
	return true
}
func SweptBox(p, v, r Vec) Box {
	end := p.Add(v)
	b := Box{}
	for a := range 3 {
		b.Min[a] = min(p[a], end[a]) - r[a]
		b.Max[a] = max(p[a], end[a]) + r[a]
	}
	return b
}
func (t Triangle) Valid() bool {
	return t[0].Finite() && t[1].Finite() && t[2].Finite() && t[1].Sub(t[0]).Cross(t[2].Sub(t[0])).Len() > 1e-12
}
