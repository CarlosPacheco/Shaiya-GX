package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// PrevSkillID names the skill that must be cooling down; any skill can have it.
func TestPreviousSkillMustBeCoolingDown(t *testing.T) {
	s, _ := potentialSession(t)
	def := &db.SkillDef{SkillID: 61, SkillLevel: 1, PrevSkillID: 104}
	if s.previousSkillCooling(def) {
		t.Fatal("usable although Halt Kick was never used")
	}
	s.skillCooldowns[104] = time.Now().Add(3 * time.Second)
	if !s.previousSkillCooling(def) {
		t.Fatal("refused while Halt Kick is cooling down")
	}
	s.skillCooldowns[104] = time.Now().Add(-time.Second)
	if s.previousSkillCooling(def) {
		t.Fatal("usable after the previous skill finished cooling down")
	}
	if !s.previousSkillCooling(&db.SkillDef{SkillID: 5}) {
		t.Fatal("a skill without a previous one must not be restricted")
	}
}
