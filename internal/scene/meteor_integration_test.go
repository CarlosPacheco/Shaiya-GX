//go:build integration

package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"os"
	"runtime"
	"testing"
)

func TestMeteorMetadataAddsNoDrawsOrTemplatesGL(t *testing.T) {
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
	assets := NewAssets(arc)
	renderer, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer renderer.Delete()
	renderer.View = render.LookAtLH(render.V3{0, 4, -10}, render.V3{}, render.V3{0, 1, 0})
	renderer.Proj = render.PerspectiveFovLH(.78, 1, .1, 100)
	for _, id := range []uint16{53, 68, 71, 162, 163} {
		profile, ok := effect.SkillMeteor(id)
		if !ok {
			t.Fatal("missing original CMeteor skill", id)
		}
		for _, seq := range []int{profile.FallingSequence, profile.ImpactSequence} {
			x, err := LoadEffect(assets, effect.AreaLibrary(id), seq)
			if err != nil {
				t.Fatal(id, seq, err)
			}
			x.SetTransform(render.Identity())
			draw := func() int {
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				render.ResetDrawCalls()
				renderer.Begin()
				x.Draw(renderer)
				renderer.End()
				return render.DrawCalls()
			}
			before := 0
			for range 240 {
				x.Update(1.0 / 60)
				before = draw()
				if before > 0 {
					break
				}
			}
			templates := len(assets.effectTemplates)
			emitters := len(x.emitters)
			x.SetMeteorTransform(render.Basis(render.V3{1, 2, 3}, render.V3{-.6, 0, -.8}, render.V3{0, 1, 0}))
			after := draw()
			if before == 0 || after != before || len(assets.effectTemplates) != templates || len(x.emitters) != emitters {
				t.Fatal("CMeteor added work/resources", id, seq, before, after)
			}
			t.Logf("skill=%d sequence=%d draw calls %d -> %d", id, seq, before, after)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatal("GL", code)
	}
}
