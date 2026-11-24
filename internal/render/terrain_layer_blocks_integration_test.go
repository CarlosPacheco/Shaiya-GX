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

func TestTerrainLayerBlocksKeepLogicalSlotsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires a local OpenGL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const size = 96
	window, err := platform.Open(platform.Config{Width: size, Height: size, Hidden: true})
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
	mesh := NewMesh([]Vertex{
		{Pos: V3{-1, -1, .5}, UV2: [2]float32{0, 0}},
		{Pos: V3{1, -1, .5}, UV2: [2]float32{size, 0}},
		{Pos: V3{1, 1, .5}, UV2: [2]float32{size, size}},
		{Pos: V3{-1, 1, .5}, UV2: [2]float32{0, size}},
	}, []uint32{0, 1, 2, 0, 2, 3})
	defer mesh.Delete()
	solid := func(red, green, blue byte) *Texture {
		return Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{red, green, blue, 255}}})
	}
	mask, blue, green, red := solid(255, 255, 255), solid(0, 0, 255), solid(0, 255, 0), solid(255, 0, 0)
	defer mask.Delete()
	defer blue.Delete()
	defer green.Delete()
	defer red.Delete()
	layers := []TerrainSplat{
		{Mask: mask, Tile: blue, TileSize: 32, LayerBit: 1},
		{Mask: mask, Tile: green, TileSize: 32, LayerBit: 1 << 2},
		{Mask: mask, Tile: red, TileSize: 32, LayerBit: 1 << 6},
	}
	blocks := []byte{0, 4, 64, 4, 68, 0, 64, 0, 4}
	selection := UploadTerrainLayerBlocks(3, blocks)
	defer selection.Delete()
	for _, mode := range []string{"individual", "eager", "streamed"} {
		ResetDrawCalls()
		gl.Viewport(0, 0, size, size)
		scene.Begin()
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		switch mode {
		case "individual":
			for i, layer := range layers {
				scene.DrawTerrainLayer(mesh, layer, nil, selection, i == 0)
			}
		case "eager":
			scene.DrawTerrainSplats(mesh, layers, nil, selection, 0)
		case "streamed":
			scene.SetTerrainRanges([]TerrainRange{{First: 0, Count: 3}, {First: 3, Count: 3}})
			scene.DrawTerrainSplats(mesh, layers, nil, selection, 2)
		}
		if mode != "individual" && drawParametersSupported && DrawCalls() != 2 {
			t.Fatalf("%s did not batch the two overlays: %d draws", mode, DrawCalls())
		}
		pixels := make([][4]byte, size*size)
		gl.ReadPixels(0, 0, size, size, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0][0]))
		for y := range size {
			for x := range size {
				want := [4]byte{0, 0, 255, 255}
				switch enabled := blocks[(y/32)*3+x/32]; {
				case enabled&64 != 0:
					want = [4]byte{255, 0, 0, 255}
				case enabled&4 != 0:
					want = [4]byte{0, 255, 0, 255}
				}
				if pixel := pixels[y*size+x]; pixel != want {
					t.Fatalf("%s (%d,%d): got %v, want %v", mode, x, y, pixel, want)
				}
			}
		}
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("%s GL error %#x", mode, code)
		}
		scene.End()
	}
}
