package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestRunTrailRequiresConfirmedHasteAndOnFootRunning(t *testing.T) {
	for _, tier := range []uint8{0, 1, 2, 3, 4, 5, 255} {
		sp := &proto.CharacterSpeed{Move: tier}
		if got := runTrailEnabled(sp, true, false, false, false, false); got != (tier == 3 || tier == 4) {
			t.Fatalf("tier %d: %v", tier, got)
		}
		for flag := 0; flag < 5; flag++ {
			args := [5]bool{true, false, false, false, false}
			args[flag] = !args[flag]
			if runTrailEnabled(sp, args[0], args[1], args[2], args[3], args[4]) {
				t.Fatalf("tier %d flag %d bypassed guard", tier, flag)
			}
		}
	}
	if runTrailEnabled(nil, true, false, false, false, false) {
		t.Fatal("unconfirmed speed produced trail")
	}
}
