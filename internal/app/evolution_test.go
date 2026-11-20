package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestEvolutionUsesLocalModelAndServerSkillList(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.PlayerForms = map[uint32]proto.PlayerFormAppearance{
		1: {User: 1, Kind: proto.FormEvolution, Skill: 127, Rank: 2},
	}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 127, Level: 2}: {ChangeLevel: 500}}
	a.npcSkillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 36, Level: 1}: {AttackRange: 12}}
	a.state.Evolution = proto.EvolutionBar{Generation: 10, Actions: [2]proto.EvolutionAction{{Skill: 36, Range: 10}}}
	a.confirmEvolutionBar()
	if a.formMonster(1) != 500 {
		t.Fatal("the client must take the Evolution model from its local player catalog")
	}
	p := proto.EvolutionCastPacket(10, 0, 5)
	if def, ok := a.evolutionCastDefinition(p); !ok || def.AttackRange != 10 {
		t.Fatal("bar must use the server's slot and reach", def, ok)
	}
	if a.learnedSkillPacket(proto.LearnedSkill{ID: 127, Level: 2}) != nil {
		t.Fatal("class skills remained castable during Evolution")
	}
	a.state.Evolution.Generation++
	if _, ok := a.evolutionCastDefinition(p); ok {
		t.Fatal("a previous form's click still works")
	}
}

func TestCopiedFormKeepsTheSelectedTemplate(t *testing.T) {
	a := inputApp()
	a.state.SelfID, a.state.Target = 1, 20
	a.state.Entities[20] = &game.Entity{ID: 20, Kind: game.KindMob}
	d := sdata.Skill{Effect: 4, Detail: 90, ChangeType: 102, Target: 2}
	if target := a.resolveSkillTarget(d); target != 20 || !a.validSkillTarget(d, target) || a.validSkillTarget(d, 1) {
		t.Fatal("Transformation treated a monster template as an ordinary self buff")
	}
}
