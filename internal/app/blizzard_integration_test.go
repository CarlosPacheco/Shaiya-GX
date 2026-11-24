//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBlizzardAreaOriginalAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 512, Height: 512, Hidden: true})
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
	assets := scene.NewAssets(arc)
	renderer, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer renderer.Delete()
	renderer.View = render.LookAtLH(render.V3{8, 6, -12}, render.V3{0, 2, 0}, render.V3{0, 1, 0})
	renderer.Proj = render.PerspectiveFovLH(.785398, 1, .1, 100)
	a := inputApp()
	a.assets, a.scene3d = assets, renderer
	raw, err := arc.ReadFile("Character/Skill.SData")
	if err != nil {
		t.Fatal(err)
	}
	a.skillDefs, err = sdata.ParseSkills(raw)
	if err != nil {
		t.Fatal(err)
	}
	a.state.SelfID = 1
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindMob, MobHeightKnown: true}
	for level := uint8(1); level <= 3; level++ {
		a.wv.impacts = nil
		a.wv.areaShowers = nil
		a.startSkillResultEffects(proto.SkillResult{ID: 71, Level: level, Source: 1, Target: 2, Result: 4})

		if len(a.wv.areaShowers) != 1 {
			t.Fatal("Blizzard announcement did not create shower", level)
		}
		if a.wv.areaShowers[0].radius <= 1 {
			t.Fatal("lost original area radius")
		}
		center := a.wv.areaShowers[0].center
		a.state.Entities[2].X = 100
		if a.wv.areaShowers[0].center != center {
			t.Fatal("shower follows fleeing target")
		}
		a.state.Entities[2].X = 0
		drawn := false
		for frame := 0; frame < 240; frame++ {
			a.frameDT = 1.0 / 60
			a.updateResolvedEffects()
			gl.Viewport(0, 0, 512, 512)
			gl.ClearColor(0, 0, 0, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			render.ResetDrawCalls()
			renderer.Begin()
			a.drawResolvedEffects()
			renderer.End()
			if render.DrawCalls() > 0 {
				drawn = true
				if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" && level == 1 && frame == 60 {
					if err := render.SavePNG(filepath.Join(dir, "blizzard.png"), 512, 512); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		if !drawn {
			t.Fatal("Blizzard produced no geometry", level)
		}
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatal("GL", code)
		}
		for _, id := range []uint32{2, 3} {
			a.startSkillSecondaryEffect(proto.SkillResult{ID: 71, Level: level, Source: 1, Target: id})
		}
		if len(a.wv.areaShowers) != 1 {
			t.Fatal("secondary hits duplicated area cloud")
		}
	}
}
