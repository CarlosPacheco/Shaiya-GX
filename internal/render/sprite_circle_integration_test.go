//go:build integration

package render

import (
	"os"
	"runtime"
	"testing"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

// With the UI drawn in a shrunk virtual window the circle mask must follow UI
// units, not framebuffer pixels (low-resolution minimaps were cut wrongly).
func TestCircleMaskFollowsUIScale(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "circle scale", Width: 512, Height: 384, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := Init(false); err != nil {
		t.Fatal(err)
	}
	s, err := NewSprites(64)
	if err != nil {
		t.Fatal(err)
	}
	white := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 255}}})
	read := func(x, y int) byte {
		var px [4]byte
		gl.ReadPixels(int32(x), int32(384-1-y), 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&px[0]))
		return px[3]
	}
	for _, scale := range []float64{1, .5} {
		gl.Viewport(0, 0, 512, 384)
		gl.ClearColor(0, 0, 0, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		SetUIScale(scale)
		w, h := int(512/scale), int(384/scale)
		s.Begin(w, h)
		// A 100-unit circle at UI (40,30): its centre is physical (90,65)*scale.
		s.DrawCircle(white, 40, 30, 100, 0, 0, 1, 1, [4]float32{1, 1, 1, 1})
		s.Flush()
		cx, cy := int(90*scale), int(80*scale)
		corner := int(43 * scale) // inside the quad, outside the circle
		if read(cx, cy) < 250 {
			t.Fatalf("scale %v: circle centre is clipped away (%d)", scale, read(cx, cy))
		}
		if a := read(int(40*scale)+corner/8, int(30*scale)+corner/8); a > 10 {
			t.Fatalf("scale %v: corner of the quad stayed visible (%d)", scale, a)
		}
		if a := read(int(40*scale)+int(98*scale), cy); a < 250 && scale == 1 {
			t.Logf("scale %v: right edge alpha %d", scale, a)
		}
	}
	SetUIScale(1)
}
