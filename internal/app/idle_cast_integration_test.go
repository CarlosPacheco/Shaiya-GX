//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"os"
	"runtime"
	"testing"
)

func TestIdleRemoteUpdatesKeepOriginalCastAnimationGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	actor, err := scene.BuildCharacter(scene.NewAssets(arc), scene.Appearance{Race: 0, Job: 5, Helmet: -1}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	a := inputApp()
	a.state.SelfID = 1
	a.wv.self = actor
	if !actor.PlayCast(88, 3) {
		t.Fatal("missing original cast animation")
	}
	move := func(motion uint8) []byte {
		w := proto.NewWriter(proto.CharacterMove)
		w.U32(1)
		w.U8(motion)
		w.U16(400)
		w.F32(0)
		w.F32(.01)
		w.F32(0)
		return w.Bytes()
	}
	for range 100 {
		a.confirmedAnimation(move(0))
		actor.Update(.005)
		if _, _, active := actor.CastTiming(); !active {
			t.Fatal("idle/height refresh cancelled cast animation")
		}
	}
	a.confirmedAnimation(move(proto.MotionFlag80))
	if _, _, active := actor.CastTiming(); !active {
		t.Fatal("non-interrupting update cancelled cast")
	}
	a.confirmedAnimation(move(1))
	if _, _, active := actor.CastTiming(); active {
		t.Fatal("real walking kept cast animation")
	}
}
