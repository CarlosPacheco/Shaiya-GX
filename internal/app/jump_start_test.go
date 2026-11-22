package app

import "testing"

// The key starts the clip; the impulse begins a frame later, as in 0x43bfa0.
func TestJumpImpulseWaitsOneFrameAfterTheClipStarts(t *testing.T) {
	v := &worldView{}
	v.startJumpAnimation()
	if v.jumping || !v.jumpPending {
		t.Fatal("the key press must not apply the impulse", v.jumping, v.jumpPending)
	}
	v.beginJump()
	if !v.jumping || v.jumpPending || v.jumpFall != -2.98 {
		t.Fatal("the next frame must start the impulse", v.jumping, v.jumpPending, v.jumpFall)
	}
}
