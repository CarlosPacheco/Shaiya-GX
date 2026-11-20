package attackspeed

import "testing"

func TestOriginalAnimationRates(t *testing.T) {
	for tier, want := range []float32{24, 27, 30, 33, 36} {
		if got := CharacterMoveFPS(uint8(tier)); got != want {
			t.Fatalf("move %d: %g", tier, got)
		}
	}
	if CharacterMoveFPS(255) != 30 || CharacterAttackFPS(4, 4, false) != 30 {
		t.Fatal("defaults")
	}
	for delta, want := range map[int]float32{-5: 15, 0: 30, 1: 34, 4: 45, 5: 50} {
		if got := CharacterAttackFPS(uint8(5+delta), 5, true); got != want {
			t.Fatalf("delta %d: %g", delta, got)
		}
	}
	if MobMoveFPS(2) != 30 || MobMoveFPS(4) != 50 || MobActionFPS(0) != 15 || MobActionFPS(6) != 45 || MobActionFPS(9) != 30 {
		t.Fatal("mob rates")
	}
}
