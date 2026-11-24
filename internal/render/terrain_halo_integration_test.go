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

func TestTerrainHaloMovesWithFeetGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires a local OpenGL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const size = 128
	window, err := platform.Open(platform.Config{Width: size, Height: size, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := Init(true); err != nil {
		t.Fatal(err)
	}
	scene, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer scene.Delete()
	scene.SetCameraView(V3{0, 20, -20}, V3{}, V3{0, 1, 0})
	scene.Proj = PerspectiveFovLH(.75, 1, .1, 100)
	mesh := NewMesh([]Vertex{
		{Pos: V3{-12, 0, -12}}, {Pos: V3{12, 0, -12}},
		{Pos: V3{12, 0, 12}}, {Pos: V3{-12, 0, 12}},
	}, []uint32{0, 1, 2, 0, 2, 3})
	defer mesh.Delete()
	tile := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{64, 64, 64, 255}}})
	defer tile.Delete()
	halo := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{229, 214, 165, 68}}})
	defer halo.Delete()
	layers := []TerrainSplat{{Mask: tile, Tile: tile, TileSize: 32}}
	gl.Viewport(0, 0, size, size)
	for _, ranges := range []int{0, 2} {
		scene.SetTerrainRanges([]TerrainRange{{First: 0, Count: 3}, {First: 3, Count: 3}})
		for _, reverse := range []bool{false, true} {
			scene.ReverseDepth = reverse
			for _, frame := range []struct {
				texture *Texture
				feet    V3
				want    [2][3]byte
			}{
				{nil, V3{}, [2][3]byte{{64, 64, 64}, {64, 64, 64}}},
				{halo, V3{}, [2][3]byte{{125, 121, 108}, {64, 64, 64}}},
				{halo, V3{6, 0, 0}, [2][3]byte{{64, 64, 64}, {125, 121, 108}}},
			} {
				scene.SetTerrainHalo(frame.texture, frame.feet)
				scene.Begin()
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				scene.DrawTerrainSplats(mesh, layers, nil, nil, ranges)
				for i, x := range []int32{size / 2, size/2 + 34} {
					var rgba [4]byte
					gl.ReadPixels(x, size/2, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&rgba[0]))
					for channel, want := range frame.want[i] {
						if delta := int(rgba[channel]) - int(want); delta < -1 || delta > 1 {
							t.Fatalf("ranges=%d reverse=%t halo=%t feet=%v x=%d: got %v, want RGB %v", ranges, reverse, frame.texture != nil, frame.feet, x, rgba, frame.want[i])
						}
					}
				}
				if code := gl.GetError(); code != gl.NO_ERROR {
					t.Fatalf("GL error %#x", code)
				}
				scene.End()
			}
		}
	}
}
