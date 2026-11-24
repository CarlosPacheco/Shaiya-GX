//go:build integration

package scene

import (
	"bytes"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"os"
	"runtime"
	"testing"
)

func TestCloakLightStabilityAndBasicOnlyCombatPoseGL(t *testing.T) {
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
	assets := NewAssets(arc)
	r, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Delete()
	r.Proj = render.PerspectiveFovLH(.785398, 1, .1, 50)
	r.SetCameraView(render.V3{0, 1.3, 2.5}, render.V3{0, 1.1, 0}, render.V3{0, 1, 0})
	c, err := BuildCharacter(assets, Appearance{Race: 0, Job: 5, Helmet: -1, Upper: 20, Lower: 20, Hand: 20, Foot: 20, CloakType: 24, CloakModel: 21, CloakFit: 7}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	if c.cloak == nil {
		t.Fatal("missing original cloak")
	}
	c.World = render.Identity()
	c.Update(1.0 / 60)
	c.cloak.update(c.World, c.preparePose())
	vertices := append([]render.Vertex(nil), c.cloak.vertices...)
	draw := func(concealed, oldLit bool) []byte {
		gl.Viewport(0, 0, 512, 512)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		r.Begin()
		if oldLit {
			r.DrawModel(c.cloak.geometry.mesh, c.cloak.geometry.texture, c.World, .5, false)
		} else {
			c.cloak.draw(r, c.World, concealed)
		}
		r.End()
		p := make([]byte, 512*512*4)
		gl.ReadPixels(0, 0, 512, 512, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
		return p
	}
	for _, concealed := range []bool{false, true} {
		z := float32(2.5)
		if concealed {
			z = -2.5
		}
		r.SetCameraView(render.V3{0, 1.3, z}, render.V3{0, 1.1, 0}, render.V3{0, 1, 0})
		r.Ambient = .05
		r.CharacterAmbient = .1
		r.LightDir = render.V3{0, 0, 1}
		before := draw(concealed, false)
		n := 0
		for i := 0; i < len(before); i += 4 {
			if before[i] != 0 || before[i+1] != 0 || before[i+2] != 0 {
				n++
			}
		}
		if n < 100 {
			t.Fatal("cloak test rendered no visible cloth", concealed, n)
		}
		r.Ambient = 1
		r.CharacterAmbient = 1
		r.LightDir = render.V3{1, -1, -1}
		after := draw(concealed, false)
		if !bytes.Equal(before, after) {
			t.Fatal("cloak color changed with lighting", concealed)
		}
	}
	for i, v := range vertices {
		if c.cloak.vertices[i] != v {
			t.Fatal("light fix changed cloth geometry")
		}
	}
	r.Ambient = 0
	r.LightDir = render.V3{0, 0, 1}
	litA := draw(false, true)
	r.Ambient = 1
	litB := draw(false, true)
	if bytes.Equal(litA, litB) {
		t.Fatal("fixture cannot detect the former lighting difference")
	}
	fighter, err := BuildCharacter(assets, Appearance{Race: 0, Job: 0, Helmet: -1, WeaponType: 51}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	for _, slot := range []int{49, 93, 102} {
		if !fighter.PlaySkillAction(38, slot) || fighter.combatPoseLeft != 0 {
			t.Fatal("spell entered combat stance", slot, fighter.combatPoseLeft)
		}
	}
	if !fighter.PlayBasicAttack(6, 0) || fighter.combatPoseLeft != 10 {
		t.Fatal("basic attack did not enter combat stance")
	}
	fighter.combatPoseLeft = 3
	if !fighter.PlaySkillAction(38, 102) || fighter.combatPoseLeft != 3 {
		t.Fatal("spell refreshed previous basic stance")
	}
	if !c.PlayCast(88, 2) || c.combatPoseLeft != 0 {
		t.Fatal("preparation entered combat stance")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatal("GL", code)
	}
}
