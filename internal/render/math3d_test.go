package render

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestMulAffineMatchesFullProduct(t *testing.T) {
	check := func(a, b M4) {
		t.Helper()
		got, want := a.MulAffine(b), a.Mul(b)
		for i, v := range want {
			if got[i] != v && !(math.IsNaN(float64(got[i])) && math.IsNaN(float64(v))) {
				t.Fatalf("element %d: got %g, want %g for %v × %v", i, got[i], v, a, b)
			}
		}
	}
	rng := rand.New(rand.NewPCG(17, 21))
	matrix := func() M4 {
		m := Identity()
		for row := range 4 {
			for col := range 3 {
				m[4*row+col] = rng.Float32()*16 - 8
			}
		}
		return m
	}
	for range 1000 {
		check(matrix(), matrix())
	}
	a, b := matrix(), matrix()
	projection := PerspectiveFovLH(.8, 1.5, .1, 300)
	check(a, projection)
	check(projection, b)
	a[4] = float32(math.Inf(1))
	b[13] = float32(math.NaN())
	check(a, Identity())
	check(Identity(), b)
	a, b = Identity(), Identity()
	a[0], b[0] = math.MaxFloat32, 2
	check(a, b)
}
