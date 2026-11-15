package attackspeed

import "testing"

// Values read directly from ps_game.exe 0x5424e0 and original status dispatch.
func TestOriginalDelaysAndStatusMessages(t *testing.T) {
	for i, want := range []uint32{4000, 3000, 2750, 2500, 2250, 2000, 1750, 1500, 1250, 1000} {
		got, ok := Milliseconds(uint8(i))
		if !ok || got != want {
			t.Fatalf("index %d: %d %v", i, got, ok)
		}
		if Message(uint8(i)) != 5271+i {
			t.Fatal("native status message", i)
		}
	}
	if ms, ok := Milliseconds(Blocked); ok || ms != 0 || Message(Blocked) != 5281 {
		t.Fatal("blocked attack")
	}
	for _, tc := range []struct {
		base  uint8
		bonus int32
		want  uint8
	}{{5, 0, 5}, {5, 1, 6}, {5, -1, 4}, {5, 100, 9}, {5, -100, 0}, {10, -1, 9}, {255, -250, 5}, {0, 0, 0}} {
		if got := Tier(tc.base, tc.bonus); got != tc.want {
			t.Fatalf("base=%d bonus=%d got=%d want=%d", tc.base, tc.bonus, got, tc.want)
		}
	}
}
