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

func TestEffectMVPMatchesGeneralProjection(t *testing.T) {
	s := &Scene3D{Proj: PerspectiveFovLH(.8, 1.5, .1, 300), TemporalJitter: [2]float32{.002, -.001}}
	eye := V3{63783, 51, -84238}
	s.SetCameraView(eye, eye.Add(V3{2, -.5, 5}), V3{0, 1, 0})
	rng := rand.New(rand.NewPCG(17, 21))
	check := func(world M4) {
		t.Helper()
		got := s.effectMVP(world, s.cameraValid && s.View == s.cameraView, sparseEffectPerspective(s.projection))
		want := s.modelView(world).Mul(s.projection)
		for i, v := range want {
			if got[i] != v && !(math.IsNaN(float64(got[i])) && math.IsNaN(float64(v))) {
				t.Fatalf("element %d: got %g, want %g for projection %v", i, got[i], v, s.projection)
			}
		}
	}
	for _, plane := range [][4]float32{{}, {0, 1, 0, 0}} {
		s.ClipPlane = plane
		for _, reverse := range []bool{false, true} {
			s.ReverseDepth = reverse
			s.projection = s.depthProjection(s.Proj)
			if !sparseEffectPerspective(s.projection) {
				t.Fatal("native projection missed the sparse path")
			}
			for range 100 {
				world := Identity()
				for row := range 3 {
					for col := range 3 {
						world[row*4+col] = rng.Float32()*4 - 2
					}
					world[12+row] = eye[row] + rng.Float32()*30 - 15
				}
				check(world)
			}
		}
	}
	nonfinite := Identity()
	nonfinite[0] = float32(math.Inf(1))
	check(nonfinite)
	s.View, s.projection = Identity(), Identity()
	if sparseEffectPerspective(s.projection) {
		t.Fatal("nonperspective projection entered the sparse path")
	}
	check(Identity())
	projective := s.Proj
	projective[4] = .25
	check(projective)
}
