package scene

import (
	"cmp"
	"math"
	"slices"
	"testing"
)

func TestVANIDepthKeyPreservesStableOrder(t *testing.T) {
	depths := []float32{
		1, float32(math.NaN()), -1, 0, 3, float32(math.Inf(1)),
		math.Float32frombits(1 << 31), 1, float32(math.Inf(-1)),
		math.Float32frombits(0xffc00001), 3, -1,
	}
	keys := make([]uint64, len(depths))
	want := make([]uint32, len(depths))
	for i, depth := range depths {
		want[i] = uint32(i)
		keys[i] = vaniDepthKey(depth, uint32(i))
	}
	slices.SortStableFunc(want, func(a, b uint32) int {
		return cmp.Compare(depths[b], depths[a])
	})
	slices.Sort(keys)
	for i, key := range keys {
		if got := uint32(key); got != want[i] {
			t.Fatalf("position %d: got placement %d, want %d", i, got, want[i])
		}
	}
}
