package app

import "testing"

// Game.exe 0x4c9140: weapons, armor, accessories and the rest each map to one
// ui_armorNNN cue; type 0 and unknown types are mute.
func TestItemSoundNames(t *testing.T) {
	for _, tc := range []struct {
		itemType uint8
		want     string
	}{
		{1, "ui_armor014.ogg"}, {45, "ui_armor014.ogg"}, {16, "ui_armor002.ogg"}, {31, "ui_armor002.ogg"},
		{26, "ui_armor024.ogg"}, {97, "ui_armor021.ogg"}, {0, ""}, {98, ""}, {255, ""},
	} {
		if got := itemSoundName(tc.itemType); got != tc.want {
			t.Fatalf("type %d: got %q want %q", tc.itemType, got, tc.want)
		}
	}
}

func TestGroundDropSound(t *testing.T) {
	for _, tc := range []struct {
		itemType uint8
		want     string
	}{{26, "ui_moneydrop1.ogg"}, {22, "ui_itemdrop2.ogg"}, {100, "ui_itemdrop2.ogg"}, {1, "ui_itemdrop1.ogg"}, {31, "ui_itemdrop1.ogg"}, {98, "ui_itemdrop1.ogg"}} {
		if got := groundDropSound(tc.itemType); got != tc.want {
			t.Fatalf("type %d: %q want %q", tc.itemType, got, tc.want)
		}
	}
}
