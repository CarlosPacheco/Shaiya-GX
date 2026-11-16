package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"testing"
)

func TestProtocolPhasesProtectActorIdentity(t *testing.T) {
	s := &Session{}
	if !s.phaseAllows(packet.SelectCharacter) || s.phaseAllows(packet.SkillUse) {
		t.Fatal("selection phase")
	}
	s.char = &db.Character{ID: 1}
	s.awaitingMapLoad = true
	if !s.phaseAllows(packet.CharacterEnteredMap) || s.phaseAllows(packet.DeleteCharacter) || s.phaseAllows(packet.SkillUse) {
		t.Fatal("loading phase")
	}
	s.awaitingMapLoad = false
	if s.phaseAllows(packet.SelectCharacter) || s.phaseAllows(packet.AccountFaction) || s.phaseAllows(packet.CharacterEnteredMap) || !s.phaseAllows(packet.SkillUse) {
		t.Fatal("world phase")
	}
	if !s.phaseAllows(packet.Ping) || !s.phaseAllows(packet.Logout) {
		t.Fatal("lifecycle commands")
	}
}
