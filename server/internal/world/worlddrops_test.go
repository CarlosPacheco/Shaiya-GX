package world

import (
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
)

func TestWorldDropRollsUseInclusiveRangesAndIndependentExactChances(t *testing.T) {
	dropCatalog.mu.Lock()
	old := dropCatalog.worldRules
	dropCatalog.worldRules = []worlddrops.Rule{
		{Index: 1, MinLevel: 20, MaxLevel: 69, DropChance: 1},
		{Index: 2, MinLevel: 69, MaxLevel: 69, DropChance: 10000},
		{Index: 3, MinLevel: 20, MaxLevel: 69, DropChance: 0},
		{Index: 4, MinLevel: 1, MaxLevel: 68, DropChance: worlddrops.RollRange},
	}
	dropCatalog.mu.Unlock()
	t.Cleanup(func() {
		dropCatalog.mu.Lock()
		dropCatalog.worldRules = old
		dropCatalog.mu.Unlock()
	})
	for _, tt := range []struct {
		rolls []uint64
		drops []int64
	}{
		{[]uint64{0, 9999}, []int64{1, 2}},
		{[]uint64{1, 10000}, nil},
	} {
		var got []int64
		n := 0
		rollWorldDrops(69, func(limit uint64) uint64 {
			if limit != 10_000_000_000 || n >= len(tt.rolls) {
				t.Fatal("unexpected random draw")
			}
			roll := tt.rolls[n]
			n++
			return roll
		}, func(rule worlddrops.Rule) { got = append(got, rule.Index) })
		if n != len(tt.rolls) || !slices.Equal(got, tt.drops) {
			t.Fatalf("draws=%d drops=%v, want %v", n, got, tt.drops)
		}
	}
}
