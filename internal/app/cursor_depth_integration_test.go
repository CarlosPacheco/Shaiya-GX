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
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestCursorHoverDoesNotUseSynchronousClickDepthGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1280, Height: 800, Hidden: true, Debug: true})
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
	width, height := win.GetFramebufferSize()
	x, y := width/2, height/2
	a := &App{state: game.New(), sprites: sprites, screen: screenWorld, lastMouseX: x, lastMouseY: y, physW: width, physH: height, wv: &worldView{pos: render.V3{0, 0, -5}}}
	a.scene3d = &render.Scene3D{View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}), Proj: render.PerspectiveFovLH(math.Pi/4, float32(width)/float32(height), .1, 1000)}
	a.state.SelfID = 1
	for n := range 256 {
		id := uint32(n + 2)
		a.state.Entities[id] = &game.Entity{ID: id, Kind: game.KindMob}
	}
	capture := func(depth float64) {
		gl.Viewport(0, 0, int32(width), int32(height))
		gl.DepthMask(true)
		gl.DepthFunc(gl.LEQUAL)
		gl.ClearDepth(depth)
		gl.Clear(gl.DEPTH_BUFFER_BIT)
		sprites.CaptureNameDepth(width, height)
	}
	capture(1)
	if got := a.CursorIndex(width, height); got != 0 {
		t.Fatal("initial hover performed synchronous readback", got)
	}
	if a.hoverPicking {
		t.Fatal("hover scope leaked into click handling")
	}
	click := &platform.Input{Clicked: true, MouseX: x, MouseY: y}
	if got := a.pickMob(click, width, height); got != 2 {
		t.Fatal("exact click did not select nearest deterministic mob", got)
	}
	gl.Finish()
	capture(1)
	if got := a.CursorIndex(width, height); got != 1 {
		t.Fatal("completed hover sample did not show attack cursor", got)
	}
	for _, offset := range [][2]int{{1, 0}, {2, 1}, {1, 1}, {0, 0}} {
		gl.Finish()
		capture(1)
		click.MouseX, click.MouseY = x+offset[0], y+offset[1]
		a.lastMouseX, a.lastMouseY = click.MouseX, click.MouseY
		if got := a.pickMob(click, width, height); got != 2 {
			t.Fatalf("fixture left the mob at offset %v: picked %d", offset, got)
		}
		if got := a.CursorIndex(width, height); got != 1 {
			t.Errorf("offset %v: cursor = %d, want attack (1)", offset, got)
		}
	}
	// Age the last completed transfer without requesting any newer ones.
	gl.Finish()
	for range 4 {
		capture(1)
	}
	if got := a.CursorIndex(width, height); got != 1 {
		t.Errorf("stationary hover lost the attack cursor between transfers: %d", got)
	}
	a.menuOpen = true
	if got := a.CursorIndex(width, height); got != 0 {
		t.Errorf("menu left the attack cursor visible: %d", got)
	}
	a.menuOpen = false
	capture(0)
	a.CursorIndex(width, height)
	if got := a.pickMob(click, width, height); got != 0 {
		t.Fatal("stale cursor permitted click through a wall", got)
	}
	gl.Finish()
	capture(0)
	if got := a.CursorIndex(width, height); got != 0 {
		t.Fatal("new wall did not hide attack cursor", got)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error: %#x", code)
	}
}
