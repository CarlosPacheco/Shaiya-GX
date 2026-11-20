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
	attacker := RegisterPlayer(attackerChar, nil, false, 1, s.out, s.logger)
	t.Cleanup(attacker.Unregister)
	sk := &db.SkillDef{SkillID: 61001, SkillLevel: 1, TypeEffect: typeEffectAttack, TypeDetail: detailDrain, DamageType: 0, DamageHP: 200, SuccessType: 1, SuccessValue: 100}
	st := s.stats()
	st.Absorption = 50
	before := st.CurrentHP
	s.applyUserSkillHit(pendingUserHit{AttackerID: attackerChar.ID, Stats: CharacterStats{Accuracy: st.Evasion}, Skill: sk, First: true, ItemCast: true}, st)
	if before-s.stats().CurrentHP != 150 {
		t.Fatal("absorbed HP loss mismatch")
	}
	heals := attacker.TakeUserHits()
	if len(heals) != 1 || !heals[0].DrainHeal || heals[0].Heal[0] != 150 {
		t.Fatal("drain healed unabsorbed amount", heals)
	}
	timeout := time.After(time.Second)
	for {
		select {
		case p := <-wire:
			if proto.Op(p) != proto.Opcode(opUserSkillDrain) {
				continue
			}
			if len(p) < 26 || binary.LittleEndian.Uint16(p[14:]) != 150 || binary.LittleEndian.Uint16(p[20:]) != 150 {
				t.Fatalf("wire drain uses pre-absorption damage: %x", p)
			}
			return
		case <-timeout:
			t.Fatal("missing PvP drain result")
		}
	}
}
