package world

import (
	"math"
	"testing"
	"time"
)

// The server books a whole hop at once; combat must use where players see the
// mob walking, and an attack waits for the hop to end.
func TestMobCombatPositionFollowsTheHopClientsSee(t *testing.T) {
	m := &MapMob{}
	t0 := time.Now()
	m.startHop(t0, mobMotionRun, true)
	m.PosX, m.PosZ = 3, 4
	m.navHop, m.moveDuration = true, time.Second
	m.startHop(t0, mobMotionRun, false)
	if !m.hopActive(t0.Add(100 * time.Millisecond)) {
		t.Fatal("the hop is not active")
	}
	x, z := m.visualXZ(t0.Add(500 * time.Millisecond))
	if math.Abs(float64(x)-1.5) > .01 || math.Abs(float64(z)-2) > .01 {
		t.Fatalf("halfway the mob shows at (%.2f, %.2f), want (1.5, 2)", x, z)
	}
	if m.hopActive(t0.Add(1100 * time.Millisecond)) {
		t.Fatal("the hop outlived its duration")
	}
	if x, z = m.visualXZ(t0.Add(1100 * time.Millisecond)); x != 3 || z != 4 {
		t.Fatalf("after the hop the mob shows at (%.2f, %.2f)", x, z)
	}
	// A new hop starts from where the mob was shown, not from the booked end.
	m.PosX, m.PosZ = 6, 8
	m.navHop, m.moveDuration = true, time.Second
	m.startHop(t0.Add(500*time.Millisecond), mobMotionRun, false)
	if m.hopFromX != 1.5 || m.hopFromZ != 2 {
		t.Fatalf("the next hop started at (%.2f, %.2f)", m.hopFromX, m.hopFromZ)
	}
}
