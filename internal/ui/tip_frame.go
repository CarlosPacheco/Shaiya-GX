package ui

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// Colours of the item tooltip frame. Every plain tip-style panel (item and
// skill tooltips, the player menu, the party list) shares them so their
// borders match.
var (
	TipFill         = [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 205.0 / 255}
	TipBorder       = [4]float32{143.0 / 255, 136.0 / 255, 115.0 / 255, 1}
	TipBorderActive = [4]float32{1, 220.0 / 255, 100.0 / 255, 1}
)

// DrawTipFrame draws the item tooltip panel: a translucent dark fill with a
// one-pixel border inset by one pixel. scale shrinks tooltips on small views.
func DrawTipFrame(s *render.Sprites, solid *render.Texture, x, y, w, h, scale float32, fill, border [4]float32) {
	rect := func(dx, dy, rw, rh float32, c [4]float32) {
		s.Draw(solid, x+dx*scale, y+dy*scale, rw*scale, rh*scale, 0, 0, 1, 1, c)
	}
	rect(0, 0, w, h, fill)
	rect(1, 1, w-2, 1, border)
	rect(1, h-2, w-2, 1, border)
	rect(1, 1, 1, h-2, border)
	rect(w-2, 1, 1, h-2, border)
}
