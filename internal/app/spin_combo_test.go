package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"testing"
)

func TestSpinAbbreviationOnlyAfterContactAndOnlyForEligibleFollowup(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Self.Speed = &proto.CharacterSpeed{Attack: 9}
	a.clock = 10
	a.markCombatAction()
	a.skills.lastSkillID, a.skills.lastSkillAt = 38, 10
	spin := sdata.Skill{Target: 6, Effect: 1}
	area := sdata.Skill{Target: 6, Effect: 3}
	a.clock = 10 + skilltiming.SpinContact.Seconds() - .001
	if a.skillActionReady(156, spin) {
		t.Fatal("cancelled before spin contact")
	}
	a.clock = 10.15
	if !a.skillActionReady(156, spin) || !a.skillActionReady(40, area) {
		t.Fatal("eligible area waited global interval")
	}
	if a.skillActionReady(38, spin) || a.skillActionReady(70, sdata.Skill{Target: 3, Effect: 1}) || a.skillActionReady(30, sdata.Skill{Target: 6, Effect: 4}) || a.skillActionReady(40, sdata.Skill{Target: 6, Effect: 1, ReadyTime: 4}) {
		t.Fatal("unrelated action bypassed cadence")
	}
	a.markCombatAction()
	if a.skillActionReady(156, spin) {
		t.Fatal("basic retained old combo permission")
	}
}
