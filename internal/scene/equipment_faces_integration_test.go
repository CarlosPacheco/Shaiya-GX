//go:build integration

package scene

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestOriginalEquipmentBackFacesGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and OpenGL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const size = 256
	win, err := platform.Open(platform.Config{Width: size, Height: size, Hidden: true})
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
	assets := NewAssets(arc)
	defer assets.CloseCharacterResources()
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.Ambient, s.HideWeaponEffects = .2, true
	s.Proj = render.PerspectiveFovLH(.7, 1, .01, 30)
	gl.Viewport(0, 0, size, size)
	capture := func(draw func(), file string) []byte {
		s.Begin()
		gl.ClearColor(.25, .28, .36, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		draw()
		if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" && file != "" {
			if err := render.SavePNG(filepath.Join(dir, file+".png"), size, size); err != nil {
				t.Fatal(err)
			}
		}
		pixels := make([]byte, size*size*4)
		gl.ReadPixels(0, 0, size, size, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	for _, look := range []Appearance{
		{Race: 1, Job: 2, WeaponType: 15, WeaponModel: 3, Helmet: -1},
		{Race: 0, Job: 0, WeaponType: 1, WeaponModel: 5, Helmet: -1},
		{Race: 1, Job: 3, WeaponType: 13, WeaponModel: 0, Helmet: -1},
	} {
		c, err := BuildCharacter(assets, look, "002_run")
		if err != nil || len(c.attachments) == 0 {
			t.Fatalf("weapon type %d fixture failed: %v", look.WeaponType, err)
		}
		c.World = render.Identity()
		overdraw := 0
		for frame := range 8 {
			c.Time = float32(frame) / 20
			finals := c.preparePose()
			weapon := c.attachments[0]
			// Isolate one attachment so the comparison measures item faces.
			world := weapon.extra.Mul(finals[weapon.bone]).Mul(c.World)
			part := weapon.part
			center := transformPoint(part.boundMin.Add(part.boundMax).Scale(.5), world)
			radius := part.boundMax.Sub(part.boundMin).Len()
			eye := center.Add(render.V3{.8, .7, -2}.Scale(radius))
			s.SetCameraView(eye, center, render.V3{0, 1, 0})
			ref := float32(-1)
			if weapon.cut {
				ref = .5
			}
			refPixels := capture(func() {
				gl.Enable(gl.CULL_FACE)
				gl.FrontFace(gl.CW)
				gl.CullFace(gl.BACK)
				s.DrawModel(part.Mesh, part.Tex, world, ref, false)
				gl.FrontFace(gl.CCW)
			}, "")
			var legacyFile, fixedFile string
			if frame == 2 {
				legacyFile = fmt.Sprintf("type-%02d-before", look.WeaponType)
				fixedFile = fmt.Sprintf("type-%02d-after", look.WeaponType)
			}
			legacy := capture(func() { s.DrawModel(part.Mesh, part.Tex, world, ref, false) }, legacyFile)
			saved := c.attachments
			c.attachments = saved[:1]
			fixed := capture(func() { c.drawAttachments(s, finals) }, fixedFile)
			c.attachments = saved
			instanced := capture(func() {
				s.SetInstances([]render.M4{world})
				s.DrawEquipmentInstanced(part.Mesh, part.Tex, ref)
			}, "")
			if !bytes.Equal(fixed, refPixels) || !bytes.Equal(instanced, refPixels) || gl.IsEnabled(gl.CULL_FACE) {
				t.Fatalf("weapon type %d frame %d differs from the original cull state", look.WeaponType, frame)
			}
			for i := 0; i < len(fixed); i += 4 {
				if int(legacy[i])+int(legacy[i+1])+int(legacy[i+2])+48 < int(fixed[i])+int(fixed[i+1])+int(fixed[i+2]) {
					overdraw++
				}
			}
		}
		if overdraw == 0 {
			t.Fatalf("weapon type %d fixture did not reproduce dark overdraw", look.WeaponType)
		}
		t.Logf("weapon type %d: %d darker pixels across eight poses with both faces drawn", look.WeaponType, overdraw)
	}
}
