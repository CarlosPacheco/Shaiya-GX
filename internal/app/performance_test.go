package app

import "testing"

func TestPerformanceHUDAveragesAlternatingDrawCalls(t *testing.T) {
	var hud performanceHUD
	const dt = 0.017 // 14*0.017 = 0.238 < 0.25 <= 15*0.017 = 0.255 (15-frame odd stride)
	const low, high = 117, 2056

	// 15 frames (~60 Hz) trigger the 0.25s refresh window (odd stride).
	// Alternating frames should report the window average rather than toggling
	// between the two single-frame extremes.
	for i := range 15 {
		calls := low
		if i%2 == 1 {
			calls = high
		}
		hud.sampleCalls(dt, calls)
	}
	// 8 low + 7 high = 15328; rounded average = (15328 + 7) / 15 = 1022.
	if hud.calls != 1022 {
		t.Fatalf("first window calls = %d, want 1022", hud.calls)
	}
	if hud.callsTotal != 0 || hud.frames != 0 {
		t.Fatalf("accumulators not reset: callsTotal=%d frames=%d", hud.callsTotal, hud.frames)
	}

	for i := range 15 {
		calls := high
		if i%2 == 1 {
			calls = low
		}
		hud.sampleCalls(dt, calls)
	}
	// 8 high + 7 low = 17267; rounded average = (17267 + 7) / 15 = 1151.
	if hud.calls != 1151 {
		t.Fatalf("second window calls = %d, want 1151", hud.calls)
	}

	// Non-positive dt must be ignored without accumulating draw calls.
	hud.sampleCalls(0, 9999)
	hud.sampleCalls(-0.01, 9999)
	if hud.callsTotal != 0 || hud.frames != 0 {
		t.Fatalf("non-positive dt mutated accumulators: callsTotal=%d frames=%d", hud.callsTotal, hud.frames)
	}

	// Even number of alternating frames (16 frames at 1/64s = 0.25s) yields the exact rounded mean.
	for i := range 16 {
		calls := low
		if i%2 == 1 {
			calls = high
		}
		hud.sampleCalls(0.25/16, calls)
	}
	// (8*117 + 8*2056 + 8) / 16 = 17392 / 16 = 1087.
	if hud.calls != 1087 {
		t.Fatalf("even window calls = %d, want 1087", hud.calls)
	}
}
