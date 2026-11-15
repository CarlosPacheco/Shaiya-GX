package ui

import "testing"

func TestChatEntryTextIsCentredInItsRow(t *testing.T) {
	// The 22px entry row with the 15px English chat font leaves 3px above
	// and 4px below; the former fixed +6 left 1px under the glyph cell.
	for _, c := range []struct{ font, want int }{{15, 3}, {13, 4}, {22, 0}, {30, 0}} {
		if got := chatEntryTextY(100, c.font) - 100; got != c.want {
			t.Errorf("font %d: offset %d, want %d", c.font, got, c.want)
		}
	}
}
