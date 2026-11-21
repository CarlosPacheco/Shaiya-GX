package ui

import (
	"encoding/json"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestUserOptionsUseBannedWordsInsteadOfRetiredChatPreference(t *testing.T) {
	m, err := NewGameMenu(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"AutoHideChat":true,"ChatRows":[7,8]}`), &m.Settings); err != nil {
		t.Fatal(err)
	}
	if m.Settings.BannedWords {
		t.Fatal("the retired preference activated the word filter")
	}
	want := m.Settings
	want.BannedWords = true
	m.Open(MenuUser)
	x, y := m.Origin(1024, 768)
	m.Update(&platform.Input{MouseX: x + 140, MouseY: y + 232, Pressed: true, MouseDown: true}, 1024, 768)
	if m.Update(&platform.Input{MouseX: x + 140, MouseY: y + 232, Clicked: true}, 1024, 768) != MenuChanged || m.Settings != want {
		t.Fatal("native checkbox did not toggle the word filter")
	}
	raw, err := json.Marshal(m.Settings)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["AutoHideChat"]; exists || string(fields["BannedWords"]) != "true" || string(fields["ChatRows"]) != "[7,8]" {
		t.Fatal("saving options did not preserve the remaining chat settings")
	}
}
