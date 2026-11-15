package collision

import (
	"math"
	"testing"
)

func TestWallSlidingCornerFloorAndHighSpeed(t *testing.T) {
	index := NewIndex()
	index.Add(Triangle{{0, -20, -20}, {0, 0, 20}, {0, 20, -20}})
	index.Add(Triangle{{-20, -20, 0}, {-20, 20, 0}, {20, 0, 0}}) // normal -Z
	index.Add(Triangle{{-20, 0, -20}, {0, 0, 20}, {20, 0, -20}})
	for _, velocity := range []Vec{{6, 0, 0}, {600, 0, 0}, {6, 0, 5}} {
		result, hit := Move(Vec{-3, 2, -3}, velocity, index.Visit)
		if !hit || result[0] > -.29 || result[2] > .001 {
			t.Fatal("solid wall/corner was crossed", velocity, result)
		}
	}
	result, hit := Move(Vec{-3, 3, -3}, Vec{0, -20, 0}, index.Visit)
	if !hit || math.Abs(result[1]-.945) > 1e-5 {
		t.Fatal("native ellipsoid floor clearance", result)
	}
}
func TestIndexBoundsAndIndependentQueries(t *testing.T) {
	index := NewIndex()
	triangle := Triangle{{-25, 0, -25}, {0, 0, 25}, {25, 0, -25}}
	index.Add(triangle)
	count := 0
	index.Visit(Box{Vec{-40, -1, -40}, Vec{40, 1, 40}}, func(Triangle) { count++ })
	if count != 1 {
		t.Fatal("duplicate cell visits", count)
	}
	index.Visit(Box{Vec{100, -1, 100}, Vec{110, 1, 110}}, func(Triangle) { t.Fatal("unrelated triangle returned") })
	index.Add(Triangle{})
	index.Add(Triangle{{math.NaN(), 0, 0}, {0, 1, 0}, {0, 0, 1}})
	if index.Len() != 1 {
		t.Fatal("invalid triangles retained")
	}
}

func TestAuthoredBlockerYawAndOneSidedFaces(t *testing.T) {
	for _, yaw := range []float64{0, math.Pi / 6, math.Pi / 2} {
		triangles := BlockerBox(Vec{10, 0, 10}, Vec{12, 4, 12}, yaw)
		index := NewIndex()
		for _, triangle := range triangles {
			index.Add(triangle)
		}
		c, s := math.Cos(yaw), math.Sin(yaw)
		transform := func(v Vec) Vec { return Vec{10 + v[0]*c + v[2]*s, v[1], 10 - v[0]*s + v[2]*c} }
		// Sweep inward through every vertical face. Each original winding must
		// block from outside; the volume remains rotated around its first corner.
		for _, pair := range [][2]Vec{{{-3, 2, 1}, {1, 2, 1}}, {{5, 2, 1}, {1, 2, 1}}, {{1, 2, -3}, {1, 2, 1}}, {{1, 2, 5}, {1, 2, 1}}} {
			start, end := transform(pair[0]), transform(pair[1])
			position, hit := Move(start, end.Sub(start), index.Visit)
			if !hit || position.Sub(end).Len() < .2 {
				t.Fatalf("authored blocker missed yaw=%f start=%v end=%v got=%v", yaw, start, end, position)
			}
		}
	}
}
