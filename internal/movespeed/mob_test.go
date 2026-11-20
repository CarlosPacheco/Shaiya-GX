package movespeed

import (
	"math"
	"testing"
)

func TestMobUnitsPerSecond(t *testing.T) {
	if got := MobUnitsPerSecond(10, 2000, 2); math.Abs(got-4.7) > 1e-9 {
		t.Fatalf("%v", got)
	}
	if got := MobUnitsPerSecond(10, 2000, 4); math.Abs(got-6.7) > 1e-9 {
		t.Fatalf("%v", got)
	}
	if MobUnitsPerSecond(10, 0, 2) != 0 || MobUnitsPerSecond(1, 10000, 0) != 0.1 {
		t.Fatal("limits")
	}
}
