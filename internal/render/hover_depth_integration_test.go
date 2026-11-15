//go:build integration

package render

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestHoverDepthAsyncAndExactClicksGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = Init(true); err != nil {
		t.Fatal(err)
	}
	sprites, err := NewSprites(4)
	if err != nil {
		t.Fatal(err)
	}
	defer sprites.ResetHoverDepth()
	capture := func(depth float64, reversed bool, width int) {
		gl.Viewport(0, 0, 64, 64)
		gl.DepthMask(true)
		gl.DepthFunc(gl.LEQUAL)
		if reversed {
			gl.DepthFunc(gl.GEQUAL)
		}
		gl.ClearDepth(depth)
		gl.Clear(gl.DEPTH_BUFFER_BIT)
		sprites.CaptureNameDepth(width, 64)
	}
	capture(.25, false, 64)
	if sprites.NameHoverVisibleAt(32, 32, 64, 64, .2) {
		t.Fatal("hover used unfinished initial transfer")
	}
	if !sprites.NameVisibleAt(32, 32, 64, 64, .2) || sprites.NameVisibleAt(32, 32, 64, 64, .3) {
		t.Fatal("exact click lost depth occlusion")
	}
	gl.Finish() // Test-only: make the previous asynchronous transfer deterministic.
	capture(.25, false, 64)
	if !sprites.NameHoverVisibleAt(32, 32, 64, 64, .2) || sprites.NameHoverVisibleAt(32, 32, 64, 64, .3) {
		t.Fatal("completed hover sample differs from click")
	}
	if sprites.NameHoverVisibleAt(31, 32, 64, 64, .2) {
		t.Fatal("hover reused a different cursor position")
	}
	capture(0, false, 64)
	if sprites.NameVisibleAt(32, 32, 64, 64, .2) {
		t.Fatal("stale hover depth permitted a click through a new wall")
	}
	sprites.NameHoverVisibleAt(32, 32, 64, 64, .2)
	gl.Finish()
	capture(0, false, 64)
	if sprites.NameHoverVisibleAt(32, 32, 64, 64, .2) {
		t.Fatal("hover did not adopt new occluder")
	}
	capture(.75, true, 64)
	if sprites.NameHoverVisibleAt(32, 32, 64, 64, .2) {
		t.Fatal("hover reused opposite depth convention")
	}
	if !sprites.NameVisibleAt(32, 32, 64, 64, .2) || sprites.NameVisibleAt(32, 32, 64, 64, .3) {
		t.Fatal("reverse-Z click mismatch")
	}
	gl.Finish()
	capture(.75, true, 64)
	if !sprites.NameHoverVisibleAt(32, 32, 64, 64, .2) {
		t.Fatal("reverse-Z hover mismatch")
	}
	capture(.75, true, 32)
	if sprites.NameHoverVisibleAt(16, 32, 32, 64, .2) {
		t.Fatal("resize retained old hover samples")
	}
	sprites.ResetHoverDepth()
	if sprites.hoverDepth != (hoverDepthReadback{}) {
		t.Fatal("reset retained readback resources")
	}
	var binding int32
	gl.GetIntegerv(gl.PIXEL_PACK_BUFFER_BINDING, &binding)
	if binding != 0 {
		t.Fatal("hover leaked pixel pack binding", binding)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error: %#x", code)
	}
}

// This isolates the per-frame depth copy/readback cost, not whole-scene FPS.
// Each top-level benchmark owns its GL context on the benchmark's locked thread.
func BenchmarkHoverDepthSynchronous(b *testing.B)  { benchmarkHoverDepth(b, false) }
func BenchmarkHoverDepthAsynchronous(b *testing.B) { benchmarkHoverDepth(b, true) }

func benchmarkHoverDepth(b *testing.B, async bool) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		b.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1280, Height: 720, Hidden: true})
	if err != nil {
		b.Fatal(err)
	}
	defer win.Close()
	if err = Init(false); err != nil {
		b.Fatal(err)
	}
	sprites, err := NewSprites(4)
	if err != nil {
		b.Fatal(err)
	}
	defer sprites.ResetHoverDepth()
	gl.Viewport(0, 0, 1280, 720)
	gl.DepthMask(true)
	gl.DepthFunc(gl.LEQUAL)
	gl.ClearDepth(.5)
	gl.Finish()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		gl.Clear(gl.DEPTH_BUFFER_BIT)
		sprites.CaptureNameDepth(1280, 720)
		if async {
			sprites.NameHoverVisibleAt(640, 360, 1280, 720, .4)
		} else {
			sprites.NameVisibleAt(640, 360, 1280, 720, .4)
		}
		gl.Flush() // Presentation flushes the frame without a CPU wait.
	}
	gl.Finish()
	if gl.GetString(gl.VERSION) == nil {
		b.Fatal("GL context is not current on benchmark thread")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		b.Fatalf("GL error: %#x", code)
	}
}
