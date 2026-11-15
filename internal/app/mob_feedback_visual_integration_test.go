//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/go-gl/gl/v4.5-core/gl"
	"math"
	"os"
	"runtime"
	"testing"
)

func TestAltarReachRespawnAndNativeHealingGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original map zero, effects and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 800, Height: 600, Hidden: true})
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
	a.state = game.New()
	a.state.SelfID = 1
	w, err := scene.LoadMap(a.assets, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	a.wv = &worldView{world: w, pos: render.V3{150.17, 39.38975, 180.2}}
	for _, pos := range []render.V3{{155.17, 0, 180.2}, {155.1, 0, 190.38}} {
		e := &game.Entity{ID: 77, Kind: game.KindMob, TypeID: 77, X: pos[0], Y: pos[1], Z: pos[2]}
		a.state.Entities[77] = e
		if w.SpellLineOfSight(a.wv.pos, pos) {
			t.Fatal("fixture no longer reproduces buried wire target")
		}
		if !a.skillRequestReach(sdata.Skill{AttackRange: 27, Target: 3}, 77) {
			t.Fatal("clear original altar approach rejected", pos)
		}
	}
	// Actual 0601 fresh-spawn flag activates start.EFT; ordinary visibility
	// snapshots cannot invent or replay the respawn presentation.
	for _, fresh := range []bool{false, true} {
		p := proto.NewWriter(proto.MobEnter)
		p.U32(77)
		p.Bool(fresh)
		p.U16(77)
		p.F32(155.17)
		p.F32(180.2)
		a.confirmedEffects(p.Bytes())
		if !fresh && len(a.wv.impacts) != 0 {
			t.Fatal("view snapshot replayed spawn")
		}
	}
	if len(a.wv.impacts) != 1 || a.wv.impacts[0].instance.Skipped != 0 {
		t.Fatal("native spawn effect incomplete", a.wv.effectErrors)
	}
	a.scene3d, err = render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer a.scene3d.Delete()
	a.prepareMonsters()
	a.frameDT = .1
	for range 5 {
		a.updateResolvedEffects()
	}
	if len(a.wv.impacts) != 1 {
		t.Fatal("spawn ended before original appearance")
	}
	a.scene3d.View = render.LookAtLH(render.V3{155, 42, 170}, render.V3{155, 40, 180}, render.V3{0, 1, 0})
	a.scene3d.Proj = render.PerspectiveFovLH(math.Pi/3, 800.0/600, .1, 100)
	gl.Viewport(0, 0, 800, 600)
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.wv.impacts[0].instance.Draw(a.scene3d)
	spawnPixels := make([]byte, 800*600*4)
	gl.ReadPixels(0, 0, 800, 600, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&spawnPixels[0]))
	visible := 0
	for i := 0; i < len(spawnPixels); i += 4 {
		if spawnPixels[i] > 0 || spawnPixels[i+1] > 0 || spawnPixels[i+2] > 0 {
			visible++
		}
	}
	if visible == 0 {
		t.Fatal("native respawn effect has no visible pixels")
	}
	for range 200 {
		a.updateResolvedEffects()
	}
	if len(a.wv.impacts) != 0 {
		t.Fatal("native respawn effect did not expire")
	}
	t.Logf("original start.EFT: %d visible pixels; authored finite lifetime", visible)
	a.scene3d.View = render.LookAtLH(render.V3{0, 2, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0})
	a.scene3d.Proj = render.PerspectiveFovLH(math.Pi/3, 800.0/600, .1, 100)
	a.combatNumbers = []combatNumber{{Point: render.V3{0, 1, 0}, Amount: 888, Healing: true}}
	gl.Viewport(0, 0, 800, 600)
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	a.sprites.Begin(800, 600)
	a.drawCombatNumbers(800, 600)
	a.sprites.End()
	pixels := make([]byte, 800*600*4)
	gl.ReadPixels(0, 0, 800, 600, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
	green := 0
	for i := 0; i < len(pixels); i += 4 {
		if pixels[i] != 0 || pixels[i+2] != 0 {
			t.Fatal("healing has non-native red/blue channels", pixels[i:i+4])
		}
		if pixels[i+1] > 0 {
			green++
		}
	}
	if green == 0 {
		t.Fatal("healing number not rendered")
	}
	t.Logf("original map-zero stationary targets accepted; start.EFT has no missing emitters; healing green pixels=%d", green)
}
