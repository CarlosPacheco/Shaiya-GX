package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestDeathChoiceIgnoresLegacyGrowth(t *testing.T) {
	for value := uint8(0); value < 4; value++ {
		a := inputApp()
		a.deathUI, _ = ui.NewDeathDialog(inputTextures{})
		a.state.SelfID = 42
		a.state.Slots[0] = proto.CharacterSlot{ID: 42, Mode: value}
		a.state.Party.Leader = 43
		a.initDeath()
		a.configureDeath()
		for choice, want := range map[int]proto.ReviveMode{1: proto.ReviveBind, 2: proto.ReviveParty} {
			if got, ok := deathChoice(&a.wv.death, choice); !ok || got != want {
				t.Errorf("legacy value %d choice %d = %d, %t", value, choice, got, ok)
			}
		}
	}
}
