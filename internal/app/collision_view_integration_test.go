//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCollisionViewOriginalMarketGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original data")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true})
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
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	v := a.wv
	v.place(render.V3{995.8, v.world.Height(995.8, 797.3), 797.3})
	v.eye = v.pos.Add(render.V3{7, 5, -10})
	v.facing = render.V3{0, 0, 1}
	v.cameraReady = false
	a.state.Self.IsGM = true
	a.collisionViewCommand("/cview on")
	for i := 0; i < 1000; i++ {
		v.world.UpdateVisible(v.eye, 200)
		if v.world.Ready() {
			break
		}
	}
	if !v.world.Ready() {
		t.Fatal("market did not stream")
	}
	gl.Viewport(0, 0, 1024, 768)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.Draw(1024, 768)
	if len(a.collisionVertices) < 300 || a.collisionLines == nil {
		t.Fatal("collision view produced no geometry")
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("GL error %#x", err)
	}
	if directory := os.Getenv("SHAIYA_TEST_ARTIFACTS"); directory != "" {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
		if err := render.SavePNG(filepath.Join(directory, "collision-view-market.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	a.collisionViewCommand("/cview off")
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.Draw(1024, 768)
}

func TestCollisionSupportAndJumpOriginalDungeonGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original data")
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
	w, err := scene.LoadMap(scene.NewAssets(arc), 3)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	found := false
	var position render.V3
	w.CollisionTriangles(render.V3{}, 100000, func(triangle collision.Triangle) {
		if found {
			return
		}
		cross := triangle[1].Sub(triangle[0]).Cross(triangle[2].Sub(triangle[0]))
		if cross.Len() < 25 || cross.Norm()[1] < .95 {
			return
		}
		centre := triangle[0].Add(triangle[1]).Add(triangle[2]).Scale(1.0 / 3)
		position = render.V3{float32(centre[0]), float32(centre[1]) + 3, float32(centre[2])}
		found = true
	})
	if !found {
		t.Fatal("dungeon has no original support surface")
	}
	v := &worldView{world: w, pos: position, eye: position.Add(render.V3{0, 2, -5})}
	for range 300 {
		v.settleOnGround(1.0 / 60)
	}
	floor := v.pos[1]
	for range 120 {
		v.settleOnGround(1.0 / 60)
	}
	if math.Abs(float64(v.pos[1]-floor)) > .001 || floor >= position[1] {
		t.Fatal("did not settle on a stable authored floor", position, v.pos)
	}
	v.beginJump()
	peak := floor
	for range 360 {
		v.advanceJump(1.0 / 60)
		peak = max(peak, v.pos[1])
		if !v.jumping {
			break
		}
	}
	if v.jumping || peak < floor+.1 || math.Abs(float64(v.pos[1]-floor)) > .002 {
		t.Fatal("jump did not return to the original support", floor, peak, v.pos)
	}
	v.place(render.V3{100000, 1000, 100000})
	v.groundFall = 0
	if !v.settleOnGround(.1) || v.pos[1] >= 1000 {
		t.Fatal("walking beyond supporting geometry did not allow falling")
	}
	t.Logf("original dungeon support %.4f; jump peak %.4f; return stable", floor, peak)
}

func TestOriginalFieldGroundJumpGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
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
	w, err := scene.LoadMap(scene.NewAssets(arc), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	p := render.V3{578, w.Height(578, 1762) + .05, 1762}
	v := &worldView{world: w, pos: p, eye: p.Add(render.V3{0, 2, -5})}
	for range 300 {
		v.settleOnGround(1.0 / 60)
	}
	floor := v.pos[1]
	v.beginJump()
	peak := floor
	for range 180 {
		v.advanceJump(1.0 / 60)
		peak = max(peak, v.pos[1])
	}
	if peak < floor+.5 || v.jumping {
		t.Fatalf("field jump blocked: floor %g peak %g end %v", floor, peak, v.pos)
	}
}
