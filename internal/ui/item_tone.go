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
	switch element.Family(uint8(e)) {
	case 1: // Light red.
		tint[1] *= .83
		tint[2] *= .83
	case 2: // Sky blue.
		tint[0] *= .76
		tint[1] *= .94
	case 3: // Green.
		tint[0] *= .79
		tint[2] *= .80
	case 4: // Soft gray.
		for i := range 3 {
			tint[i] *= .92
		}
		s.DrawMutedColor(tex, fx, fy, n, n, uv.U0, uv.V0, uv.U1, uv.V1, tint)
		return
	}
	s.Draw(tex, fx, fy, n, n, uv.U0, uv.V0, uv.U1, uv.V1, tint)
}
