package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func TestCopiedFormsUseTheSelectedMonsterAndEnemy(t *testing.T) {
	base, defs := lapisSession(t)
	mobID, mob := lapisMob(t, base, defs)
	base.char.PosX, base.char.PosZ = 1, 1
	s, _ := gmWireSession(t, *base.char, false)
	s.items = base.items
	s.skillCooldowns = map[uint16]time.Time{}
	s.player = RegisterPlayer(*s.char, nil, false, 0, s.out, s.logger)
	t.Cleanup(s.player.Unregister)

	transform := defs.Skill(77, 1)
	k := db.CharacterSkill{SkillID: 77, Level: 1, Number: 1}
	if _, ok := s.copyFormTarget(transform, s.char.ID, false); ok {
		t.Fatal("Transformation accepted the caster as the template")
	}
	s.castCopiedForm(packet.SkillUse, k, transform, mobID, s.stats(), false)
	if got := s.transformedMob(); got != mob.MobID {
		t.Fatalf("Transformation picked monster %d, want %d", got, mob.MobID)
	}
	s.breakCombatForm()
	if s.transformedMob() != 0 {
		t.Fatal("Transformation survived combat")
	}

	target := *s.char
	target.ID, target.Name, target.Hair, target.Level = target.ID+1, "Copied Player", 5, s.char.Level
	p := RegisterPlayer(target, []db.Item{{Bag: 0, Slot: 0, Type: 17, TypeID: 1}}, false, 1, nil, s.logger)
	t.Cleanup(p.Unregister)
	disguise := defs.Skill(107, 1)
	s.castCopiedForm(packet.SkillUseNoTarget, db.CharacterSkill{SkillID: 107, Level: 1, Number: 2}, disguise, p.CharacterID, s.stats(), false)
	look := s.player.formLook.Load()
	if look == nil || look.kind != proto.FormDisguise {
		t.Fatalf("Disguise was not published: %+v", look)
	}
	shape, err := proto.DecodePlayerForm(buildFormAppearance(s.char.ID, *look)[2:])
	if err != nil || shape.User != s.char.ID || shape.Shape.Name != target.Name || shape.Shape.Hair != target.Hair || shape.Shape.Equipment[0][0] != 17 {
		t.Fatalf("disguise snapshot %+v (%v)", shape, err)
	}
	if s.char.Name == target.Name || s.player.Faction() == p.Faction() {
		t.Fatal("appearance replaced the real server identity")
	}
	s.breakCombatForm()
	if s.player.formLook.Load() != nil {
		t.Fatal("Disguise survived combat")
	}
	target.Level++
	p.UpdateCharacter(target)
	if _, ok := s.copyFormTarget(disguise, p.CharacterID, false); ok {
		t.Fatal("Disguise accepted a higher-level player")
	}
}

func TestEvolutionGrantsNativeMonsterSkillUntilExpiry(t *testing.T) {
	base, defs := lapisSession(t)
	target, md := lapisMob(t, base, defs)
	base.char.PosX, base.char.PosZ, base.char.Level = 1, 1, md.Level
	s, wire := gmWireSession(t, *base.char, false)
	s.items = base.items
	s.player = RegisterPlayer(*s.char, nil, false, 0, s.out, s.logger)
	t.Cleanup(s.player.Unregister)

	for _, pair := range []struct{ mob, skill uint16 }{{2049, 0}, {413, 36}, {679, 119}} {
		actions := evolutionActions(defs.Mobs[pair.mob])
		if pair.skill == 0 && actions[0].skill != nil || pair.skill != 0 &&
			(actions[0].skill == nil || actions[0].skill.SkillID != pair.skill || actions[0].slot != 1) || actions[1].skill != nil {
			t.Fatalf("monster %d acquired the wrong slots: %+v", pair.mob, actions)
		}
	}
	testMob := *defs.Mobs[413]
	testMob.Attacks[2] = db.MobAttack{Type: 119, OK: 1}
	actions := evolutionActions(&testMob)
	if actions[0].slot != 1 || actions[1].skill == nil || actions[1].skill.SkillID != 119 || actions[1].slot != 2 {
		t.Fatalf("second monster skill was not granted: %+v", actions)
	}
	skill := *defs.NPCSkill(36)
	skill.SuccessType, skill.SuccessValue = 1, 100
	defs.NPCSkills[db.SkillKey{SkillID: 36, Level: 1}] = &skill

	evolution := defs.Skill(127, 2)
	before := s.stats()
	buff, ok := s.addEffect(evolution, 0, s.char.ID)
	if !ok || s.evolutionGrant == 0 || s.stats() != before {
		t.Fatal("Evolution failed to grant a form without changing stats")
	}
	gen := s.evolutionGrant
	s.handleEvolutionCast(proto.EvolutionCastPacket(gen+1, 0, target))
	s.handleEvolutionCast(proto.EvolutionCastPacket(gen, 1, target))
	if len(s.evolutionReadyAt) != 0 {
		t.Fatal("ungranted generation or slot cast a skill")
	}
	s.handleEvolutionCast(proto.EvolutionCastPacket(gen, 0, target))
	result := nextSkillFeedback(t, wire, proto.SkillUse)
	if result.ID != 36 || result.Level != 100 || result.Target != target || result.Result > 1 ||
		!s.evolutionReadyAt[36].After(time.Now()) || len(s.skillCooldowns) != 0 {
		t.Fatalf("Evolution lost the original skill or its separate cooldown: %+v", result)
	}
	m, _ := MobByID(s.char.MapID, target)
	if len(m.Effects) != 1 || m.Effects[0].Def.SkillID != 36 {
		t.Fatalf("native monster effect did not apply: %+v", m.Effects)
	}
	s.processBuffsAt(buff.EndsAt)
	if s.evolutionGrant != 0 || s.evolutionSkill(gen, 0) != nil || s.transformedMob() != 0 {
		t.Fatal("temporary skills or appearance survived Evolution")
	}
	if _, ok := s.addEffect(evolution, 0, s.char.ID); !ok || s.evolutionGrant == gen || s.definitionReady(&skill) {
		t.Fatal("reapplying Evolution reused the old generation or reset cooldown")
	}
}
