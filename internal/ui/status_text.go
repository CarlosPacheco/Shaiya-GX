package ui

import "github.com/Spelunkern/Shaiya-GX/internal/render"

var statusLabelTint = [4]float32{.933, .894, .808, 1}

// Portrait names use warm text with a one-pixel black shadow.
func drawStatusText(s *render.Sprites, f *Font, text string, x, y, scale float32, maxW int) {
	f.DrawScaled(s, text, x+1, y+1, scale, [4]float32{0, 0, 0, 1}, maxW)
	f.DrawScaled(s, text, x, y, scale, statusLabelTint, maxW)
}

// Resource values and the clock use the same white face and one-pixel shadow.
func drawResourceText(s *render.Sprites, f *Font, text string, x, y, scale float32, maxW int) {
	f.DrawScaled(s, text, x+1, y+1, scale, [4]float32{0, 0, 0, 1}, maxW)
	f.DrawScaled(s, text, x, y, scale, white, maxW)
}
