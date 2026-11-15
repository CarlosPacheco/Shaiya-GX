//go:build integration

package scene

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestOriginalItemGlowsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w, err := platform.Open(platform.Config{Width: 640, Height: 640, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := NewAssets(arc)
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	k, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer k.Delete()
	for selector := int32(0); selector < 4; selector++ {
		for family := uint8(1); family <= 15; family++ {
			x := a.itemGlow(family, selector)
			if x == nil || len(x.emitters) != 1 || x.Skipped != 0 {
				t.Fatalf("family%d glow%d missing: %v", family, selector, a.Missing)
			}
		}
	}
	for _, typ := range []uint8{19, 34, 69, 75, 84, 90} {
		if a.itemGlow(typ, 0) != nil {
			t.Fatal("shield received unavailable weapon-family emitter", typ)
		}
	}
	if a.itemGlow(12, -1) != nil || a.itemGlow(12, 4) != nil {
		t.Fatal("invalid ITM selector emitted")
	}
	// Real staff models use all four selectors, independent of enchant level.
	for selector, model := range []uint8{135, 136, 137, 138} {
		c, err := BuildCharacter(a, Appearance{Race: 0, Job: 5, Helmet: -1, WeaponType: 12, WeaponModel: model}, "000_normal")
		if err != nil {
			t.Fatal(err)
		}
		if len(c.attachments) != 1 || c.attachments[0].glow == nil || c.attachments[0].enchant != nil {
			t.Fatalf("staff%d: missing innate +0 glow", model)
		}
		if c.attachments[0].glow.emitters[0].definition != &a.itemGlowLibraries[selector].Effects[11] {
			t.Fatal("ITM selector chose the wrong library or family")
		}
		for range 120 {
			c.Update(1.0 / 60)
		}
		gl.Viewport(0, 0, 640, 640)
		s.SetCameraView(render.V3{0, 1, 4}, render.V3{0, 1, 0}, render.V3{0, 1, 0})
		s.Proj = render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)
		// Warm particles at the actual attachment, then isolate their pixels.
		for range 120 {
			c.Update(1.0 / 60)
			c.drawAttachments(s, c.preparePose())
		}
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		x := c.attachments[0].glow
		x.Draw(s)
		pixels := make([]byte, 640*640*4)
		gl.ReadPixels(0, 0, 640, 640, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		visible := 0
		for i := 0; i < len(pixels); i += 4 {
			if pixels[i] > 8 || pixels[i+1] > 8 || pixels[i+2] > 8 {
				visible++
			}
		}
		if visible < 20 {
			t.Fatalf("glow%d invisible: %d pixels", selector, visible)
		}
		t.Logf("glow%d staff model%d: %d visible pixels", selector, model, visible)
		dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
		if dir != "" {
			if err := render.SavePNG(filepath.Join(dir, fmt.Sprintf("glow-%d.png", selector)), 640, 640); err != nil {
				t.Fatal(err)
			}
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			c.Draw(k, s)
			if err := render.SavePNG(filepath.Join(dir, fmt.Sprintf("staff-%d.png", selector)), 640, 640); err != nil {
				t.Fatal(err)
			}
		}
		before := x.Time
		c.drawAttachments(s, c.preparePose())
		if x.Time != before {
			t.Fatal("second camera pass advanced innate effect")
		}
		s.HideWeaponEffects = true
		c.Update(1.0 / 60)
		c.drawAttachments(s, c.preparePose())
		if x.Time != before {
			t.Fatal("disabled weapon effects advanced")
		}
		s.HideWeaponEffects = false
		s.Shadow = true
		c.drawAttachments(s, c.preparePose())
		if x.Time != before {
			t.Fatal("shadow pass advanced innate effect")
		}
		s.Shadow = false
		c.Concealed = true
		c.drawAttachments(s, c.preparePose())
		if x.Time != before {
			t.Fatal("concealed actor advanced innate effect")
		}
	}
	for _, model := range []uint8{0, 21} {
		c, err := BuildCharacter(a, Appearance{Race: 0, Job: 5, Helmet: -1, WeaponType: 12, WeaponModel: model}, "000_normal")
		if err != nil {
			t.Fatal(err)
		}
		if (c.attachments[0].glow != nil) != (model == 21) {
			t.Fatal("plain/nonelemental staff selection", model)
		}
	}
	if a.itemGlow(60, 0).emitters[0].definition != &a.itemGlowLibraries[0].Effects[11] {
		t.Fatal("alternate staff family was not normalized")
	}
	c, err := BuildCharacter(a, Appearance{Race: 0, Helmet: -1, WeaponType: 5, WeaponModel: 12, WeaponEnchant: 5}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.attachments) != 2 {
		t.Fatal("dual binding count")
	}
	for _, p := range c.attachments {
		if p.glow == nil || p.enchant == nil {
			t.Fatal("innate/enchant did not coexist on dual bindings")
		}
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("GL error %#x", err)
	}
}
