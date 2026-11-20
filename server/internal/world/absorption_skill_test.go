package world

import (
	"encoding/binary"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func TestMobSkillAbsorptionAndWireDamage(t *testing.T) {
	for _, tt := range []struct {
		name   string
		hp     int32
		absorb uint32
		chance uint8
		want   uint32
	}{
		{"subtract", 200, 50, 100, 150}, {"minimum", 20, 500, 100, 1}, {"resource_only", 0, 500, 100, 0}, {"miss", 200, 50, 0, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			base, defs := lapisSession(t)
			id, md := lapisMob(t, base, defs)
			s, wire := gmWireSession(t, *base.char, false)
			s.items = base.items
			s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
			t.Cleanup(s.player.Unregister)
			sk := &db.SkillDef{SkillID: 61000, SkillLevel: 100, TypeEffect: typeEffectAttack, TypeDetail: detailAttack, DamageType: 0, DamageHP: tt.hp, DamageSP: 50, DamageMP: 60, SuccessType: 1, SuccessValue: tt.chance}
			before := s.stats()
			st := before
			st.Absorption = tt.absorb
			mc := mobCombatFor(md)
			mc.Accuracy = st.Evasion
			mc.Level = int32(st.Level)
			s.applyMobSkill(pendingDamage{MobGlobalID: id, MobID: md.MobID, First: true}, md, &mc, sk, &st)
			if got := before.CurrentHP - s.stats().CurrentHP; got != tt.want {
				t.Fatalf("HP loss=%d want%d", got, tt.want)
			}
			if tt.chance == 100 && (before.CurrentSP-s.stats().CurrentSP != 50 || before.CurrentMP-s.stats().CurrentMP != 60) {
				t.Fatal("mob skill resource damage missing")
			}
			timeout := time.After(time.Second)
			for {
				select {
				case p := <-wire:
					if proto.Op(p) != 0x060D {
						continue
					}
					if len(p) < 20 || uint32(binary.LittleEndian.Uint16(p[14:])) != tt.want {
						t.Fatalf("wire HP differs from absorbed hit: %x", p)
					}
					return
				case <-timeout:
					t.Fatal("missing mob-skill result")
				}
			}
		})
	}
}

func TestPVPSkillAbsorptionControlsDisplayAndDrain(t *testing.T) {
	base, _ := lapisSession(t)
	s, wire := gmWireSession(t, *base.char, false)
	s.items = base.items
	s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	attackerChar := *s.char
	attackerChar.ID++
	attacker, _ := gmWireSession(t, attackerChar, false)
	attacker.items = base.items
	max := attacker.stats()
	attacker.char.CurrentHP, attacker.char.CurrentSP, attacker.char.CurrentMP = max.MaxHP-60, max.MaxSP-20, max.MaxMP
	attacker.player = RegisterPlayer(*attacker.char, nil, false, 1, attacker.out, attacker.logger)
	t.Cleanup(attacker.player.Unregister)
	sk := &db.SkillDef{SkillID: 61001, SkillLevel: 1, TypeEffect: typeEffectAttack, TypeDetail: detailDrain,
		DamageType: 0, DamageHP: 200, DamageSP: 50, DamageMP: 60, SuccessType: 1, SuccessValue: 100}
	st := s.stats()
	st.Absorption = 50
	before := st.CurrentHP
	s.applyUserSkillHit(pendingUserHit{AttackerID: attackerChar.ID, Stats: CharacterStats{Accuracy: st.Evasion}, Skill: sk, First: true, ItemCast: true}, st)
	if before-s.stats().CurrentHP != 150 {
		t.Fatal("absorbed HP loss mismatch")
	}
	attacker.tickUserHits()
	if st := attacker.stats(); st.CurrentHP != st.MaxHP || st.CurrentSP != st.MaxSP || st.CurrentMP != st.MaxMP {
		t.Fatal("caster was not healed")
	}
	r := nextSkillFeedback(t, wire, proto.SkillUserDrain)
	if r.HP != 150 || r.SP != 50 || r.MP != 60 || r.HealedHP != 60 || r.HealedSP != 20 || r.HealedMP != 0 {
		t.Fatalf("PvP drain result disagrees with target damage or caster recovery: %+v", r)
	}
}

func TestMobResourceDrainIsRepeatableWithoutResourceDamage(t *testing.T) {
	base, defs := lapisSession(t)
	id, md := lapisMob(t, base, defs)
	md.MaxMP, md.MaxSP = 1, 1 // Legacy resource fields do not cap fixed drains.
	s, wire := gmWireSession(t, *base.char, false)
	s.items = base.items
	s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	s.char.Level = md.Level
	def := &db.SkillDef{SkillID: 61001, SkillLevel: 1, TypeEffect: typeEffectAttack,
		TypeDetail: detailDrain, DamageSP: 30, DamageMP: 50, SuccessType: 1, SuccessValue: 100}
	k := db.CharacterSkill{SkillID: def.SkillID, Level: def.SkillLevel}
	for range 2 {
		before := s.stats()
		s.skillHit(id, def, k, true)
		after := s.stats()
		r := nextSkillFeedback(t, wire, proto.SkillMobDrain)
		if r.Result != 0 || r.HP != 0 || r.MP != 0 || r.SP != 0 || r.HealedMP != 50 || r.HealedSP != 30 ||
			after.CurrentMP-before.CurrentMP != 50 || after.CurrentSP-before.CurrentSP != 30 {
			t.Fatalf("incorrect monster drain: %+v", r)
		}
	}
	def.TypeDetail = detailAttack
	def.AddDamageMP, def.AddDamageSP = 50, 30
	before := s.stats()
	s.skillHit(id, def, k, true)
	r := nextSkillFeedback(t, wire, proto.SkillUse)
	if after := s.stats(); r.Result != 0 || r.HP != 0 || r.MP != 0 || r.SP != 0 || after.CurrentMP != before.CurrentMP || after.CurrentSP != before.CurrentSP {
		t.Fatalf("resource destruction affected the monster or caster: %+v", r)
	}
}

func nextSkillFeedback(t *testing.T, wire <-chan []byte, op proto.Opcode) proto.SkillResult {
	t.Helper()
	timeout := time.After(time.Second)
	for {
		select {
		case p := <-wire:
			if proto.Op(p) != op {
				continue
			}
			result, err := proto.DecodeSkillResult(p)
			if err != nil {
				t.Fatal(err)
			}
			return result
		case <-timeout:
			t.Fatalf("missing skill result %#x", op)
		}
	}
}
