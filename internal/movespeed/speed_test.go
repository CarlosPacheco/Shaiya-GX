package movespeed

import "testing"

func TestOriginalOnFootTiersAndClamp(t *testing.T) {
	for i, want := range []float32{3, 4, 5, 6, 7} {
		if got := Units(uint8(i)); got != want {
			t.Fatalf("tier %d: %g != %g", i, got, want)
		}
	}
	for _, tc := range []struct {
		bonus int32
		tier  uint8
	}{{-2147483648, 0}, {-3, 0}, {-2, 0}, {-1, 1}, {0, 2}, {1, 3}, {2, 4}, {3, 4}, {2147483647, 4}} {
		if got := Tier(tc.bonus); got != tc.tier {
			t.Fatalf("bonus %d: %d != %d", tc.bonus, got, tc.tier)
		}
	}
	if Units(Blocked) != 0 || Units(5) != 5 {
		t.Fatal("blocked/unknown native fallback")
	}
}

func TestOriginalMovementStatusMessages(t *testing.T) {
	for i, want := range []int{5273, 5274, 5276, 5278, 5279} {
		if Message(uint8(i)) != want {
			t.Fatal(i, want)
		}
	}
	if Message(Blocked) != 5282 {
		t.Fatal("native movement block text")
	}
}
