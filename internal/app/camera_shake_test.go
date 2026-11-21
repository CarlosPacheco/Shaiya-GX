package app

import "testing"

func TestCameraShakeKeepsTheLongerTimer(t *testing.T) {
	a := inputApp()
	a.wv = &worldView{}
	a.cameraShake(.5)
	a.cameraShake(.2)
	if a.wv.shake != .5 {
		t.Fatalf("shorter request lowered the timer: %v", a.wv.shake)
	}
	a.cameraShake(2)
	if a.wv.shake != 2 {
		t.Fatalf("longer request ignored: %v", a.wv.shake)
	}
}
