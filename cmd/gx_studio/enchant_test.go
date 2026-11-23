package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
)

func TestEnchantChanceEditorsStayInSync(t *testing.T) {
	test.NewApp()

	updates := 0
	row := enchantInputs(enchantrules.Level{Number: 1, SuccessChance: 1}, func() { updates++ })
	if row.chance.Text != "1" || row.percent.Text != "0.0001" {
		t.Fatalf("one per million displayed as %q and %q", row.chance.Text, row.percent.Text)
	}

	for _, c := range []struct {
		input, converted string
		percent          bool
	}{
		{"12.5%", "125000", true},
		{"100", "1000000", true},
		{"0", "0", true},
		{".0001", "1", true},
		{"0.00001", "", true},
		{"100.0001", "", true},
		{"-1", "", true},
		{"450001", "45.0001", false},
		{"1000001", "", false},
		{"125000", "12.5", false},
	} {
		entry, other := row.chance, row.percent
		if c.percent {
			entry, other = other, entry
		}
		before := updates
		entry.SetText(c.input)
		if entry.Text != c.input || other.Text != c.converted || updates != before+1 {
			t.Errorf("edit %q produced %q and %q with %d refreshes", c.input, entry.Text, other.Text, updates-before)
		}
	}
}
