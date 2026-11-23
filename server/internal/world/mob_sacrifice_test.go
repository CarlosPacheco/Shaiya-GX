package world

import "testing"

// A mob whose attack is a sacrifice skill ("Self Destruction", mines) dies
// after the blow, with no credit; the player still takes the damage.
func TestMobSacrificeSkillKillsTheMob(t *testing.T) {
	base, defs := lapisSession(t)
	id, mob := lapisMob(t, base, defs)
	base.char.Level = mob.Level
	base.char.CurrentHP = base.stats().MaxHP
	s, _ := gmWireSession(t, *base.char, false)
	s.items = base.items
	s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	skill := defs.NPCSkill(171) // Mine, TypeDetail 23
	if skill == nil || skill.TypeDetail != detailSacrifice {
		t.Fatalf("unexpected NPC skill 171: %+v", skill)
	}
	s.applyIncomingMobDamage(pendingDamage{MobID: mob.MobID, MobGlobalID: id, SkillID: 171})
	if m, ok := MobByID(s.char.MapID, id); ok && m.CurrentHP != 0 {
		t.Fatalf("self-destructing mob survived with %d HP", m.CurrentHP)
	}
}
