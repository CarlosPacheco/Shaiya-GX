package app

import img "github.com/Spelunkern/Shaiya-GX/internal/data/image"

// A small neutral checker identifies unavailable artwork without hiding labels.
func missingInterfaceTexture() img.Texture {
	const size = 256
	pixels := make([]byte, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			i := (y*size + x) * 4
			shade := byte(48)
			if (x/16+y/16)%2 == 0 {
				shade = 64
			}
			pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = shade, shade, shade, 220
		}
	}
	return img.Texture{RGBA: &img.RGBA{W: size, H: size, Pix: pixels}}
}
