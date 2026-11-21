package ui

import "testing"

func TestBuffLayoutNativeRowsAndSmallWindows(t *testing.T) {
	for _, width := range []int{320, 640, 800, 1024, 1920} {
		for i := range 32 {
			x, y := BuffSlot(width, i)
			if x < 0 || x+32 > width || BuffHit(width, 32, x+1, y+1) != i {
				t.Fatal("buff clipped or wrong instance hit", width, i, x, y)
			}
		}
	}
	if x, y := BuffSlot(1024, 10); x != 640 || y != 104 {
		t.Fatal("native ten-column/44px-row layout", x, y)
	}
	if BuffHit(1024, 11, 641, 99) != -1 {
		t.Fatal("row gap captured world pointer")
	}
}

func TestBuffDuration(t *testing.T) {
	for seconds, want := range map[uint32]string{0: "0s", 59: "59s", 60: "1m", 61: "2m", 3600: "1h", 3601: "2h", 0xffffffff: "1193047h"} {
		if got := BuffDuration(seconds, true); got != want {
			t.Fatal(seconds, got, want)
		}
	}
	if BuffDuration(0, false) != "-" {
		t.Fatal("untimed effect displayed as expired")
	}
}

func TestBuffRemainingTextUsesOriginalClockFormat(t *testing.T) {
	for _, tc := range []struct {
		seconds uint32
		want    string
	}{{0, ""}, {9, "00:00:09"}, {60, "00:01:00"}, {119, "00:01:59"}, {3600, "01:00:00"}, {86400, "01 - 00:00:00"}, {2*86400 + 3661, "02 - 01:01:01"}} {
		if got := BuffRemainingText(tc.seconds); got != tc.want {
			t.Errorf("%d seconds: got %q, want %q", tc.seconds, got, tc.want)
		}
	}
}
