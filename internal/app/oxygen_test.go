package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestOxygenBarDrainsThenRefillsAndHides(t *testing.T) {
	for _, tc := range []struct {
		phase   proto.OxygenPhase
		ms      uint16
		elapsed float64
		want    float64
		visible bool
	}{
		{proto.OxygenHidden, 0, 0, 0, false},
		{proto.OxygenDraining, 30000, 0, 1, true},
		{proto.OxygenDraining, 30000, 15, .5, true},
		{proto.OxygenDraining, 30000, 45, 0, true},
		{proto.OxygenRefilling, 0, 1.5, .5, true},
		{proto.OxygenRefilling, 0, 3, 1, false},
	} {
		got, visible := oxygenFraction(proto.OxygenStatus{Phase: tc.phase, RemainingMS: tc.ms}, tc.elapsed)
		if got != tc.want || visible != tc.visible {
			t.Fatalf("oxygen %+v returned %v/%t", tc, got, visible)
		}
	}
}
