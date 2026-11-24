//go:build integration

package scene

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWindSpinsAndStingerOriginalActorsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 640, Height: 480, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
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
	skin, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer skin.Delete()
	renderer.SetCameraView(render.V3{0, 1.5, 4}, render.V3{0, 1, 0}, render.V3{0, 1, 0})
	renderer.Proj = render.PerspectiveFovLH(math.Pi/4, 640.0/480, .1, 50)
	renderer.LightDir = render.V3{.3, -1, -.5}
	renderer.Ambient = .8
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	draw := func(c *Character, name string) {
		t.Helper()
		gl.Viewport(0, 0, 640, 480)
		gl.ClearColor(.1, .1, .1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		renderer.Begin()
		c.Draw(skin, renderer)
		renderer.End()
		if err := gl.GetError(); err != gl.NO_ERROR {
			t.Fatalf("GL %#x", err)
		}
		if err := render.SavePNG(filepath.Join(dir, name+".png"), 640, 480); err != nil {
			t.Fatal(err)
		}
	}
	for _, race := range []uint8{0, 3} {
		for sex := uint8(0); sex < 2; sex++ {
			c, err := BuildCharacter(assets, Appearance{Race: race, Gender: sex, Job: 0, Helmet: -1, Height: 2}, "000_normal")
			if err != nil {
				t.Fatal(err)
			}
			c.World = render.Identity()
			for _, id := range []uint16{38, 156} {
				if !c.PlaySkillAction(id, 102) {
					t.Fatal("missing spin", race, sex)
				}
				for range 90 {
					c.Update(1.0 / 60)
				}
				if !c.PlayingAction(102) || c.Time >= .2 {
					t.Fatal("spin not looping at 1.5s")
				}
				draw(c, fmt.Sprintf("spin-%d-%d-%d", race, sex, id))
			}
			old := c.ActionID()
			if !c.PlaySkillAction(38, 102) || c.ActionID() == old {
				t.Fatal("combo failed to replace spin")
			}
			if !c.PlaySkillAction(56, 101) {
				t.Fatal("missing Stinger")
			}
			contact := c.actions[101].Metadata[2]
			if contact != skilltiming.StingerContact(race, sex) {
				t.Fatal("server/client contact mismatch", race, sex, contact)
			}
			d := proto.SkillDash{From: [3]float32{}, To: [3]float32{0, 0, 20}, Contact: contact}
			for _, elapsed := range []float64{.2, .4, .5, d.TravelEnd() + .1} {
				frame := d.Frame(elapsed)
				c.SetActionFrame(frame)
				c.Update(1.0 / 60)
				if math.Abs(float64(c.Time*30-frame)) > .001 {
					t.Fatal("travel did not freeze cursor")
				}
				draw(c, fmt.Sprintf("stinger-%d-%d-%.2f", race, sex, elapsed))
			}
		}
	}
	if len(assets.Missing) > 0 {
		t.Fatal("missing assets", assets.Missing)
	}
}
