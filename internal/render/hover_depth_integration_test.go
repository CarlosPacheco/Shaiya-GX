//go:build integration

package render

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
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
	capture := func(depth float64, reversed bool, width int, occluder ...[2]int32) {
		gl.Viewport(0, 0, 64, 64)
		gl.DepthMask(true)
		gl.DepthFunc(gl.LEQUAL)
		if reversed {
			gl.DepthFunc(gl.GEQUAL)
		}
		gl.ClearDepth(depth)
		gl.Clear(gl.DEPTH_BUFFER_BIT)
		if len(occluder) != 0 {
			x, y := occluder[0][0], occluder[0][1]
			gl.Enable(gl.SCISSOR_TEST)
			gl.Scissor(x, 63-y, 1, 1)
			gl.ClearDepth(0)
			gl.Clear(gl.DEPTH_BUFFER_BIT)
			gl.Disable(gl.SCISSOR_TEST)
		}
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
	if !sprites.NameHoverVisibleAt(31, 32, 64, 64, .2) || sprites.NameHoverVisibleAt(20, 32, 64, 64, .2) {
		t.Fatal("hover did not stay within the sampled region")
	}
	// Model four frames with no completed transfer while retaining the real GL sample.
	frame, issued := sprites.depthFrame, sprites.hoverDepth.issued
	capturedAt := sprites.hoverDepth.ready.capturedAt
	sprites.depthFrame = sprites.hoverDepth.ready.frame + 4
	sprites.hoverDepth.issued = sprites.depthFrame
	sprites.hoverDepth.ready.capturedAt = time.Now()
	if !sprites.NameAttackHoverVisibleAt(32, 32, 64, 64, .2) ||
		sprites.NameAttackHoverVisibleAt(31, 32, 64, 64, .2) ||
		sprites.NameHoverVisibleAt(32, 32, 64, 64, .2) {
		t.Fatal("only stationary attack hover may use a delayed depth sample")
	}
	sprites.hoverDepth.ready.capturedAt = time.Now().Add(-attackHoverGrace)
	if sprites.NameAttackHoverVisibleAt(32, 32, 64, 64, .2) {
		t.Fatal("stationary attack hover exceeded its grace period")
	}
	sprites.depthFrame, sprites.hoverDepth.issued = frame, issued
	sprites.hoverDepth.ready.capturedAt = capturedAt
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
	edgeOccluder := [2]int32{1, 0}
	capture(.25, false, 32, edgeOccluder)
	if sprites.hoverDepth != (hoverDepthReadback{}) {
		t.Fatal("resize retained old hover readback resources")
	}
	if sprites.NameHoverVisibleAt(0, 0, 32, 64, .2) {
		t.Fatal("resized hover used an unfinished transfer")
	}
	gl.Finish()
	capture(.25, false, 32, edgeOccluder)
	if !sprites.NameHoverVisibleAt(0, 0, 32, 64, .2) || !sprites.NameHoverVisibleAt(1, 1, 32, 64, .2) ||
		sprites.NameHoverVisibleAt(1, 0, 32, 64, .2) {
		t.Fatal("hover sampled the wrong pixel next to an occluder at the framebuffer edge")
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
