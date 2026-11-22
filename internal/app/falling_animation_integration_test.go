//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"os"
	"runtime"
	"testing"
)

func TestJumpActionFinishesOnceDuringLongFallGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original models and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 128, Height: 128, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	v := a.wv
	start := v.pos
	for _, walking := range []bool{false, true} {
		for _, fps := range []int{30, 60, 144} {
			v.self.ReleaseAction()
			v.place(start.Add(render.V3{0, 80, 0}))
			v.walking = walking
			v.startJumpAnimation()
			v.beginJump()
			if !v.self.PlayingAction(8) {
				t.Fatal("jump did not start original one-shot")
			}
			action := v.self.ActionID()
			for range fps * 3 {
				// This presentation fixture has no connection. Suppress packet scheduling;
				// the network regression separately tests real moving-jump input.
				v.moving, v.motion, v.sendT = true, 0x84, 1000
				a.updateWorld(&platform.Input{Down: "W"}, 1/float64(fps))
			}
			want := "002_run"
			if walking {
				want = "001_walk"
			}
			if !v.jumping || v.self.PlayingAction(8) || v.self.PlayingClip("008_jump") || !v.self.PlayingClip(want) {
				t.Fatalf("walk=%v fps=%d long fall retained jump or wrong locomotion", walking, fps)
			}
			if v.self.ActionID() != action+1 {
				t.Fatal("jump action restarted during fall")
			}
		}
	}
	for _, walking := range []bool{false, true} {
		v.self.ReleaseAction()
		v.self.SetClip("008_jump")
		v.place(start.Add(render.V3{0, 80, 0}))
		v.jumping = false
		v.groundFall = 0
		v.walking = walking
		v.moving, v.motion, v.sendT = false, 0, 1000
		a.updateWorld(&platform.Input{}, 1.0/60)
		if !v.self.PlayingClip("000_normal") {
			t.Fatal("uncommanded fall must use idle", walking)
		}
	}

	// Releasing movement during an actual jump must return to idle, including
	// the frame where the one-shot jump finishes.
	for _, walking := range []bool{false, true} {
		v.self.ReleaseAction()
		v.place(start.Add(render.V3{0, 80, 0}))
		v.walking = walking
		v.startJumpAnimation()
		v.beginJump()
		for frame := 0; frame < 180; frame++ {
			in := &platform.Input{}
			v.moving, v.motion, v.sendT = false, 0, 1000
			if v.jumping {
				v.motion = 0x84
			}
			if frame < 5 {
				in.Down = "W"
				v.moving, v.motion = true, 0x84
			}
			a.updateWorld(in, 1.0/60)
			if frame >= 5 && !v.self.ActionActive() && !v.self.PlayingClip("000_normal") {
				t.Fatalf("released movement forced locomotion during fall: walk=%v frame=%d", walking, frame)
			}
		}
	}

	// Use level ground under the real character animation so we can check
	// the few frames where physics has landed and action 8 is still playing.
	world := v.world
	v.world = nil
	defer func() { v.world = world }()
	v.self.ReleaseAction()
	v.place(start)
	v.startJumpAnimation()
	v.beginJump()
	action := v.self.ActionID()
	landed := false
	for frame := 0; frame < 65; frame++ {
		in := platform.Input{Keys: []platform.Key{platform.KeyEnter}}
		if !a.updateChat(&in) {
			t.Fatal("Enter was not captured by chat")
		}
		controls := chatWorldInput(in)
		if landed && v.self.PlayingAction(8) {
			// A second jump edge during the last frames must not restart them.
			controls.Keys = []platform.Key{platform.KeySpace}
		}
		v.motion, v.sendT = 0, 1000
		if v.jumping {
			// Keep the movement send fixture away from the landing transition.
			v.motion = 0x84
			if v.pos[1]+(9+max(-20, v.jumpFall*(1+2.8/60)))/60 <= v.jumpFloor {
				v.motion = 0
			}
		}
		a.updateWorld(&controls, 1.0/60)
		if v.jumpPending || v.self.PlayingAction(8) && v.self.ActionID() != action {
			t.Fatalf("jump clip restarted on frame %d", frame)
		}
		if !v.jumping {
			landed = landed || v.self.PlayingAction(8)
			if !v.self.PlayingAction(8) && !v.self.PlayingClip("000_normal") {
				t.Fatalf("grounded jump leaked its finished clip on frame %d", frame)
			}
		}
	}
	if !landed {
		t.Fatal("fixture did not land while action 8 was active")
	}
	controls := platform.Input{Keys: []platform.Key{platform.KeySpace}}
	a.updateWorld(&controls, 1.0/60)
	if !v.jumpPending || !v.self.PlayingAction(8) || v.self.ActionID() == action {
		t.Fatal("a new jump did not start after the previous clip completed")
	}

}
