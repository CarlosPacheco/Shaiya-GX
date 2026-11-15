package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestStatusHotkeyFooterAndPointerOwnership(t *testing.T) {
	a := inputApp()
	a.inventory.open = true
	a.worldControls(&platform.Input{Chars: []rune{'T'}}, 1024, 768)
	if !a.status.open || !a.inventory.open {
		t.Fatal("native Status hotkey closed inventory")
	}
	x, y := a.statusUI.Origin(1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 2, MouseY: y + 157, Pressed: true, MouseDown: true, Down: "W"}, 1024, 768)
	controls := a.worldControls(&platform.Input{MouseX: x + 2, MouseY: y + 157, Clicked: true, Down: "W"}, 1024, 768)
	if !a.statusUI.Expanded || controls.Clicked || controls.Down != "W" {
		t.Fatal("status tab leaked a click or swallowed held movement")
	}
	controls = a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.status.open || len(controls.Keys) != 0 {
		t.Fatal("status close reached world logout")
	}
	a.worldControls(&platform.Input{MouseX: 682, MouseY: 715, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: 682, MouseY: 715, Clicked: true}, 1024, 768)
	if !a.status.open {
		t.Fatal("status footer button did not open")
	}
	a.toggleSkills(false)
	if !a.status.open || !a.skills.open {
		t.Fatal("opening skills closed status")
	}
}

func TestStatusUsesOnlyConfirmedNativeSpeedMessages(t *testing.T) {
	a := inputApp()
	a.msg = map[int]string{5271: "none", 5276: "normal", 5278: "fast", 5280: "extremely fast", 5281: "can't attack", 5282: "can't move"}
	if s := a.statusState(); s.AttackSpeed != "-" || s.MoveSpeed != "-" {
		t.Fatal("unconfirmed speed inferred", s)
	}
	a.state.Self.Speed = &proto.CharacterSpeed{Attack: 9, Move: 3}
	if s := a.statusState(); s.AttackSpeed != "extremely fast" || s.MoveSpeed != "fast" {
		t.Fatal("native confirmed labels", s)
	}
	a.state.Self.Speed = &proto.CharacterSpeed{Attack: 255, Move: 255}
	if s := a.statusState(); s.AttackSpeed != "can't attack" || s.MoveSpeed != "can't move" {
		t.Fatal("native blocked labels", s)
	}
}

func TestStatusTitleFollowsConfirmedKillUpdates(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 100
	a.msg = map[int]string{5450: "Blue", 5451: "Yellow", 5389: "localized first", 5392: "localized fourth", 5404: "localized final"}
	for _, c := range []struct {
		op    proto.Opcode
		kills uint32
		want  string
	}{{proto.UserKillCountUpdate, 1000, "Blue localized fourth"}, {proto.CharacterPvPKills, 200000, "Yellow localized first"}, {proto.UserKillCountUpdate, 1000000, "localized final"}, {proto.UserKillCountUpdate, 0, ""}} {
		w := proto.NewWriter(c.op)
		if c.op == proto.UserKillCountUpdate {
			w.U8(0)
		} else {
			w.U32(100)
		}
		w.U32(c.kills)
		if _, err := a.state.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		if got := a.statusState().Title; got != c.want {
			t.Fatalf("kills %d title %q, want %q", c.kills, got, c.want)
		}
	}
}
