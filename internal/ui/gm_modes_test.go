package ui

import "testing"

func TestGMModesPanelHasThreeCellsAndMoves(t *testing.T) {
	var g GMModes
	const w, h = 1024, 768
	x, y := g.Origin(w, h)
	if px, py := QuickPotionOrigin(WindowPosition{}, w, h); y != py-60 || x+GMModesWidth != px+QuickPotionWidth {
		t.Fatal("default sits above the potion bar", x, y)
	}
	for i := 0; i < GMModeCount; i++ {
		cx, cy := g.Cell(w, h, i)
		if cx != x+11+i*40 || cy != y+13 || g.Hit(w, h, cx+16, cy+16) != i {
			t.Fatal("cell", i)
		}
	}
	if g.Hit(w, h, x+44, y+25) != -1 || g.Hit(w, h, x+2, y+25) != -1 {
		t.Fatal("divider and cap are not cells")
	}
	g.Position.Move(100, 200, w, h, GMModesWidth, GMModesHeight)
	if nx, ny := g.Origin(w, h); nx != 100 || ny != 200 {
		t.Fatal("moved panel", nx, ny)
	}
}
