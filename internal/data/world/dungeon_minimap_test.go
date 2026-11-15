package world

import (
	"strings"
	"testing"
)

func TestDungeonMinimapFixedPointScales(t *testing.T) {
	source := "# legacy non-UTF8 comments: \xb0\xa1\n[WorldMap]\ncount=2\n[0]\nMapNum=15\nImgPos_x=1664\nImgPos_y=1472\n[1]\nMapNum=7\nImgPos_x=1317\nImgPos_y=1515\n"
	rows, err := ParseDungeonMinimap([]byte(source))
	if err != nil || len(rows) != 2 || rows[15] != (DungeonMinimap{1.664, 1.472}) || rows[7] != (DungeonMinimap{1.317, 1.515}) {
		t.Fatal(rows, err)
	}
	for _, bad := range []string{
		strings.Replace(source, "count=2", "count=3", 1),
		strings.Replace(source, "count=2", "count=0", 1),
		strings.Replace(source, "ImgPos_x=1664", "ImgPos_x=0", 1),
		strings.Replace(source, "MapNum=7", "MapNum=15", 1),
		strings.Replace(source, "ImgPos_y=1472", "ImgPos_y=oops", 1),
		strings.Replace(source, "MapNum=7", "MapNum=-1", 1),
		source + "[1]\nMapNum=8\n",
	} {
		if _, err := ParseDungeonMinimap([]byte(bad)); err == nil {
			t.Fatal("invalid calibration accepted", bad)
		}
	}
}
