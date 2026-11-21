package app

import (
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestRememberCharacterSlotAcrossLogins(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("AppData", dir)
	cfg := Config{LoginAddr: "localhost:30800", WorldPort: 30810}
	path := characterSlotPath(cfg, 1, 42)
	a := &App{state: game.New(), characterSlotFile: path}
	a.state.Slots[3] = proto.CharacterSlot{ID: 90}
	a.state.SelfID, a.state.Phase = 90, game.PhaseLoading
	enter := proto.NewWriter(proto.CharacterEnteredMap)
	enter.U32(90)
	enter.U8(1)
	enter.U16(0)
	enter.F32(10)
	enter.F32(20)
	enter.F32(30)
	enter.U32(0)
	enter.U32(0)
	a.applyWorldPacket(enter.Bytes())
	if data, err := os.ReadFile(path); err != nil || string(data) != "3" {
		t.Fatalf("stored slot = %q, err = %v", data, err)
	}

	for _, tc := range []struct {
		name string
		path string
		slot proto.CharacterSlot
		want int
	}{
		{"replacement occupies remembered slot", path, proto.CharacterSlot{ID: 100}, 3},
		{"remembered slot is empty", path, proto.CharacterSlot{}, 0},
		{"remembered slot is pending deletion", path, proto.CharacterSlot{ID: 100, PendingDelete: true}, 0},
		{"another account", characterSlotPath(cfg, 1, 43), proto.CharacterSlot{ID: 100}, 0},
		{"another server", characterSlotPath(cfg, 2, 42), proto.CharacterSlot{ID: 100}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &App{
				state: game.New(), chars: &ui.CharSelect{Selected: 1},
				characterSlotFile: tc.path, restoreCharacterSlot: true,
			}
			a.state.Slots[0] = proto.CharacterSlot{ID: 1}
			a.state.Slots[1] = proto.CharacterSlot{ID: 2}
			if tc.slot.ID != 0 {
				a.state.Slots[3] = tc.slot
			}
			a.refreshCharacterRoster()
			if a.chars.Selected != tc.want {
				t.Fatalf("selected slot = %d, want %d", a.chars.Selected, tc.want)
			}
			a.chars.Selected = 1
			a.refreshCharacterRoster()
			if a.chars.Selected != 1 {
				t.Fatal("refresh overwrote the manual selection")
			}
		})
	}
}
