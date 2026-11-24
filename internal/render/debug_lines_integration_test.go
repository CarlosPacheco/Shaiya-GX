//go:build integration

package render

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"os"
	"runtime"
	"testing"
)

func TestCollisionLinesDepthAndStateGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err = Init(true); err != nil {
		t.Fatal(err)
	}
	scene, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer scene.Delete()
	scene.View, scene.Proj = Identity(), Identity()
	lines, err := NewDebugLines()
	if err != nil {
		t.Fatal(err)
	}
	defer lines.Delete()
	vertices := []DebugLineVertex{{V3{-.8, 0, .5}, [4]float32{1, 0, 1, 1}}, {V3{.8, 0, .5}, [4]float32{1, 0, 1, 1}}}
	for _, reverse := range []bool{false, true} {
		scene.SetReverseDepth(reverse)
		gl.Viewport(0, 0, 64, 64)
		gl.DepthMask(true)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		scene.Begin()
		gl.Enable(gl.CULL_FACE)
		gl.Enable(gl.CLIP_DISTANCE0)
		gl.Enable(gl.BLEND)
		gl.BlendFuncSeparate(gl.ONE, gl.ZERO, gl.ZERO, gl.ONE)
		gl.DepthMask(false)
		var program, vao int32
		gl.GetIntegerv(gl.CURRENT_PROGRAM, &program)
		gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &vao)
		lines.Draw(scene, vertices)
		var gotProgram, gotVAO, src, dst int32
		var mask bool
		gl.GetIntegerv(gl.CURRENT_PROGRAM, &gotProgram)
		gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &gotVAO)
		gl.GetBooleanv(gl.DEPTH_WRITEMASK, &mask)
		gl.GetIntegerv(gl.BLEND_SRC_RGB, &src)
		gl.GetIntegerv(gl.BLEND_DST_ALPHA, &dst)
		if !gl.IsEnabled(gl.CULL_FACE) || !gl.IsEnabled(gl.CLIP_DISTANCE0) || !gl.IsEnabled(gl.DEPTH_TEST) || !gl.IsEnabled(gl.BLEND) || mask || program != gotProgram || vao != gotVAO || src != gl.ONE || dst != gl.ONE {
			t.Fatal("collision view leaked GL state")
		}
		pixels := make([]byte, 64*64*4)
		gl.ReadPixels(0, 0, 64, 64, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		found := false
		for i := 0; i < len(pixels); i += 4 {
			if pixels[i] > 200 && pixels[i+2] > 200 && pixels[i+1] < 20 {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("collision lines not visible", reverse)
		}
		if err := gl.GetError(); err != gl.NO_ERROR {
			t.Fatalf("GL error %#x", err)
		}
		gl.Disable(gl.CLIP_DISTANCE0)
		gl.Disable(gl.CULL_FACE)
		gl.DepthMask(true)
		scene.End()
	}
}
