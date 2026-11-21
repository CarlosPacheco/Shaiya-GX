package ui

import (
	"encoding/json"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestUserOptionsIgnoreRetiredChatPreference(t *testing.T) {
	m, err := NewGameMenu(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"AutoHideChat":true,"ChatRows":[7,8]}`), &m.Settings); err != nil {
		t.Fatal(err)
	}
	want := m.Settings
	m.Open(MenuUser)
	x, y := m.Origin(1024, 768)
	m.Update(&platform.Input{MouseX: x + 140, MouseY: y + 232, Pressed: true, MouseDown: true}, 1024, 768)
	if m.Update(&platform.Input{MouseX: x + 140, MouseY: y + 232, Clicked: true}, 1024, 768) == MenuChanged || m.Settings != want {
		t.Fatal("retired checkbox still changes settings")
	}
	raw, err := json.Marshal(m.Settings)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["AutoHideChat"]; exists || string(fields["ChatRows"]) != "[7,8]" {
		t.Fatal("saving options did not preserve the remaining chat settings")
	}
}
