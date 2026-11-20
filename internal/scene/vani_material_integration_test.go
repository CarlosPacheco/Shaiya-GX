//go:build integration

package scene

import (
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
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
