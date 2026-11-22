package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestPartyOriginKeepsTopWhenMembersChange(t *testing.T) {
	p := &PartyPanel{}
	const w, h = 1024, 768
	dx, dy := PartyOrigin(w, h, 3)
	if x, y := p.Origin(w, h, 3); x != dx || y != dy {
		t.Fatal("default party position changed", x, y)
	}
	p.Position.Move(400, 300, w, h, PartyWidth, PartyRowHeight)
	x2, y2 := p.Origin(w, h, 2)
	x6, y6 := p.Origin(w, h, 6)
	if x2 != 400 || y2 != 300 || x6 != 400 || y6 != 300 {
		t.Fatalf("moved party list shifted with its size: %d,%d %d,%d", x2, y2, x6, y6)
	}
	p.Position.Move(400, 740, w, h, PartyWidth, PartyRowHeight)
	if _, y := p.Origin(w, h, 6); y != h-6*PartyRowHeight {
		t.Fatalf("party list left the screen: %d", y)
	}
}

func TestMessageHeaderDragsOnlyWhenMovable(t *testing.T) {
	const w, h = 1024, 768
	m, err := NewMessage(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	m.Open("move me", true)
	x, y := m.Origin(w, h)
	m.Update(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 20, MouseY: y + 10}, w, h)
	m.Update(&platform.Input{MouseDown: true, MouseX: x + 220, MouseY: y + 110}, w, h)
	m.Update(&platform.Input{Clicked: true, MouseX: x + 220, MouseY: y + 110}, w, h)
	if nx, ny := m.Origin(w, h); nx != x || ny != y {
		t.Fatal("non-movable message moved")
	}
	m.Movable = true
	if r := m.Update(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 20, MouseY: y + 10}, w, h); r != MessageNone {
		t.Fatal("header press answered the dialog")
	}
	m.Update(&platform.Input{MouseDown: true, MouseX: x + 220, MouseY: y + 110}, w, h)
	m.Update(&platform.Input{Clicked: true, MouseX: x + 220, MouseY: y + 110}, w, h)
	if nx, ny := m.Origin(w, h); nx != x+200 || ny != y+100 || !m.Visible {
		t.Fatalf("header drag did not move the dialog: %d,%d visible=%v", nx, ny, m.Visible)
	}
	m.Open("again", true)
	if nx, ny := m.Origin(w, h); nx != x+200 || ny != y+100 {
		t.Fatal("moved position lost when the dialog reopened")
	}
}

func TestDeathDialogHeaderDrags(t *testing.T) {
	const w, h = 1024, 768
	d, err := NewDeathDialog(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	d.Set(DeathSingle, "dead")
	x, y := d.Origin(w, h)
	d.Update(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 20, MouseY: y + 10}, w, h)
	d.Update(&platform.Input{MouseDown: true, MouseX: x + 220, MouseY: y + 110}, w, h)
	d.Update(&platform.Input{Clicked: true, MouseX: x + 220, MouseY: y + 110}, w, h)
	if nx, ny := d.Origin(w, h); nx != x+200 || ny != y+100 {
		t.Fatalf("header drag did not move the death dialog: %d,%d", nx, ny)
	}
	d.Reset()
	if nx, ny := d.Origin(w, h); nx != x+200 || ny != y+100 {
		t.Fatal("position lost when the dialog was reset")
	}
}
