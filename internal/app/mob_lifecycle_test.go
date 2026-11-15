package app

import "testing"

func TestCorpseUsesOpaqueHoldAndLightweightFade(t *testing.T) {
	for _, tc := range []struct {
		age  float64
		fade float32
	}{{0, 0}, {3.5, 0}, {4.25, .5}, {5, 1}, {100, 1}} {
		if got := corpseFade(tc.age); got != tc.fade {
			t.Fatal(tc, got)
		}
	}
}

func TestRepeatedDeathDoesNotRestartFadeAndRespawnClearsIt(t *testing.T) {
	m := mobActor{}
	if m.updateCorpse(true, 10) != 0 || m.updateCorpse(true, 14.25) != .5 || m.updateCorpse(true, 20) != 1 {
		t.Fatal("death updates restarted the lifetime")
	}
	if m.updateCorpse(false, 21) != 0 || m.updateCorpse(true, 25) != 0 || m.updateCorpse(true, 29.25) != .5 {
		t.Fatal("respawn did not start a new lifecycle")
	}
}
