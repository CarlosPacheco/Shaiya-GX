package ui

// GMModes is the small bar of the three GM switches (Char, Attack, Turbo). It
// follows the buff strips: 32px icons with a 6px gap, freely movable.
type GMModes struct{ Position WindowPosition }

const GMModeCount = 3

// GMModeIcons are the skill icons the modes borrow: Stealth, Untouchable and
// Sprinter.
var GMModeIcons = [GMModeCount]uint16{63, 90, 114}

func (g *GMModes) Rect(w, h int) (x, y, width, height int) {
	width, height = GMModeCount*38-6, 32
	x, y = g.Position.Origin(w, h, width, height, max(0, min(w/2+128, w-width)), 60+2*48)
	return
}

func (g *GMModes) Slot(w, h, index int) (int, int) {
	x, y, _, _ := g.Rect(w, h)
	return x + index*38, y
}

// Hit returns the icon under the point, or -1.
func (g *GMModes) Hit(w, h, mx, my int) int {
	for i := 0; i < GMModeCount; i++ {
		x, y := g.Slot(w, h, i)
		if mx >= x && mx < x+32 && my >= y && my < y+32 {
			return i
		}
	}
	return -1
}
