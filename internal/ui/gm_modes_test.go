package ui

import "testing"

func TestGMModesBarHitsThreeIconsAndMoves(t *testing.T) {
	var g GMModes
	const w, h = 1024, 768
	x, y := g.Slot(w, h, 0)
	for i := 0; i < GMModeCount; i++ {
		sx, sy := g.Slot(w, h, i)
		if sx != x+i*38 || sy != y || g.Hit(w, h, sx+16, sy+16) != i {
			t.Fatal("icon", i)
		}
	}
	if g.Hit(w, h, x+34, y+16) != -1 || g.Hit(w, h, x-1, y) != -1 {
		t.Fatal("the gap between icons is not an icon")
	}
	g.Position.Move(100, 200, w, h, GMModeCount*38-6, 32)
	if nx, ny := g.Slot(w, h, 0); nx != 100 || ny != 200 {
		t.Fatal("moved bar", nx, ny)
	}
}
