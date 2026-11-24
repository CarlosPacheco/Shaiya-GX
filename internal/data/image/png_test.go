package image

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	stdimage "image"
	"image/color"
	"image/png"
	"testing"
)

func TestPNGPalettePreservesTransparentRGB(t *testing.T) {
	p := stdimage.NewPaletted(stdimage.Rect(0, 0, 2, 1), color.Palette{color.NRGBA{R: 200, G: 80, B: 30, A: 0}, color.NRGBA{R: 10, G: 20, B: 30, A: 255}})
	p.Pix[1] = 1
	var b bytes.Buffer
	if err := png.Encode(&b, p); err != nil {
		t.Fatal(err)
	}
	tex, err := Decode(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(tex.RGBA.Pix, []byte{200, 80, 30, 0, 10, 20, 30, 255}) {
		t.Fatal(tex.RGBA.Pix)
	}
}
func TestPNGAuthoredMipmapsAndInvalidDimensions(t *testing.T) {
	encode := func(w, h int) []byte {
		var b bytes.Buffer
		png.Encode(&b, stdimage.NewNRGBA(stdimage.Rect(0, 0, w, h)))
		return b.Bytes()
	}
	base, mip := encode(2, 2), encode(1, 1)
	pack := func(mip []byte) []byte {
		var p bytes.Buffer
		binary.Write(&p, binary.LittleEndian, uint32(1))
		binary.Write(&p, binary.LittleEndian, uint32(len(mip)))
		p.Write(mip)
		var out bytes.Buffer
		out.Write(base[:len(base)-12])
		binary.Write(&out, binary.BigEndian, uint32(p.Len()))
		out.WriteString("gxMP")
		out.Write(p.Bytes())
		sum := crc32.NewIEEE()
		sum.Write([]byte("gxMP"))
		sum.Write(p.Bytes())
		binary.Write(&out, binary.BigEndian, sum.Sum32())
		out.Write(base[len(base)-12:])
		return out.Bytes()
	}
	tex, err := Decode(pack(mip))
	if err != nil || tex.DDS == nil || len(tex.DDS.Levels) != 2 {
		t.Fatal(tex, err)
	}
	if _, err = Decode(pack(encode(2, 2))); err == nil {
		t.Fatal("wrong mip dimensions accepted")
	}
}

func TestInspectResourceClearsOnlyExteriorBlack(t *testing.T) {
	canvas := stdimage.NewNRGBA(stdimage.Rect(0, 0, 5, 5))
	black := color.NRGBA{A: 255}
	frame := color.NRGBA{R: 96, G: 64, B: 32, A: 255}
	for y := range 5 {
		for x := range 5 {
			canvas.SetNRGBA(x, y, black)
			if x >= 1 && x <= 3 && y >= 1 && y <= 3 && (x != 2 || y != 2) {
				canvas.SetNRGBA(x, y, frame)
			}
		}
	}
	canvas.SetNRGBA(0, 0, color.NRGBA{})
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		t.Fatal(err)
	}
	original := bytes.Clone(buf.Bytes())
	got, err := DecodeResource(`Interface\INSPECT.PNG`, buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		x, y  int
		pixel color.NRGBA
	}{{1, 0, color.NRGBA{}}, {1, 2, frame}, {2, 2, black}}
	for _, tc := range want {
		i := (tc.y*5 + tc.x) * 4
		p := got.RGBA.Pix[i : i+4]
		if !bytes.Equal(p, []byte{tc.pixel.R, tc.pixel.G, tc.pixel.B, tc.pixel.A}) {
			t.Errorf("pixel (%d,%d) = %v, want %v", tc.x, tc.y, p, tc.pixel)
		}
	}
	ordinary, err := DecodeResource("interface/other.png", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if ordinary.RGBA.Pix[7] != 255 || !bytes.Equal(original, buf.Bytes()) {
		t.Fatal("inspect correction altered other textures or original PNG bytes")
	}
}
