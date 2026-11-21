package ui

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// GMModes is the potion-bar style panel of the three GM switches (Char, Attack,
// Turbo): the original QuickPotion frame widened to three 32px cells at
// (11 + 40*i, 13), movable like the other bars.
type GMModes struct{ Position WindowPosition }

const GMModeCount = 3
const GMModesWidth, GMModesHeight = 94 + 40, 54

// GMModeIcons are the skill icons the modes borrow: Stealth, Untouchable and
// Sprinter.
var GMModeIcons = [GMModeCount]uint16{63, 90, 114}

// Origin sits above the potion bar until the player moves it.
func (g *GMModes) Origin(w, h int) (int, int) {
	px, py := QuickPotionOrigin(WindowPosition{}, w, h)
	return g.Position.Origin(w, h, GMModesWidth, GMModesHeight, px+QuickPotionWidth-GMModesWidth, py-60)
}

func (g *GMModes) Cell(w, h, index int) (int, int) {
	x, y := g.Origin(w, h)
	return x + 11 + index*40, y + 13
}

// Hit returns the cell under the point, or -1.
func (g *GMModes) Hit(w, h, mx, my int) int {
	for i := 0; i < GMModeCount; i++ {
		x, y := g.Cell(w, h, i)
		if mx >= x && mx < x+32 && my >= y && my < y+32 {
			return i
		}
	}
	return -1
}

// DrawGMModesFrame widens the two-cell frame: left cap and first cell, two
// copies of the 40px cell-and-divider period, then the right cap.
func DrawGMModesFrame(s *render.Sprites, frame *render.Texture, x, y int) {
	if frame == nil {
		return
	}
	fw, fh := float32(frame.W), float32(frame.H)
	part := func(sx0, sx1 float32, dx int) {
		s.Draw(frame, float32(x+dx), float32(y), sx1-sx0, 54, sx0/fw, 0, sx1/fw, 54/fh, white)
	}
	part(0, 47, 0)
	part(47, 87, 47)
	part(47, 87, 87)
	part(87, 94, 127)
}
