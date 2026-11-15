//go:build integration

package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCloakShoulderVisibilityGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
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
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	s := NewAssets(arc)
	r, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Delete()
	k, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer k.Delete()
	r.Proj = render.PerspectiveFovLH(math.Pi/4, 800.0/600, .1, 50)
	r.Ambient = .8
	r.LightDir = render.V3{.3, -1, -.5}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	for _, fit := range []uint8{0, 7} {
		a, err := BuildCharacter(s, Appearance{Race: 0, Gender: 0, Job: 5, Upper: 20, Lower: 20, Hand: 20, Foot: 20, Helmet: -1, CloakType: 24, CloakModel: 21, CloakFit: fit}, "000_normal")
		if err != nil {
			t.Fatal(err)
		}
		if a.cloak == nil {
			t.Fatal("missing cloak", s.Missing)
		}
		a.World = render.Identity()
		a.Update(1.0 / 60)
		for _, back := range []bool{false, true} {
			z := float32(-1.8)
			if back {
				z = 1.8
			}
			r.SetCameraView(render.V3{0, 1.5, z}, render.V3{0, 1.4, 0}, render.V3{0, 1, 0})
			draw := func() []byte {
				gl.Viewport(0, 0, 800, 600)
				gl.ClearColor(.15, .18, .22, 1)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				r.Begin()
				a.Draw(k, r)
				r.End()
				p := make([]byte, 800*600*4)
				gl.ReadPixels(0, 0, 800, 600, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
				return p
			}
			full := draw()
			name := fmtName(fit, back)
			if err := render.SavePNG(filepath.Join(dir, name+".png"), 800, 600); err != nil {
				t.Fatal(err)
			}
			parts := a.parts
			a.parts = parts[:len(parts)-1]
			without := draw()
			a.parts = parts
			n := 0
			for i := 0; i < len(full); i += 4 {
				if full[i] != without[i] || full[i+1] != without[i+1] || full[i+2] != without[i+2] {
					n++
				}
			}
			t.Logf("fit=%d back=%v shoulder contribution=%d", fit, back, n)
			if fit == 7 && n < 100 {
				t.Fatal("correct armor fit did not expose the shoulder")
			}
			cloth := a.cloak
			a.cloak = nil
			a.parts = []*skinPart{parts[len(parts)-1]}
			draw()
			if err := render.SavePNG(filepath.Join(dir, name+"-only.png"), 800, 600); err != nil {
				t.Fatal(err)
			}
			a.parts = parts
			a.cloak = cloth
		}
	}
}
func fmtName(fit uint8, back bool) string {
	n := "shoulder-fit" + string(rune('0'+fit))
	if back {
		return n + "-back"
	}
	return n + "-front"
}
