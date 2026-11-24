//go:build integration

package app

import (
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestRelicMarkersOriginalPixelsMiniAndFullMapGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1280, Height: 800, Hidden: true, Debug: true})
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
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	for _, full := range []bool{false, true} {
		for _, c := range []struct {
			file string
			kind ui.MapMarkerKind
		}{{"obelisk_0.png", ui.MapRelicNeutral}, {"obelisk_1.png", ui.MapRelicLight}, {"obelisk_2.png", ui.MapRelicFury}, {"portal.png", ui.MapRelicPortal}} {
			f, err := os.Open(filepath.Join(os.Getenv("SHAIYA_DATA"), "interface", c.file))
			if err != nil {
				t.Fatal(err)
			}
			original, err := png.Decode(f)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			gl.Viewport(0, 0, 1280, 800)
			gl.ClearColor(0, 0, 0, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			a.sprites.Begin(1280, 800)
			markers := []ui.MapMarker{{U: .5, V: .5, Kind: c.kind}}
			left, top := 0, 0
			if full {
				x, y, scale := a.minimap.FullRect(1280, 800)
				if scale != 1 {
					t.Fatal("fixture scale")
				}
				left, top = x+16+256-16, y+43+256-16
				a.minimap.DrawFull(a.sprites, a.font, a.white, a.white, 1280, 800, ui.UV{U1: 1, V1: 1}, .9, .9, 0, 0, markers)
			} else {
				x, y := a.minimap.Origin(1280, 800)
				left, top = x+68, y+68
				a.minimap.Draw(a.sprites, a.font, a.white, a.white, 1280, 800, ui.UV{U1: 1, V1: 1}, .9, .9, 0, 0, 0, 0, markers)
			}
			a.sprites.End()
			p := make([]byte, 1280*800*4)
			gl.ReadPixels(0, 0, 1280, 800, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(p))
			checked := 0
			for y := 0; y < 32; y++ {
				for x := 0; x < 32; x++ {
					r, g, b, alpha := original.At(x, y).RGBA()
					if alpha != 65535 {
						continue
					}
					at := ((800-1-(top+y))*1280 + left + x) * 4
					for ch, want := range []uint32{r, g, b} {
						d := int(p[at+ch]) - int(want>>8)
						if d < -3 || d > 3 {
							t.Fatalf("%s full=%v pixel=%d,%d channel=%d got=%d want=%d", c.file, full, x, y, ch, p[at+ch], want>>8)
						}
					}
					checked++
				}
			}
			if checked < 10 {
				t.Fatal("fixture icon lacks opaque pixels")
			}
			if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" && c.file == "obelisk_1.png" {
				name := "relic-minimap.png"
				if full {
					name = "relic-full-map.png"
				}
				if err := render.SavePNG(filepath.Join(dir, name), 1280, 800); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
