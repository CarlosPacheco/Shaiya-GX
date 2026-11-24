//go:build integration

package render

import (
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"os"
	"runtime"
	"testing"
)

func TestWorldPairedFacesAndUnpairedCutoutsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "world face coverage", Width: 64, Height: 64, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = Init(true); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View, s.Proj = Identity(), Identity()
	s.Ambient = .25
	s.LightDir = V3{0, -1, 0}
	texture := Upload(img.Texture{RGBA: &img.RGBA{W: 2, H: 1, Pix: []byte{255, 0, 0, 0, 255, 0, 0, 255}}})
	defer texture.Delete()
	var vertices []Vertex
	for _, normal := range []V3{{0, 1, 0}, {0, -1, 0}} {
		for i, p := range []V3{{-.8, -.8, .5}, {.8, -.8, .5}, {.8, .8, .5}, {-.8, .8, .5}} {
			uv := [2]float32{}
			if i == 1 || i == 2 {
				uv[0] = 1
			}
			vertices = append(vertices, Vertex{Pos: p, Normal: normal, UV: uv})
		}
	}
	paired := NewMesh(vertices, []uint32{0, 2, 1, 0, 3, 2, 4, 5, 6, 4, 6, 7})
	paired.CullBack = true
	defer paired.Delete()
	unpaired := NewMesh(vertices[:4], []uint32{0, 1, 2, 0, 2, 3})
	defer unpaired.Delete()
	for _, reverse := range []bool{false, true} {
		for _, back := range []bool{false, true} {
			for _, instanced := range []bool{false, true} {
				gl.Viewport(0, 0, 64, 64)
				s.SetReverseDepth(reverse)
				gl.DepthMask(true)
				gl.ClearColor(0, 0, 1, 1)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				s.Begin()
				matrix := Identity()
				if back {
					matrix[0], matrix[10], matrix[14] = -1, -1, 1
				}
				// Hostile inherited cull state must not choose the inward sheet.
				gl.Enable(gl.CULL_FACE)
				gl.CullFace(gl.FRONT)
				gl.FrontFace(gl.CCW)
				if instanced {
					s.SetInstances([]M4{matrix})
					s.DrawWorldInstanced(paired, texture, .5)
				} else {
					s.DrawWorldModel(paired, texture, matrix, .5)
				}
				var pixel [4]byte
				x := int32(48)
				want := byte(255)
				if back {
					x = 16
					want = 64
				}
				gl.ReadPixels(x, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
				if (int(pixel[0])-int(want) < -1 || int(pixel[0])-int(want) > 1) || pixel[2] != 0 {
					t.Fatal("wrong front/back lighting or lost coverage", reverse, back, instanced, pixel, want)
				}
				gl.ReadPixels(64-x, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
				if pixel != [4]byte{0, 0, 255, 255} {
					t.Fatal("cutout hole became opaque", pixel)
				}
				var front int32
				gl.GetIntegerv(gl.FRONT_FACE, &front)
				if gl.IsEnabled(gl.CULL_FACE) || front != gl.CCW {
					t.Fatal("world faces leaked raster state")
				}
				// A genuine one-sided authored leaf remains visible from both directions.
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				gl.Enable(gl.CULL_FACE)
				gl.CullFace(gl.FRONT)
				s.DrawWorldModel(unpaired, texture, matrix, .5)
				gl.ReadPixels(x, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
				if pixel[0] < 250 || pixel[2] != 0 {
					t.Fatal("unpaired cutout backside was culled", reverse, back, pixel)
				}
			}
		}
	}
}
