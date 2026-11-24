//go:build integration

package scene

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

// SHAIYA_EMBLEM_OUT names the PNG written by the render check.
func TestCloakEmblemOverlayGL(t *testing.T) {
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
	skin, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer skin.Delete()
	r.Proj = render.PerspectiveFovLH(.785398, 1, .1, 50)
	ap := Appearance{Race: 0, Job: 5, Helmet: -1, Upper: 20, Lower: 20, Hand: 20, Foot: 20, CloakType: 24, CloakModel: cloakModel(), CloakFit: 7,
		CloakEmblem: [6]uint8{4, 3, 9, 0, 6, 11}}
	c, err := BuildCharacter(assets, ap, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	if c.cloak == nil || c.cloak.emblem == nil {
		t.Fatal("cloak or emblem missing")
	}
	for i, layer := range c.cloak.emblem.layers {
		if layer.tex == nil {
			t.Fatalf("layer %d has no texture; missing: %v", i, assets.Missing)
		}
	}
	c.World = render.Identity()
	c.Update(1.0 / 60)
	c.cloak.update(c.World, c.preparePose())
	gl.Viewport(0, 0, 512, 512)
	gl.ClearColor(.2, .2, .25, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	r.SetCameraView(render.V3{0, 1.2, 2.2}, render.V3{0, 1.1, 0}, render.V3{0, 1, 0})
	r.Ambient, r.CharacterAmbient, r.LightDir = .5, 1, render.V3{0, 0, 1}
	r.Begin()
	c.Draw(skin, r)
	r.End()
	p := make([]byte, 512*512*4)
	gl.ReadPixels(0, 0, 512, 512, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
	img := image.NewNRGBA(image.Rect(0, 0, 512, 512))
	for y := 0; y < 512; y++ {
		copy(img.Pix[y*img.Stride:], p[(511-y)*512*4:(512-y)*512*4])
	}
	out := os.Getenv("SHAIYA_EMBLEM_OUT")
	if out == "" {
		out = filepath.Join(t.TempDir(), "emblem.png")
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func cloakModel() uint8 {
	var v int
	if s := os.Getenv("SHAIYA_CLOAK_MODEL"); s != "" {
		fmt.Sscan(s, &v)
		return uint8(v)
	}
	return 21
}
