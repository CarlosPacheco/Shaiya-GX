package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
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
		{"", "", true},
		{"100%", "1000000", true},
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

func TestEnchantSaveRequiresValidChangedRules(t *testing.T) {
	app := test.NewApp()
	window := app.NewWindow("Enchant")
	t.Cleanup(window.Close)
	e := editor{
		window: window,
		enchant: &studio.EnchantCatalog{
			File: &studio.ServerFile{Path: "ItemEnchant.ini"},
			Rules: enchantrules.Catalog{
				Attack:  []enchantrules.Level{{Number: 1, SuccessChance: 125000, Bonus: 2147483647}},
				Defense: []enchantrules.Level{{Number: 1, SuccessChance: 1}},
			},
		},
	}
	e.enchantCatalog()

	var save *widget.Button
	var entries []*widget.Entry
	var visit func(fyne.CanvasObject)
	visit = func(object fyne.CanvasObject) {
		switch v := object.(type) {
		case *fyne.Container:
			for _, child := range v.Objects {
				visit(child)
			}
		case *container.AppTabs:
			visit(v.Items[0].Content)
		case *container.Scroll:
			visit(v.Content)
		case *widget.Button:
			if v.Text == "Save server file" {
				save = v
			}
		case *widget.Entry:
			entries = append(entries, v)
		}
	}
	visit(window.Content())
	if save == nil || len(entries) != 3 {
		t.Fatalf("expected Save and three Attack entries, found %v and %d entries", save != nil, len(entries))
	}
	raw, percent, bonus := entries[0], entries[1], entries[2]
	assertState := func(label string, dirty, enabled bool) {
		t.Helper()
		actualDirty, actualEnabled := e.serverDraftDirty(), !save.Disabled()
		if actualDirty != dirty || actualEnabled != enabled {
			t.Errorf("%s: dirty = %t, Save enabled = %t; want %t, %t", label, actualDirty, actualEnabled, dirty, enabled)
		}
	}
	assertState("open", false, false)
	percent.SetText("012.5000%")
	assertState("same value", false, false)
	percent.SetText("101")
	assertState("invalid percentage", true, false)
	percent.SetText("")
	assertState("cleared percentage", true, false)
	percent.SetText("0.0001")
	if raw.Text != "1" {
		t.Fatalf("corrected percentage set raw to %q, want 1", raw.Text)
	}
	assertState("corrected value", true, true)
	bonus.SetText("2147483648")
	assertState("invalid bonus", true, false)
	bonus.SetText("2147483647")
	assertState("valid bonus", true, true)
	raw.SetText("125000")
	assertState("reverted", false, false)
}
