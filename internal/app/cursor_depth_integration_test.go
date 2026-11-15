//go:build integration

package app

import (
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestCursorHoverDoesNotUseSynchronousClickDepthGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 800, Height: 600, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	sprites, err := render.NewSprites(4)
	if err != nil {
		t.Fatal(err)
	}
	defer sprites.ResetHoverDepth()
	a := &App{state: game.New(), sprites: sprites, screen: screenWorld, lastMouseX: 400, lastMouseY: 300, wv: &worldView{pos: render.V3{0, 0, -5}}}
	a.scene3d = &render.Scene3D{View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}), Proj: render.PerspectiveFovLH(math.Pi/4, 800.0/600, .1, 1000)}
	a.state.SelfID = 1
	for n := range 256 {
		id := uint32(n + 2)
		a.state.Entities[id] = &game.Entity{ID: id, Kind: game.KindMob}
	}
	capture := func(depth float64) {
		gl.Viewport(0, 0, 800, 600)
		gl.DepthMask(true)
		gl.DepthFunc(gl.LEQUAL)
		gl.ClearDepth(depth)
		gl.Clear(gl.DEPTH_BUFFER_BIT)
		sprites.CaptureNameDepth(800, 600)
	}
	capture(1)
	if got := a.CursorIndex(800, 600); got != 0 {
		t.Fatal("initial hover performed synchronous readback", got)
	}
	if a.hoverPicking {
		t.Fatal("hover scope leaked into click handling")
	}
	click := &platform.Input{Clicked: true, MouseX: 400, MouseY: 300}
	if got := a.pickMob(click, 800, 600); got != 2 {
		t.Fatal("exact click did not select nearest deterministic mob", got)
	}
	gl.Finish()
	capture(1)
	if got := a.CursorIndex(800, 600); got != 1 {
		t.Fatal("completed hover sample did not show attack cursor", got)
	}
	capture(0)
	a.CursorIndex(800, 600)
	if got := a.pickMob(click, 800, 600); got != 0 {
		t.Fatal("stale cursor permitted click through a wall", got)
	}
	gl.Finish()
	capture(0)
	if got := a.CursorIndex(800, 600); got != 0 {
		t.Fatal("new wall did not hide attack cursor", got)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error: %#x", code)
	}
}
