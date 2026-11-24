//go:build integration

package app

import (
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestSoulBurnEffectStaysAtCorpseAfterRespawnGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires the native Soul Burn effect and an OpenGL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const size = 320
	window, err := platform.Open(platform.Config{Width: size, Height: size, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(true); err != nil {
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
	a.state, a.wv = game.New(), &worldView{}
	a.state.SelfID = 1
	const victim = 2
	a.state.Entities[victim] = &game.Entity{ID: victim, Kind: game.KindUser, Dead: true, X: 20, Y: 5, Z: 30}
	send := func(w *proto.Writer) {
		t.Helper()
		p := w.Bytes()
		if _, err := a.state.Apply(p); err != nil {
			t.Fatal(err)
		}
		a.confirmedEffects(p)
	}
	result := func(code uint8) *proto.Writer {
		w := proto.NewWriter(proto.SkillUseNoTarget)
		w.U8(code)
		w.U32(1)
		w.U32(victim)
		w.U16(125)
		w.U8(1)
		w.U16(0)
		w.U16(0)
		w.U16(0)
		return w
	}
	send(result(3))
	if len(a.wv.impacts) != 0 {
		t.Fatal("a failed Soul Burn created an impact")
	}
	send(result(0))
	if len(a.wv.impacts) != 1 || a.wv.impacts[0].instance.Skipped != 0 {
		t.Fatal("Soul Burn's native effect failed to load", a.wv.effectErrors)
	}
	leave := proto.NewWriter(proto.CharacterReviveInit)
	leave.U32(victim)
	send(leave)
	delete(a.state.Entities, victim)
	a.scene3d, err = render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer a.scene3d.Delete()
	a.scene3d.View = render.LookAtLH(render.V3{20, 8, 20}, render.V3{20, 7, 30}, render.V3{0, 1, 0})
	a.scene3d.Proj = render.PerspectiveFovLH(math.Pi/3, 1, .1, 100)
	a.frameDT = .1
	pixels := make([]byte, size*size*4)
	for range 15 {
		a.updateResolvedEffects()
		if len(a.wv.impacts) == 0 {
			break
		}
		impact := a.wv.impacts[0]
		m, ok := a.impactTransform(impact)
		if !ok || m[12] != 20 || m[13] != 5 || m[14] != 30 {
			t.Fatalf("Soul Burn lost its corpse position: %v", m)
		}
		gl.Viewport(0, 0, size, size)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		impact.instance.Draw(a.scene3d)
		gl.ReadPixels(0, 0, size, size, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		for i := 0; i < len(pixels); i += 4 {
			if pixels[i]|pixels[i+1]|pixels[i+2] != 0 {
				return
			}
		}
	}
	t.Fatal("Soul Burn produced no visible pixels at the old corpse location")
}
