package image

import "encoding/binary"

// Opaque requires full alpha coverage at every authored mip. Unknown or cutout
// textures must never turn a window, grate or doorway into an occluding wall.
func Opaque(t Texture) bool {
	if t.DDS == nil {
		return t.RGBA != nil && opaqueRGBA(t.RGBA.Pix)
	}
	d := t.DDS
	if len(d.Levels) == 0 {
		return false
	}
	for _, data := range d.Levels {
		switch d.Format {
		case FormatRGBA:
			if !opaqueRGBA(data) {
				return false
			}
		case FormatDXT1:
			if len(data) == 0 || len(data)%8 != 0 {
				return false
			}
			for i := 0; i < len(data); i += 8 {
				if binary.LittleEndian.Uint16(data[i:]) > binary.LittleEndian.Uint16(data[i+2:]) {
					continue
				}
				bits := binary.LittleEndian.Uint32(data[i+4:])
				for p := 0; p < 16; p++ {
					if bits&3 == 3 {
						return false
					}
					bits >>= 2
				}
			}
		case FormatDXT3:
			if len(data) == 0 || len(data)%16 != 0 {
				return false
			}
			for i := 0; i < len(data); i += 16 {
				for _, a := range data[i : i+8] {
					if a != 255 {
						return false
					}
				}
			}
		case FormatDXT5:
			if len(data) == 0 || len(data)%16 != 0 {
				return false
			}
			for i := 0; i < len(data); i += 16 {
				a, b := data[i], data[i+1]
				var bits uint64
				for j := 0; j < 6; j++ {
					bits |= uint64(data[i+2+j]) << uint(8*j)
				}
				for p := 0; p < 16; p++ {
					index := bits & 7
					bits >>= 3
					opaque := index == 0 && a == 255 || index == 1 && b == 255 || a == 255 && b == 255 && index != 6 || a <= b && index == 7
					if !opaque {
						return false
					}
				}
			}
		default:
			return false
		}
	}
	return true
}
func opaqueRGBA(data []byte) bool {
	if len(data) == 0 || len(data)%4 != 0 {
		return false
	}
	for i := 3; i < len(data); i += 4 {
		if data[i] != 255 {
			return false
		}
	}
	return true
}

// TerrainMaskEmpty proves that every alpha sample is below the terrain overlay
// discard threshold. Interpolation and generated mips cannot exceed it.
func TerrainMaskEmpty(t Texture) bool {
	const limit = 7
	noRGBA := func(data []byte) bool {
		if len(data) == 0 || len(data)%4 != 0 {
			return false
		}
		for i := 3; i < len(data); i += 4 {
			if data[i] > limit {
				return false
			}
		}
		return true
	}
	if t.DDS == nil {
		return t.RGBA != nil && noRGBA(t.RGBA.Pix)
	}
	if len(t.DDS.Levels) == 0 {
		return false
	}
	for _, data := range t.DDS.Levels {
		switch t.DDS.Format {
		case FormatRGBA:
			if !noRGBA(data) {
				return false
			}
		case FormatDXT1:
			if len(data) == 0 || len(data)%8 != 0 {
				return false
			}
			for i := 0; i < len(data); i += 8 {
				if binary.LittleEndian.Uint16(data[i:]) > binary.LittleEndian.Uint16(data[i+2:]) || binary.LittleEndian.Uint32(data[i+4:]) != ^uint32(0) {
					return false
				}
			}
		case FormatDXT3:
			if len(data) == 0 || len(data)%16 != 0 {
				return false
			}
			for i := 0; i < len(data); i += 16 {
				if binary.LittleEndian.Uint64(data[i:]) != 0 {
					return false
				}
			}
		case FormatDXT5:
			if len(data) == 0 || len(data)%16 != 0 {
				return false
			}
			for i := 0; i < len(data); i += 16 {
				a, b := int(data[i]), int(data[i+1])
				al := [8]int{a, b}
				if a > b {
					for j := 2; j < 8; j++ {
						al[j] = ((8-j)*a + (j-1)*b + 6) / 7
					}
				} else {
					for j := 2; j < 6; j++ {
						al[j] = ((6-j)*a + (j-1)*b + 4) / 5
					}
					al[7] = 255
				}
				bits := binary.LittleEndian.Uint64(data[i:]) >> 16
				for range 16 {
					if al[bits&7] > limit {
						return false
					}
					bits >>= 3
				}
			}
		default:
			return false
		}
	}
	return true
}
