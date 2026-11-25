//go:build integration

package scene

import (
	"cmp"
	"math"
	"os"
	"runtime"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestVANIWorldGroupAlphaGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "VANI materials", Width: 128, Height: 128, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("world/select_a.wld")
	if err != nil {
		t.Fatal(err)
	}
	wld, err := world.ParseWLD(raw)
	if err != nil {
		t.Fatal(err)
	}
	assets := NewAssets(arc)
	selection := &World{WLD: wld}
	selection.addVanis(assets)
	defer selection.Close()
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View, s.Proj, s.Ambient = render.Identity(), render.Identity(), 1
	const size = 128
	indices := []uint32{0, 1, 2}
	background := [4]byte{17, 37, 73, 255}
	cover := render.NewMesh([]render.Vertex{
		{Pos: render.V3{.26, .26, .2}}, {Pos: render.V3{.74, .26, .2}}, {Pos: render.V3{.5, .74, .2}},
	}, indices)
	defer cover.Delete()
	for _, file := range []string{"L_A1_Skycloth_cloth.dds", "L_A1_Login_Fage_Glassine.dds"} {
		t.Log(file)
		tex := assets.FindTexture(file)
		if tex == nil {
			t.Fatal("missing original texture")
		}
		found := false
		for _, placement := range selection.blendedVani {
			for _, part := range placement.asset.parts {
				found = found || part.texture == tex && len(placement.asset.worlds) == 0
			}
		}
		if !found {
			t.Fatal("original selection WLD did not place this material in the blended group")
		}
		pixels := make([]byte, 4*tex.W*tex.H)
		gl.GetTextureImage(tex.ID, 0, gl.RGBA, gl.UNSIGNED_BYTE, int32(len(pixels)), gl.Ptr(&pixels[0]))
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("read texture %dx%d: GL error %#x", tex.W, tex.H, code)
		}
		var uv [2]float32
		var texel [4]byte
		for i := 0; i < len(pixels); i += 4 {
			if alpha := pixels[i+3]; alpha >= 40 && alpha < 70 && pixels[i] > 100 {
				copy(texel[:], pixels[i:i+4])
				uv = [2]float32{(float32(i/4%tex.W) + .5) / float32(tex.W), (float32(i/4/tex.W) + .5) / float32(tex.H)}
				break
			}
		}
		if texel[3] == 0 {
			t.Fatal("missing low-alpha texels discarded by the old draw")
		}
		frame := []world.Vertex{
			{Pos: world.Vec3{-.24, -.24, .5}, Normal: world.Vec3{0, 1, 0}, UV: uv},
			{Pos: world.Vec3{.24, -.24, .5}, Normal: world.Vec3{0, 1, 0}, UV: uv},
			{Pos: world.Vec3{0, .24, .5}, Normal: world.Vec3{0, 1, 0}, UV: uv},
		}
		asset := &vaniAsset{frame: -1, count: 1, parts: []vaniPart{{
			mesh: render.NewAnimatedMesh(len(frame), indices), texture: tex,
			frames: [][]world.Vertex{frame}, vertices: make([]render.Vertex, len(frame)),
		}}, worlds: []render.M4{translate(render.V3{-.5, -.5, 0})}}
		defer asset.close()
		mapScene := &World{vanis: []*vaniAsset{asset}, blendedVani: []vaniPlacement{
			{asset: asset, matrix: translate(render.V3{.5, -.5, 0})},
			{asset: asset, matrix: translate(render.V3{.5, .5, 0})},
		}}
		gl.Viewport(0, 0, size, size)
		gl.ClearDepth(1)
		gl.ClearColor(float32(background[0])/255, float32(background[1])/255, float32(background[2])/255, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		mapScene.DrawOpaque(s)
		s.DrawSolid(cover, [4]float32{0, 1, 0, 1})
		mapScene.DrawTranslucentVANI(s)
		read := func(x, y int32) [4]byte {
			var pixel [4]byte
			gl.ReadPixels(x, y, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
			return pixel
		}
		if cutout := read(32, 32); cutout != background {
			t.Fatalf("cutout group kept a fragment below 80: %v", cutout)
		}
		if covered := read(96, 96); covered != [4]byte{0, 255, 0, 255} {
			t.Fatalf("blended VANI drew over a nearer opaque surface: %v", covered)
		}
		blended := read(96, 32)
		for c := range 3 {
			a := float64(texel[3]) / 255
			want := float64(texel[c])*a + float64(background[c])*(1-a)
			if math.Abs(float64(blended[c])-want) > 2 {
				t.Fatalf("sample %v over %v gave %v; channel %d wants %.1f", texel, background, blended, c, want)
			}
		}
		var depth float32
		gl.ReadPixels(96, 32, 1, 1, gl.DEPTH_COMPONENT, gl.FLOAT, gl.Ptr(&depth))
		var depthWrite bool
		gl.GetBooleanv(gl.DEPTH_WRITEMASK, &depthWrite)
		if depth != 1 || !depthWrite || gl.IsEnabled(gl.BLEND) {
			t.Fatalf("blended group did not preserve depth state: depth %f, depth writes %v", depth, depthWrite)
		}
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("GL error %#x", code)
		}
		t.Logf("source %v gives blended pixel %v", texel, blended)
	}
}

func TestTranslucentVANIBatchingPixelParityGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	for _, tc := range []struct {
		name            string
		disableBindless bool
		disableDrawParm bool
	}{
		{"bindless", false, false},
		{"bound", true, false},
		{"core", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			const width, height = 640, 360
			win, err := platform.Open(platform.Config{Title: "VANI batch parity", Width: width, Height: height, Hidden: true, Debug: true})
			if err != nil {
				t.Fatal(err)
			}
			defer win.Close()
			if err := render.InitWithOptions(render.InitOptions{
				DisableBindless:       tc.disableBindless,
				DisableDrawParameters: tc.disableDrawParm,
				Debug:                 true,
			}); err != nil {
				t.Fatal(err)
			}
			arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
			if err != nil {
				t.Fatal(err)
			}
			defer arc.Close()
			raw, err := arc.ReadFile("world/2.wld")
			if err != nil {
				t.Fatal(err)
			}
			wld, err := world.ParseWLD(raw)
			if err != nil {
				t.Fatal(err)
			}
			assets := NewAssets(arc)
			w := &World{WLD: wld, FogColor: render.V3{wld.FogColor[0] / 255, wld.FogColor[1] / 255, wld.FogColor[2] / 255}, FogStart: wld.FogStart, FogEnd: wld.FogEnd}
			w.addVanis(assets)
			defer w.Close()

			s, err := render.NewScene3D()
			if err != nil {
				t.Fatal(err)
			}
			defer s.Delete()
			eye := render.V3{1300, 60, 1100}
			s.SetCameraView(eye, eye.Add(render.V3{0.6, -0.1, 0.8}), render.V3{0, 1, 0})
			s.Proj = render.PerspectiveFovLH(math.Pi/4, float32(width)/float32(height), .1, max(float32(400), w.FogEnd))
			s.FogColor, s.FogStart, s.FogEnd = w.FogColor, w.FogStart, w.FogEnd

			// Collect visible placements once using the exact same culling + sorting as DrawTranslucentVANI.
			f := render.ViewFrustum(s.View, s.VisibilityProjection())
			var visible []vaniPlacement
			for _, placement := range w.blendedVani {
				asset, matrix := placement.asset, placement.matrix
				center := transformPoint(asset.center, matrix)
				if asset.visible(s, f, matrix, center) {
					depth := center[0]*s.View[2] + center[1]*s.View[6] + center[2]*s.View[10]
					visible = append(visible, vaniPlacement{asset: asset, matrix: matrix, depth: depth})
				}
			}
			if len(visible) < 100 {
				t.Fatalf("expected dense VaniB field view, got only %d visible placements", len(visible))
			}
			slices.SortStableFunc(visible, func(a, b vaniPlacement) int { return cmp.Compare(b.depth, a.depth) })

			begin := func() {
				gl.Viewport(0, 0, width, height)
				gl.DepthMask(true)
				gl.ClearColor(.13, .17, .2, 1)
				gl.ClearDepth(1)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				s.Begin()
			}
			read := func() []byte {
				pixels := make([]byte, width*height*4)
				gl.ReadPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
				return pixels
			}

			for _, elapsed := range []float64{0, 0.25} {
				w.elapsed = elapsed

				// Serial reference matching the previous DrawTranslucentVANI loop.
				begin()
				render.ResetDrawCalls()
				s.BeginTranslucentWorldModels()
				var runMatrices []render.M4
				for i := 0; i < len(visible); {
					asset := visible[i].asset
					asset.updatePose(w.elapsed)
					runLen := 1
					if len(asset.parts) == 1 {
						for i+runLen < len(visible) && visible[i+runLen].asset == asset {
							runLen++
						}
					}
					runMatrices = runMatrices[:0]
					for _, p := range visible[i : i+runLen] {
						runMatrices = append(runMatrices, p.matrix)
					}
					for _, part := range asset.parts {
						s.DrawTranslucentWorldModel(part.mesh, part.texture, runMatrices, !asset.unlit)
					}
					i += runLen
				}
				s.EndTranslucentWorldModels()
				serialCalls := render.DrawCalls()
				want := read()

				// Batched DrawTranslucentVANI.
				begin()
				render.ResetDrawCalls()
				w.DrawTranslucentVANI(s)
				batchedCalls := render.DrawCalls()
				got := read()

				different, largest := 0, 0
				for i, v := range want {
					d := int(v) - int(got[i])
					if d < 0 {
						d = -d
					}
					if d != 0 {
						different++
					}
					largest = max(largest, d)
				}
				t.Logf("path=%s elapsed=%.2f visible=%d draws=%d -> %d differing_channels=%d max_diff=%d",
					tc.name, elapsed, len(visible), serialCalls, batchedCalls, different, largest)
				if batchedCalls > 2 {
					t.Fatalf("expected <= 2 draw calls for batched VaniB, got %d (serial was %d)", batchedCalls, serialCalls)
				}
				if serialCalls <= batchedCalls {
					t.Fatalf("expected batched draws (%d) < serial draws (%d)", batchedCalls, serialCalls)
				}
				if largest > 1 || different > len(want)/100_000 {
					t.Fatalf("batched VaniB pixels differ: %d channels (max %d)", different, largest)
				}
			}
			if code := gl.GetError(); code != gl.NO_ERROR {
				t.Fatalf("GL error %#x", code)
			}
		})
	}
}
