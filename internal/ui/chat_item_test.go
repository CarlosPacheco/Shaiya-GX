package ui

import (
	"strings"
	"testing"
)

func TestAcquiredItemIconWrapsAtTextHeight(t *testing.T) {
	u, err := testUnicodeFace(t)
	if err != nil {
		t.Fatal(err)
	}
	f := &Font{unicode: u, Height: 15}
	text := strings.Repeat("Obtained ", 12) + "\uFFFC Sword"
	rows, lines := wrapChatMarkup(f, text, white, true, true)
	icons := 0
	for _, row := range rows {
		for _, run := range row {
			width := f.Width(run.text)
			if run.icon {
				icons++
				width = f.Height
			}
			if run.x+width > 316 {
				t.Fatal("run outside chat", run)
			}
		}
	}
	if icons != 1 || !strings.Contains(strings.Join(lines, " "), "Sword") {
		t.Fatal("icon or item name lost", icons, lines)
	}
	for _, trusted := range []bool{false, true} {
		rows, _ := wrapChatMarkup(f, "\uFFFC Sword", white, trusted)
		for _, row := range rows {
			for _, run := range row {
				if run.icon {
					t.Fatal("text without structured item metadata produced icon")
				}
			}
		}
	}
}
