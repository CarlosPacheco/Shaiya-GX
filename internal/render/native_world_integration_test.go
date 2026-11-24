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

func TestOriginalWorldTexturePolicyGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "native world textures", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = Init(true); err != nil {
		t.Fatal(err)
	}
	base := make([]byte, 32)
	for i := 0; i < 4; i++ {
		copy(base[i*8:], []byte{31, 0, 31, 0, 0, 0, 0, 0})
	}
	level1 := []byte{0, 248, 0, 248, 0, 0, 0, 0}
	source := img.Texture{DDS: &img.DDS{W: 8, H: 8, Format: img.FormatDXT1, Levels: [][]byte{base, level1}}}
	tex := UploadWorld(source)
	defer tex.Delete()
	var levels, filter int32
	gl.GetTextureParameteriv(tex.ID, gl.TEXTURE_IMMUTABLE_LEVELS, &levels)
	if levels != 4 {
		t.Fatalf("world levels=%d want4", levels)
	}
	gl.GetTextureParameteriv(tex.ID, gl.TEXTURE_MIN_FILTER, &filter)
	if filter != gl.LINEAR_MIPMAP_NEAREST {
		t.Fatal("native world mip selection changed", filter)
	}
	if compressedDDS[img.FormatDXT1] {
		for i, want := range source.DDS.Levels {
			got := make([]byte, len(want))
			gl.GetCompressedTextureImage(tex.ID, int32(i), int32(len(got)), gl.Ptr(&got[0]))
			if !bytes.Equal(got, want) {
				t.Fatalf("overwrote authored level %d", i)
			}
		}
	}
	rgba := make([]byte, 4)
	gl.GetTextureImage(tex.ID, 3, gl.RGBA, gl.UNSIGNED_BYTE, 4, gl.Ptr(&rgba[0]))
	if rgba[0] < 240 || rgba[2] > 16 {
		t.Fatalf("generated level did not descend from last authored red mip: %v", rgba)
	}
	plain := Upload(source)
	defer plain.Delete()
	gl.GetTextureParameteriv(plain.ID, gl.TEXTURE_IMMUTABLE_LEVELS, &levels)
	if levels != 2 {
		t.Fatal("changed non-world texture policy")
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View, s.Proj = Identity(), Identity()
	s.Ambient = 1
	s.LightDir = V3{0, -1, 0}
	mesh := NewMesh([]Vertex{{Pos: V3{-1, -1, .5}, Normal: V3{0, 1, 0}}, {Pos: V3{1, -1, .5}, Normal: V3{0, 1, 0}}, {Pos: V3{0, 1, .5}, Normal: V3{0, 1, 0}}}, []uint32{0, 1, 2})
	defer mesh.Delete()
	for _, mani := range []bool{false, true} {
		threshold := byte(80)
		if mani {
			threshold = 8
		}
		for _, a := range []byte{threshold - 1, threshold, threshold + 1} {
			image := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, a}}})
			gl.Viewport(0, 0, 64, 64)
			gl.DepthMask(true)
			gl.ClearDepth(1)
			gl.ClearColor(0, 0, 0, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			s.Begin()
			s.DrawModel(mesh, image, Identity(), NativeWorldAlphaRef(mani), false)
			var p [4]byte
			gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
			image.Delete()
			if (p[0] > 0) != (a >= threshold) {
				t.Fatalf("mani=%v alpha=%d threshold=%d pixel=%v", mani, a, threshold, p)
			}
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("native world GL error %#x", code)
	}
}

func TestDDSUploadsWithoutS3TCGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := Init(true); err != nil {
		t.Fatal(err)
	}
	native := compressedDDS
	defer func() { compressedDDS = native }()
	configureTextureFormats(false, false)
	red := []byte{0, 248, 0, 248, 0, 0, 0, 0}
	for _, tc := range []struct {
		format img.Format
		data   []byte
		alpha  byte
	}{
		{img.FormatDXT1, red, 255},
		{img.FormatDXT3, append(bytes.Repeat([]byte{0x88}, 8), red...), 136},
		{img.FormatDXT5, append([]byte{64, 32, 0, 0, 0, 0, 0, 0}, red...), 64},
	} {
		tex := UploadWorld(img.Texture{DDS: &img.DDS{W: 4, H: 4, Format: tc.format, Levels: [][]byte{tc.data}}})
		for mip := range int32(3) {
			var format int32
			gl.GetTextureLevelParameteriv(tex.ID, mip, gl.TEXTURE_INTERNAL_FORMAT, &format)
			side := 4 >> mip
			pixels := make([]byte, side*side*4)
			gl.GetTextureImage(tex.ID, mip, gl.RGBA, gl.UNSIGNED_BYTE, int32(len(pixels)), gl.Ptr(pixels))
			want := bytes.Repeat([]byte{255, 0, 0, tc.alpha}, side*side)
			if format != gl.RGBA8 || !bytes.Equal(pixels, want) {
				t.Errorf("DDS format %d mip %d read back format %#x pixel %v", tc.format, mip, format, pixels[:4])
			}
		}
		tex.Delete()
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("DDS fallback GL error %#x", code)
	}
}
