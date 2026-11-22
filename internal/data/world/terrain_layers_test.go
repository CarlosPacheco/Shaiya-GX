package world

import (
	"slices"
	"testing"
)

func TestTerrainLayerBlocksMatchNativeSampleSelection(t *testing.T) {
	const size = 96
	const stride = size/2 + 1
	w := &WLD{MapSize: size, TextureMap: make([]byte, stride*stride)}
	set := func(x, z int, layer byte) { w.TextureMap[z*stride+x] = layer }
	set(16, 16, 1) // Local to block (1,1) and within the northwest margin.
	set(34, 26, 2) // Across the east edge of (1,1), outside its north/south margins.
	set(7, 2, 3)   // Odd X samples are skipped.
	set(2, 7, 4)   // Odd Z samples are skipped.
	set(48, 6, 5)  // Final row and column are excluded.
	set(6, 48, 6)
	set(2, 2, 255) // Unassigned sample.
	set(32, 2, 7)  // The highest valid layer remains selectable.
	side, blocks := w.TerrainLayerBlocks()
	want := []byte{3, 131, 129, 3, 7, 5, 1, 1, 1}
	if side != 3 || !slices.Equal(blocks, want) {
		t.Fatalf("got %dx%d %v, want 3x3 %v", side, side, blocks, want)
	}
	w.TextureMap = w.TextureMap[:len(w.TextureMap)-1]
	if side, grid := w.TerrainLayerBlocks(); side != 0 || grid != nil {
		t.Fatalf("incomplete field grid produced %dx%d blocks", side, side)
	}
}
