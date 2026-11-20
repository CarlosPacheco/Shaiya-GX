package catalogfiles

import (
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/itemboxes"
	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
)

func TestLootFilesRoundTripBothBoxTypes(t *testing.T) {
	rows := []itemboxes.Box{
		{Index: 1, Description: "Grade; weighted", BoxType: itemboxes.Normal, ItemType: itemboxes.Grade,
			Rewards: []itemboxes.Reward{{Item: 55, Chance: 1}, {Item: 73, Chance: 999}}},
		{Index: 2, Description: "Direct // independent", BoxType: itemboxes.Independent, ItemType: itemboxes.Direct,
			Rewards: []itemboxes.Reward{{Item: 100131, Chance: 1}, {Item: 100132, Chance: worlddrops.RollRange}}},
	}
	parsed, err := ParseItemBoxes(EncodeItemBoxes(rows))
	if err != nil || !itemboxes.Equal(parsed, rows) {
		t.Fatalf("boxes changed in INI: %v, %v", parsed, err)
	}
	drops := []worlddrops.Rule{{Index: 1, Description: "A; B // C", Grade: 11, MinLevel: 10, MaxLevel: 20, DropChance: 1}}
	got, err := ParseWorldDrops(EncodeWorldDrops(drops))
	if err != nil || !slices.Equal(drops, got) {
		t.Fatalf("drops changed in INI: %v, %v", got, err)
	}
}

func TestNoticeFileRetainsBothSchedulesAndLiteralMessage(t *testing.T) {
	rows := []Notice{{Index: 1, Message: "An hourly announcement; read // carefully", Frequency: 60},
		{Index: 2, Message: "Fixed", Frequency: 10, Fixed: [5]string{"1/12/10", "", "4/18/05"}}, {Index: 3, Message: "Manual"}}
	got, err := ParseNotices(EncodeNotices(rows))
	if err != nil || !slices.Equal(rows, got) {
		t.Fatalf("notices changed: %v, %v", got, err)
	}
}
