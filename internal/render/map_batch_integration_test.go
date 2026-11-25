//go:build integration

package render

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestBoundMapBatchPreservesMaterialsAndOrderGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	t.Run("automatic", func(t *testing.T) { testBoundMapBatchPreservesMaterialsAndOrderGL(t, false) })
	t.Run("core", func(t *testing.T) { testBoundMapBatchPreservesMaterialsAndOrderGL(t, true) })
}

func testBoundMapBatchPreservesMaterialsAndOrderGL(t *testing.T, core bool) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const width, height = 128, 32
	win, err := platform.Open(platform.Config{Width: width, Height: height, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := InitWithOptions(InitOptions{DisableBindless: true, DisableDrawParameters: core, Debug: true}); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	if !mapIndirectSupported {
		t.Skip("requires shader storage blocks")
	}
	if s.mapPrograms().bound == 0 {
		t.Fatal("bound map program failed to link")
	}
	if core && !s.mapPrograms().materialAttribute {
		t.Fatal("bound map program did not select the core material attribute")
	}
	s.View, s.Proj, s.Ambient = Identity(), Identity(), 1
	batch := MapBatch{Transforms: []M4{Identity(), Identity()}}
	batch.Transforms[1][13] = .8
	allPlacements := batch.Placements([]uint32{0, 1})
	firstPlacement := batch.Placements([]uint32{0})
	defer batch.Delete()
	light := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{128, 255, 255, 255}}})
	defer light.Delete()
	for i := range drawTextureLimit + 3 {
		x := -.94 + float32(i%drawTextureLimit)*.12
		vertices := []Vertex{
			{Pos: V3{x - .05, -.75, .5}, Normal: V3{0, 1, 0}},
			{Pos: V3{x + .05, -.75, .5}, Normal: V3{0, 1, 0}},
			{Pos: V3{x, .75, .5}, Normal: V3{0, 1, 0}},
		}
		mesh := NewMesh(vertices, []uint32{0, 1, 2})
		defer mesh.Delete()
		pixels := []byte{byte(10 + 11*i), 96, 160, 255}
		if i == drawTextureLimit+1 {
			pixels[3] = 0
		}
		texture := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: pixels}})
		defer texture.Delete()
		var lightmap *Texture
		if i%5 == 0 {
			lightmap = light
		}
		if lightmap != nil {
			batch.Add(mesh, texture, lightmap, NativeWorldAlphaRef(true), firstPlacement, 1)
		} else {
			batch.Add(mesh, texture, nil, NativeWorldAlphaRef(true), allPlacements, 2)
		}
	}
	begin := func() {
		gl.Viewport(0, 0, width, height)
		gl.DepthMask(true)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
	}
	read := func() []byte {
		pixels := make([]byte, width*height*4)
		gl.ReadPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	begin()
	serial := append([]mapDraw(nil), batch.draws...)
	for _, d := range serial {
		if d.lightmap != nil {
			s.drawLightmapped(d.mesh, d.tex, d.lightmap, true)
		} else {
			for _, id := range batch.visible[d.first : d.first+d.count] {
				s.DrawVisibleWorldModel(d.mesh, d.tex, batch.Transforms[id], d.alpha)
			}
		}
	}
	want := read()
	begin()
	ResetDrawCalls()
	batch.Draw(s)
	if calls := DrawCalls(); calls != 2 {
		t.Fatalf("expected two ordered submissions after filling the texture units, got %d", calls)
	}
	if got := read(); !bytes.Equal(got, want) {
		t.Fatal("bound batches differ from direct draws")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}

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
	if p := s.mapPrograms(); p.draw == 0 && p.bound == 0 {
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

func TestMapBatchPartialFallbackPreservesPixelsAndBatchingGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const width, height = 128, 32
	win, err := platform.Open(platform.Config{Width: width, Height: height, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := Init(true); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	if p := s.mapPrograms(); p.draw == 0 && p.bound == 0 {
		t.Skip("requires indirect map drawing")
	}
	s.View, s.Proj, s.Ambient = Identity(), Identity(), 1

	const smallCount = 6
	batch := MapBatch{
		Transforms:  []M4{Identity()},
		maxVertices: smallCount * 3 * vertexSize,
	}
	defer batch.Delete()
	placement := batch.Placements([]uint32{0})

	for i := range smallCount {
		x := -.85 + float32(i)*.24
		verts := []Vertex{
			{Pos: V3{x - .08, -.7, .5}, Normal: V3{0, 1, 0}},
			{Pos: V3{x + .08, -.7, .5}, Normal: V3{0, 1, 0}},
			{Pos: V3{x, .7, .5}, Normal: V3{0, 1, 0}},
		}
		mesh := NewMesh(verts, []uint32{0, 1, 2})
		defer mesh.Delete()
		tex := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{byte(40 + 30*i), 110, 200, 255}}})
		defer tex.Delete()
		batch.Add(mesh, tex, nil, -1, placement, 1)
	}
	// One extra mesh exceeds maxVertices and must fall back serially without
	// collapsing the six packed draws out of indirect rendering.
	overflowVerts := make([]Vertex, 64)
	overflowVerts[0] = Vertex{Pos: V3{.55, -.7, .4}, Normal: V3{0, 1, 0}}
	overflowVerts[1] = Vertex{Pos: V3{.9, -.7, .4}, Normal: V3{0, 1, 0}}
	overflowVerts[2] = Vertex{Pos: V3{.72, .7, .4}, Normal: V3{0, 1, 0}}
	overflowMesh := NewMesh(overflowVerts, []uint32{0, 1, 2})
	defer overflowMesh.Delete()
	overflowTex := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{250, 210, 40, 255}}})
	defer overflowTex.Delete()
	batch.Add(overflowMesh, overflowTex, nil, -1, placement, 1)

	begin := func() {
		gl.Viewport(0, 0, width, height)
		gl.DepthMask(true)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
	}
	read := func() []byte {
		pixels := make([]byte, width*height*4)
		gl.ReadPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}

	begin()
	for _, d := range batch.draws {
		s.DrawVisibleWorldModel(d.mesh, d.tex, batch.Transforms[batch.visible[d.first]], d.alpha)
	}
	want := read()

	begin()
	ResetDrawCalls()
	batch.Draw(s)
	if calls := DrawCalls(); calls != 2 {
		t.Fatalf("expected 1 indirect + 1 serial draw call, got %d", calls)
	}
	if got := read(); !bytes.Equal(got, want) {
		t.Fatal("partial fallback pixels differ from serial reference")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}

func TestMapBatchTranslucentPixelParityGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
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
			const width, height = 96, 96
			win, err := platform.Open(platform.Config{Width: width, Height: height, Hidden: true, Debug: true})
			if err != nil {
				t.Fatal(err)
			}
			defer win.Close()
			if err := InitWithOptions(InitOptions{
				DisableBindless:       tc.disableBindless,
				DisableDrawParameters: tc.disableDrawParm,
				Debug:                 true,
			}); err != nil {
				t.Fatal(err)
			}
			s, err := NewScene3D()
			if err != nil {
				t.Fatal(err)
			}
			defer s.Delete()
			if !mapIndirectSupported {
				t.Skip("requires shader storage blocks")
			}
			p := s.mapPrograms()
			if p.draw == 0 && p.bound == 0 {
				t.Fatal("map indirect program failed to link")
			}
			if tc.disableDrawParm && !p.materialAttribute {
				t.Fatal("expected core material attribute path")
			}
			s.View = Identity()
			s.Proj = Identity()
			s.LightDir = V3{-1, -1, -1}
			s.Ambient = 0.35
			s.FogColor = V3{0.2, 0.3, 0.4}
			s.FogStart, s.FogEnd = 0.2, 0.9
			s.ClipPlane = [4]float32{1, 0, 0, 0.45} // clip x < -0.45

			// Create 4 overlapping semi-transparent quads at decreasing depths (back-to-front),
			// mixing lit/unlit and distinct textures, plus one mutable mesh for ordered fallback.
			makeQuad := func(z float32, nrm V3) []Vertex {
				return []Vertex{
					{Pos: V3{-0.7, -0.7, z}, Normal: nrm, UV: [2]float32{0, 0}},
					{Pos: V3{0.7, -0.7, z}, Normal: nrm, UV: [2]float32{1, 0}},
					{Pos: V3{0.7, 0.7, z}, Normal: nrm, UV: [2]float32{1, 1}},
					{Pos: V3{-0.7, 0.7, z}, Normal: nrm, UV: [2]float32{0, 1}},
				}
			}
			quadIdx := []uint32{0, 1, 2, 0, 2, 3}
			colors := [][4]byte{
				{220, 60, 40, 140},
				{40, 200, 80, 130},
				{50, 90, 230, 150},
				{230, 210, 50, 120},
				{200, 60, 220, 110},
			}
			var meshes []*Mesh
			var textures []*Texture
			for i, col := range colors {
				z := 0.8 - float32(i)*0.12
				nrm := V3{0, 1, 0}
				if i%2 == 1 {
					nrm = V3{0.4, 0.6, 0.7}
				}
				m := NewMesh(makeQuad(z, nrm), quadIdx)
				m.CullBack = true // verify translucent keeps two-sided faces
				defer m.Delete()
				meshes = append(meshes, m)
				tex := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: col[:]}})
				defer tex.Delete()
				textures = append(textures, tex)
			}
			mutableVerts := makeQuad(0.56, V3{0, 1, 0})
			mutableMesh := NewAnimatedMesh(len(mutableVerts), quadIdx)
			mutableMesh.UpdateVertices(mutableVerts)
			defer mutableMesh.Delete()

			begin := func() {
				gl.Viewport(0, 0, width, height)
				gl.DepthMask(true)
				gl.ClearColor(0.1, 0.15, 0.2, 1)
				gl.ClearDepth(1)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				s.Begin()
			}
			read := func() []byte {
				pixels := make([]byte, width*height*4)
				gl.ReadPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
				return pixels
			}

			// Case 1: All immutable meshes -> 1 indirect draw call, pixel-identical to serial.
			type layer struct {
				mesh *Mesh
				tex  *Texture
				lit  bool
			}
			allImmutable := []layer{
				{meshes[0], textures[0], true},
				{meshes[1], textures[1], false},
				{meshes[2], textures[2], true},
				{meshes[3], textures[3], false},
				{meshes[4], textures[4], true},
			}
			begin()
			s.BeginTranslucentWorldModels()
			for _, l := range allImmutable {
				s.DrawTranslucentWorldModel(l.mesh, l.tex, []M4{Identity()}, l.lit)
			}
			s.EndTranslucentWorldModels()
			want := read()

			var batch MapBatch
			defer batch.Delete()
			batch.Transforms = []M4{Identity()}
			p0 := batch.Placements([]uint32{0})
			for _, l := range allImmutable {
				batch.AddTranslucent(l.mesh, l.tex, l.lit, p0, 1)
			}
			begin()
			ResetDrawCalls()
			batch.DrawTranslucent(s)
			if calls := DrawCalls(); calls != 1 {
				t.Fatalf("expected 1 indirect draw call for 5 translucent layers, got %d", calls)
			}
			var depthMask bool
			gl.GetBooleanv(gl.DEPTH_WRITEMASK, &depthMask)
			if !depthMask || gl.IsEnabled(gl.BLEND) || gl.IsEnabled(gl.CULL_FACE) {
				t.Fatalf("GL state not restored after DrawTranslucent: depthMask=%v blend=%v cull=%v",
					depthMask, gl.IsEnabled(gl.BLEND), gl.IsEnabled(gl.CULL_FACE))
			}
			if got := read(); !bytes.Equal(got, want) {
				t.Fatal("batched translucent pixels differ from serial DrawTranslucentWorldModel")
			}

			// Case 2: Mutable mesh in the middle -> 3 ordered draw calls [indirect, serial, indirect],
			// preserving exact back-to-front alpha compositing.
			withMutable := []layer{
				{meshes[0], textures[0], true},
				{meshes[1], textures[1], false},
				{mutableMesh, textures[2], true},
				{meshes[3], textures[3], false},
				{meshes[4], textures[4], true},
			}
			begin()
			s.BeginTranslucentWorldModels()
			for _, l := range withMutable {
				s.DrawTranslucentWorldModel(l.mesh, l.tex, []M4{Identity()}, l.lit)
			}
			s.EndTranslucentWorldModels()
			wantMixed := read()

			batch.ResetTransforms()
			p1 := batch.Placements([]uint32{0})
			for _, l := range withMutable {
				batch.AddTranslucent(l.mesh, l.tex, l.lit, p1, 1)
			}
			begin()
			ResetDrawCalls()
			batch.DrawTranslucent(s)
			if calls := DrawCalls(); calls != 3 {
				t.Fatalf("expected 3 ordered draw calls (indirect, serial, indirect), got %d", calls)
			}
			if got := read(); !bytes.Equal(got, wantMixed) {
				t.Fatal("ordered fallback translucent pixels differ from serial reference")
			}
			if code := gl.GetError(); code != gl.NO_ERROR {
				t.Fatalf("GL error %#x", code)
			}
		})
	}
}
