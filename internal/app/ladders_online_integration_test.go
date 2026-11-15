//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"math"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestLaddersAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_LADDER") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires isolated ladder fixture and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 640, Height: 480, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
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
	v := a.wv
	l := v.world.PlayerLadders()[0]
	v.facing = render.V3(l.Normal).Scale(-1)
	step := func(key byte, d time.Duration) {
		t.Helper()
		end := time.Now().Add(d)
		previous := time.Now()
		for time.Now().Before(end) {
			now := time.Now()
			dt := now.Sub(previous).Seconds()
			previous = now
			a.clock += dt
			a.frameDT = dt
			a.pumpWorld()
			in := &platform.Input{}
			if key != 0 {
				in.Down = string(key)
			}
			a.updateWorld(in, dt)
			time.Sleep(time.Second / 120)
		}
		a.pumpWorld()
	}
	start := v.pos[1]
	step('W', 500*time.Millisecond)
	if !v.ladder.Active || st.Self.ClimbMode != mapgeometry.LadderUp || v.pos[1] < start+.5 {
		t.Fatal("climb not confirmed", v.pos, st.Self.ClimbMode, v.ladder, st.LastPlayerCorrection, "start", start)
	}
	held := v.pos[1]
	step(0, 350*time.Millisecond)
	if math.Abs(float64(v.pos[1]-held)) > .12 || st.Self.ClimbMode != mapgeometry.LadderHold {
		t.Fatal("climber did not hold", v.pos, held, st.Self.ClimbMode)
	}
	step('S', 250*time.Millisecond)
	if v.pos[1] > held-.3 {
		t.Fatal("did not descend", v.pos, held)
	}
	step('W', 6*time.Second)
	if v.ladder.Active || v.ladderFinishing || st.Self.ClimbMode != 0 || !v.world.PlayerSupported(v.pos) || v.pos[1] < start+8 {
		t.Fatal("did not land above", v.pos, st.Self.ClimbMode)
	}
	t.Logf("Native ladder climbed, held, descended and landed: %.3f -> %.3f", start, v.pos[1])
}
