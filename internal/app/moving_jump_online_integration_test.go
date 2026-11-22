//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestMovingJumpAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_MOVING_JUMP") != "1" {
		t.Skip("requires isolated moving-jump fixture")
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
	st, wire, _, _ := testworld.Enter(t)
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, wire, screenWorld
	a.enterWorld()
	a.pumpWorld()
	v := a.wv
	v.facing = render.V3{1, 0, 0}
	v.resetCamera() // W follows the camera heading.
	start, last := time.Now(), time.Now()
	var jumped time.Time
	initialY, peak := v.pos[1], v.pos[1]
	corrections := 0
	for time.Since(start) < 2*time.Second {
		now := time.Now()
		dt := now.Sub(last).Seconds()
		last = now
		// Trace corrections before applying their authoritative state.
	drain:
		for {
			select {
			case p := <-wire.Packets:
				if _, err := st.Apply(p); err != nil {
					t.Fatal(err)
				}
				if proto.Op(p) == proto.GXPlayerCorrection {
					corrections++
					t.Logf("correction t=%.3f client=%v jumping=%v fall=%g server=%+v", time.Since(start).Seconds(), v.pos, v.jumping, v.jumpFall, st.LastPlayerCorrection)
				}
				a.confirmedPosition(p)
			default:
				break drain
			}
		}
		in := &platform.Input{Down: "W"}
		if jumped.IsZero() && now.Sub(start) > 300*time.Millisecond {
			jumped = now
			initialY = v.pos[1]
			in.Keys = []platform.Key{platform.KeySpace}
		}
		a.updateWorld(in, dt)
		peak = max(peak, v.pos[1])
		if !jumped.IsZero() && now.Sub(jumped) < 450*time.Millisecond && !v.jumpPending && !v.jumping {
			t.Fatalf("moving jump cancelled at %v; peak=%g startY=%g correctionCount=%d", now.Sub(jumped), peak, initialY, corrections)
		}
		time.Sleep(time.Second / 120)
	}
	t.Logf("moving jump peak=%g delta=%g corrections=%d end=%v", peak, peak-initialY, corrections, v.pos)
	if corrections != 0 {
		t.Fatal("valid moving jump rewound by correction", corrections)
	}
	if peak-initialY < .8 {
		t.Fatal("moving jump arc truncated")
	}
}
