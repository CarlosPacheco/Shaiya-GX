//go:build integration

package scene

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Real market wood walls have coincident front/back quads with different
// diagonals. Microscopic camera motion must not change their lighting. Use the
// application's float depth target, not the default window's integer depth.
func TestMarketSurfaceMotionGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" || os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires market assets/GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "market surface motion", Width: 800, Height: 600, Hidden: true, Debug: true})
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
	w, err := loadStreamedField(NewAssets(arc), "1")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	center := render.V3{989.6, 15, 806}
	for range 1000 {
		w.UpdateVisible(center, 180)
		if w.Ready() {
			break
		}
	}
	if !w.Ready() {
		t.Fatal("market did not become resident")
	}
	r, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Delete()
	r.Proj = render.PerspectiveFovLH(math.Pi/4, 800.0/600, .1, 500)
	r.LightDir = render.V3{.3, -1, .4}
	r.Ambient = .4
	target, err := render.NewWaterTarget(800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Delete()
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir != "" {
		os.MkdirAll(dir, 0755)
	}
	for _, reverse := range []bool{false, true} {
		r.SetReverseDepth(reverse)
		for turn := range 8 {
			angle := float64(turn) * math.Pi / 4
			flashes := 0
			target.Capture(render.V3{.3, .4, .5}, func() {
				var previous []byte
				for frame := range 32 {
					delta := float32(frame) * .0001
					r.SetCameraView(center.Add(render.V3{float32(math.Sin(angle))*12 + delta, 5, float32(math.Cos(angle)) * 12}), center.Add(render.V3{delta, 0, 0}), render.V3{0, 1, 0})
					gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
					r.Begin()
					w.stream.draw(r)
					pixels := make([]byte, 800*600*4)
					gl.ReadPixels(0, 0, 800, 600, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
					if previous != nil {
						for p := 0; p < len(pixels); p += 4 {
							d := 0
							for ch := range 3 {
								v := int(pixels[p+ch]) - int(previous[p+ch])
								if v < 0 {
									v = -v
								}
								d += v
							}
							if d > 30 {
								flashes++
							}
						}
					}
					previous = pixels
					if frame == 0 && dir != "" {
						if err = render.SavePNG(filepath.Join(dir, fmt.Sprintf("market-turn%d-reverse%v.png", turn, reverse)), 800, 600); err != nil {
							t.Fatal(err)
						}
					}
					if gl.IsEnabled(gl.CULL_FACE) {
						t.Fatal("world surface draw leaked face culling")
					}
				}
			})
			// Silhouette edge crossings remain. Old reverse-depth market views reach
			// 259k abrupt changes, versus under 7k after isolating face pairs.
			t.Logf("reverse=%v turn=%d abrupt pixels across 32 subpixel steps=%d", reverse, turn, flashes)
			if flashes > 10000 {
				t.Errorf("coplanar market surfaces flicker during camera motion: %d", flashes)
			}
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("market GL error %#x", code)
	}
}
