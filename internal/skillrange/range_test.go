package skillrange

import (
	"math"
	"testing"
)

func TestLaunchReleaseAndAreaBoundaries(t *testing.T) {
	a := Position{}
	launch := Reach(10, 0, false)
	release := Reach(10, 0, true)
	if launch != 12.3 || release != 14.8 {
		t.Fatal(launch, release)
	}
	for _, tc := range []struct {
		p    Position
		r    float32
		want bool
	}{
		{Position{12.3, 0, 0}, launch, true}, {Position{12.31, 0, 0}, launch, false},
		{Position{14.8, 0, 0}, release, true}, {Position{14.81, 0, 0}, release, false},
		{Position{0, 20, 0}, release, false}, {Position{100, 0, 0}, release, false},
		{Position{0, 5, 0}, 5, true}, {Position{0, 5.01, 0}, 5, false},
		{Position{float32(math.NaN()), 0, 0}, release, false},
	} {
		if got := Within(a, tc.p, tc.r); got != tc.want {
			t.Fatalf("point=%v radius=%g got=%v", tc.p, tc.r, got)
		}
	}
	if Reach(10, 3, false) != 15.3 {
		t.Fatal("mob size missing")
	}
}
