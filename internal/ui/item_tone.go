package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/element"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// ElementUV selects the monster target atlas symbol for the elemental family.
func ElementUV(value uint8) (UV, bool) {
	family := element.Family(value)
	if family == 0 {
		return UV{}, false
	}
	u := float32(family-1) / 4
	return UV{u, 0, u + .25, 1}, true
}

// ItemTone colors equipment icons by elemental family.
type ItemTone uint8

func (e ItemTone) Draw(s *render.Sprites, tex *render.Texture, uv UV, x, y, size int, tint [4]float32) {
	if tex == nil {
		return
	}
	fx, fy, n := float32(x), float32(y), float32(size)
	family := element.Family(uint8(e))
	if family != 0 {
		overlay := RGBColor(coreElementRGB[family-1])
		overlay[3] = 0x50 / 255.0
		s.DrawColorOverlay(tex, fx, fy, n, n, uv.U0, uv.V0, uv.U1, uv.V1, tint, overlay)
		return
	}
	s.Draw(tex, fx, fy, n, n, uv.U0, uv.V0, uv.U1, uv.V1, tint)
}

// Shaiya Core item_icon.cpp at 7964273, in Fire/Water/Earth/Wind order.
var coreElementRGB = [...]uint32{0xFF0000, 0x00FFFF, 0x00FF00, 0xFFFFFF}
