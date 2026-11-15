package scene

import (
	"testing"
	"time"
)

func TestPhysicsClockDoesNotUseClampedRenderTime(t *testing.T) {
	w := &World{elapsed: 4}
	if w.DynamicTime() != 4 {
		t.Fatal("offline animation clock changed")
	}
	w.SyncPhysicsClock(100)
	w.physicsClockAt = time.Now().Add(-2 * time.Second)
	w.elapsed = 500 // simulated frames and paused rendering cannot shift physics.
	if got := w.DynamicTime(); got < 102 || got > 102.1 {
		t.Fatal("physics depended on rendered time", got)
	}
}
