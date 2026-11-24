//go:build integration

package render

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestMapBatchDrawsAfterPackedBufferGrowthGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const side = 64
	win, err := platform.Open(platform.Config{Width: side, Height: side, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := Init(false); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	if s.mapPrograms().draw == 0 {
		t.Skip("requires indirect map drawing")
	}
	s.View, s.Proj, s.Ambient = Identity(), Identity(), 1
	triangle := []Vertex{
		{Pos: V3{-.6, -.6, .5}, Normal: V3{0, 1, 0}},
		{Pos: V3{.6, -.6, .5}, Normal: V3{0, 1, 0}},
		{Pos: V3{0, .6, .5}, Normal: V3{0, 1, 0}},
	}
	indices := []uint32{0, 1, 2}
	small := NewMesh(triangle, indices)
	defer small.Delete()
	// Both additions cross a capacity threshold in a single draw. A driver
	// can recycle the name of the first vertex buffer on the second resize.
	padded := make([]Vertex, 7000)
	copy(padded, triangle)
	first := NewMesh(padded, indices)
	defer first.Delete()
	second := NewMesh(padded, indices)
	defer second.Delete()

	b := MapBatch{Transforms: []M4{Identity()}}
	defer b.Delete()
	begin := func() {
		gl.Viewport(0, 0, side, side)
		gl.DepthMask(true)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
	}
	draw := func(meshes ...*Mesh) {
		at := b.Placements([]uint32{0})
		for _, m := range meshes {
			b.Add(m, nil, nil, -1, at, 1)
		}
		b.Draw(s)
	}
	read := func() []byte {
		pixels := make([]byte, side*side*4)
		gl.ReadPixels(0, 0, side, side, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}

	begin()
	draw(small, small, small, small)
	if b.vao == 0 {
		t.Fatal("warm-up did not use the packed draw")
	}
	originalVertex := b.vertices.id
	gl.BindVertexArray(0)
	begin()
	draw(first, second, first, second)
	got := read()
	if b.vertices.capacity < 1<<20 {
		t.Fatal("packed vertex buffer did not grow twice")
	}
	t.Logf("packed vertex buffer name %d -> %d", originalVertex, b.vertices.id)

	begin()
	s.DrawVisibleWorldModel(second, nil, Identity(), -1)
	if want := read(); want[(side*side/2+side/2)*4] == 0 || !bytes.Equal(got, want) {
		t.Fatal("indirect pixels differ from the direct draw after resizing the vertex buffer")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
