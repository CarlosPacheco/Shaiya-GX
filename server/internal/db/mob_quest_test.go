package db

import "testing"

func TestMobQuestItemNativeIndex(t *testing.T) {
	for _, tt := range []struct {
		index int32
		want  ItemKey
	}{
		{0, ItemKey{}}, {-1, ItemKey{}}, {1, ItemKey{27, 1}}, {248, ItemKey{27, 248}},
		{1000, ItemKey{27, 232}}, {1001, ItemKey{28, 1}}, {1139, ItemKey{28, 139}},
		{2000, ItemKey{28, 232}}, {2001, ItemKey{29, 1}},
	} {
		if got := MobQuestItem(tt.index); got != tt.want {
			t.Fatalf("%d: %+v, want %+v", tt.index, got, tt.want)
		}
	}
}
