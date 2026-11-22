package world

import (
	"strings"
	"testing"
)

func TestDungeonMinimapScales(t *testing.T) {
	source := "; calibration, pixels per world unit\n[Map 15]\nScaleX=1.664\nScaleZ=1.472\n[Map 7]\nScaleX=1.317\nScaleZ=1.515\n"
	rows, err := ParseDungeonMinimap([]byte(source))
	if err != nil || len(rows) != 2 || rows[15] != (DungeonMinimap{1.664, 1.472}) || rows[7] != (DungeonMinimap{1.317, 1.515}) {
		t.Fatal(rows, err)
	}
	for _, bad := range []string{
		"",
		strings.Replace(source, "ScaleX=1.664", "ScaleX=0", 1),
		strings.Replace(source, "[Map 7]", "[Map 15]", 1),
		strings.Replace(source, "ScaleZ=1.472", "ScaleZ=oops", 1),
		strings.Replace(source, "[Map 7]", "[Map -1]", 1),
		strings.Replace(source, "[Map 7]", "[Stadium]", 1),
		source + "[Map 8]\nScaleX=1\n",
	} {
		if _, err := ParseDungeonMinimap([]byte(bad)); err == nil {
			t.Fatal("invalid calibration accepted", bad)
		}
	}
}
