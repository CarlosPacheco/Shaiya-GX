package ui

import (
	"encoding/json"
	"testing"
)

func TestOtherSetupDefaultsAndPersistence(t *testing.T) {
	d := DefaultClientSettings()
	if !d.WarningMsg || !d.ShakeScreen || !d.WordBalloon ||
		d.RejectBattle || d.RejectParty || d.RejectTrade || d.HideDamage {
		t.Fatalf("original defaults changed: %+v", d)
	}
	// A saved "off" must survive a reload that starts from the defaults.
	off := d
	off.WarningMsg, off.ShakeScreen, off.WordBalloon = false, false, false
	off.RejectParty, off.HideDamage = true, true
	raw, err := json.Marshal(off)
	if err != nil {
		t.Fatal(err)
	}
	got := DefaultClientSettings()
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.WarningMsg || got.ShakeScreen || got.WordBalloon || !got.RejectParty || !got.HideDamage {
		t.Fatalf("saved values were lost: %+v", got)
	}
	// An old file without the new keys keeps the defaults.
	old := DefaultClientSettings()
	if err := json.Unmarshal([]byte(`{"Music":0.5}`), &old); err != nil || !old.WarningMsg || !old.ShakeScreen {
		t.Fatalf("old file: %v %+v", err, old)
	}
}
