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
		want := "002_run"
		if walking {
			want = "001_walk"
		}
		if !v.self.PlayingClip(want) {
			t.Fatal("uncommanded fall retained stale jump or idle", walking)
		}
	}

}
