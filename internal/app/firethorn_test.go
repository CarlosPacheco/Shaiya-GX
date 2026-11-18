package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Fire Thorn is a buff that hurts: its periodic damage reads as damage, and its
// hits show the damage effect of its element (fire is sequence 2).
func TestFireThornPeriodicDamageIsDamageNotHealing(t *testing.T) {
	if !effect.PeriodicHit(47) || !effect.PeriodicHit(48) || effect.PeriodicHit(46) {
		t.Fatal("only Fire Thorn and Fire Bush hurt periodically")
	}
	if effect.AttributeDamageSequence(1) != 2 || effect.AttributeDamageSequence(0) != 4 {
		t.Fatal("the fire damage sequence is 2")
	}
	a := inputApp()
	a.msg = text.SysMsg{1310: "damage taken <v>", 1103: "healed <v>"}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 47, Level: 1}: {Name: "Fire Thorn", Effect: 4, Detail: 84}}
	a.state.SelfID = 7
	a.state.LastPeriodicEffect = proto.PeriodicEffect{Target: 7, Skill: 47, Level: 1, HP: 8}
	a.periodicSystemChat()
	if n := len(a.state.Chat); n == 0 || a.state.Chat[n-1].Text == "healed 8" {
		t.Fatalf("Fire Thorn damage read as healing: %+v", a.state.Chat)
	}
}
