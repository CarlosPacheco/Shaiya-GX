package mapgeometry

const (
	// The EP5 terrain encoding spans -200 to 1110.7 units. The wider envelope
	// also leaves room for model geometry, water and moving platforms.
	minPlayerHeight     float32 = -256
	maxPlayerHeight     float32 = 2048
	dungeonHeightMargin float32 = 32
)

// PlayerHeightLimits bounds prediction and authoritative locomotion. Static
// dungeons use their collision envelope with clearance for jumps and portal
// volumes. Terrain and moving geometry use the wider world limits.
func (g *Geometry) PlayerHeightLimits() (low, high float32) {
	if g != nil && !g.HasTerrain() && len(g.Dynamic) == 0 {
		if bottom, top, ok := g.Index.VerticalBounds(); ok {
			low := max(minPlayerHeight, float32(bottom)-dungeonHeightMargin)
			high := min(maxPlayerHeight, float32(top)+dungeonHeightMargin)
			if low < high {
				return low, high
			}
		}
	}
	return minPlayerHeight, maxPlayerHeight
}

func (g *Geometry) ValidPlayerHeight(y float32) bool {
	low, high := g.PlayerHeightLimits()
	return y >= low && y <= high
}
