package world

import (
	"fmt"
	"maps"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func lapisSession(t *testing.T) (*Session, *db.Defs) {
	t.Helper()
	s, _ := forgeRuleSession(t)
	loaded, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	defs := *loaded
	defs.Items = maps.Clone(loaded.Items)
	defs.Skills = maps.Clone(loaded.Skills)
	defs.Mobs = maps.Clone(loaded.Mobs)
	db.SetDefsForTest(&defs)
	defs.Items[db.ItemKey{Type: 1, TypeID: 1}] = &db.ItemDef{Type: 1, TypeID: 1, Server: 1, ReqWis: 1, AttackTime: 5, Effect1: 1000, ConstHP: 20000, ConstSP: 20000, ConstMP: 20000}
	s.items = []db.Item{{Bag: 0, Slot: weaponSlot, Type: 1, TypeID: 1, Count: 1}}
	s.char.Str, s.char.Rec, s.char.Int, s.char.Wis, s.char.Dex = 300, 300, 300, 300, 300
	s.char.CurrentHP, s.char.CurrentSP, s.char.CurrentMP = 15000, 15000, 15000
	s.skillCooldowns = map[uint16]time.Time{}
	s.itemCooldowns = map[uint8]time.Time{}
	return s, &defs
}

func TestAllCatalogLapisSelectNPCNamespaceAndDefinedChance(t *testing.T) {
	s, defs := lapisSession(t)
	for ident := uint8(151); ident <= 176; ident++ {
		item := s.items[0]
		item.Gems[0] = ident
		gem := defs.Item(30, ident)
		selected := gemSkillsOf(&item)
		if len(selected) != 1 || selected[0] != defs.NPCSkill(uint16(gem.Special)) || selected[0].SkillLevel != 100 {
			t.Fatalf("lapis %d wrong namespace: %v", ident, selected)
		}
		want := uint8(8)
		if ident >= 164 {
			want = 13
		}
		if ident == 176 {
			want = 15
		}
		if selected[0].SuccessType != 1 || selected[0].SuccessValue != want {
			t.Fatalf("lapis %d lost NpcSkill chance", ident)
		}
	}
	// A same-numbered player buff never supplies an NPC effect.
	delete(defs.Skills, db.SkillKey{SkillID: 205, Level: 100})
	if defs.Skill(205, 1) == nil {
		t.Fatal("missing collision fixture")
	}
	s.items[0].Gems[0] = 151
	if len(weaponGemSkills(s.items)) != 0 {
		t.Fatal("player skill substituted for NpcSkill")
	}
	// Defensive resistance lapis stay ignored instead of counter-casting blindness.
	it := db.Item{Bag: 0, Slot: 1, Type: 17, TypeID: 1, Gems: [6]uint8{117}}
	if len(gemSkillsOf(&it)) != 0 {
		t.Fatal("Anti-Darkness cast an offensive skill")
	}
}

func TestLapisDefinedChanceBoundariesOnPlayersAndMobs(t *testing.T) {
	_, defs := lapisSession(t)
	sk := *defs.NPCSkill(205)
	for _, chance := range []uint8{0, 100} {
		sk.SuccessValue = chance
		for i := 0; i < 200; i++ {
			p := userSkillHits(&CharacterStats{}, &CharacterStats{}, &sk)
			m := playerSkillHits(&CharacterStats{}, &mobCombat{}, &db.MobDef{}, &sk)
			if p != (chance == 100) || m != (chance == 100) {
				t.Fatal("chance boundary ignored", chance, p, m)
			}
		}
	}
	// State-resistant mobs remain immune even to a 100% proc.
	sk.SuccessValue = 100
	immune := db.MobDef{}
	immune.ResistState[sk.StateType-1] = 1
	if playerSkillHits(&CharacterStats{}, &mobCombat{}, &immune, &sk) {
		t.Fatal("proc bypassed mob state immunity")
	}
}

func TestLapisDebuffFamiliesApplyExpireAndDrainPools(t *testing.T) {
	for _, id := range []uint16{205, 206, 208, 209, 211, 212, 214, 215, 217, 218, 220, 221, 223, 224, 226, 227, 229, 230, 232, 233, 235, 236, 238, 239, 241, 242} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			s, defs := lapisSession(t)
			defs.Items[db.ItemKey{Type: 1, TypeID: 1}].ExpAbsorb = 255
			sk := *defs.NPCSkill(id)
			sk.SuccessValue = 100
			defs.Skills[db.SkillKey{SkillID: id, Level: 100}] = &sk
			before := s.stats()
			speed := s.speeds()
			if s.applyUserSkillHit(pendingUserHit{AttackerID: 990001, Skill: &sk, First: true, ItemCast: true}, before) {
				t.Fatal("zero-damage debuff killed target")
			}
			if len(s.buffs) != 1 || s.buffs[0].SkillLevel != 100 {
				t.Fatal("debuff not applied in NPC namespace")
			}
			after := s.stats()
			if after.CurrentHP != before.CurrentHP {
				t.Fatal("zero-HP debuff created absorbed damage")
			}
			family := id
			if (id-205)%3 == 1 {
				family--
			}
			switch family {
			case 205, 241:
				if !s.actionBlocked() || s.speeds().move != 255 {
					t.Fatal("stun/sleep did not block actions")
				}
			case 208:
				if s.actionBlocked() || s.speeds().move != 255 {
					t.Fatal("root should block movement only")
				}
			case 211:
				if after.Disabled[0] <= 0 {
					t.Fatal("blindness did not disable physical attacks")
				}
			case 214:
				if after.Disabled[2] <= 0 {
					t.Fatal("silence did not disable magic")
				}
			case 217:
				if s.speeds().move >= speed.move {
					t.Fatal("movement slow missing")
				}
			case 220:
				if after.Guard[0] >= before.Guard[0] || after.Guard[2] >= before.Guard[2] {
					t.Fatal("defense reduction missing")
				}
			case 223:
				if after.Attack[0] >= before.Attack[0] || after.Attack[2] >= before.Attack[2] {
					t.Fatal("attack reduction missing")
				}
			case 226:
				if s.speeds().attack >= speed.attack {
					t.Fatal("attack-speed slow missing")
				}
			case 238:
				for i := 0; i < 3; i++ {
					if after.Accuracy[i] >= before.Accuracy[i] {
						t.Fatal("accuracy reduction missing")
					}
				}
			case 229, 232, 235:
				now := time.Now()
				s.buffs[0].prevTick = now.Add(-effectTick)
				// Known percentage at exactly one tick; absorption must not reduce DoT.
				damage := playerTimeDamage(&sk, &before, effectTick)
				s.buffs = append(s.buffs, ActiveBuff{ID: 2, Effect: typeEffectBuff, EndsAt: now.Add(time.Hour)})
				if s.tickDebuffs(now) {
					t.Fatal("unexpected death")
				}
				got := s.stats()
				if got.CurrentHP != before.CurrentHP-uint32(damage[0]) || got.CurrentSP != before.CurrentSP-uint32(damage[1]) || got.CurrentMP != before.CurrentMP-uint32(damage[2]) {
					t.Fatalf("periodic resource loss not applied: %v -> %v damage%v", before, got, damage)
				}
			}
			for i := range s.buffs {
				s.buffs[i].EndsAt = time.Now().Add(-time.Millisecond)
			}
			s.processBuffs()
			if len(s.buffs) != 0 || s.actionBlocked() || s.stats().Disabled != ([3]int32{}) {
				t.Fatal("expired debuff left restrictions")
			}
		})
	}
}

func TestSilenceBlocksHealingAndPreparedCastAndPreservesPlayerNamespace(t *testing.T) {
	s, defs := lapisSession(t)
	heal := &db.SkillDef{SkillID: 61000, SkillLevel: 1, TypeEffect: typeEffectHeal, TypeDetail: detailHeal, TargetType: targetTypeSelf, Kind: attackMagic, HealHP: 50, ReadyTime: 4}
	defs.Skills[db.SkillKey{SkillID: heal.SkillID, Level: 1}] = heal
	s.skills = []db.CharacterSkill{{SkillID: heal.SkillID, Level: 1, Number: 1}}
	s.cast = &pendingCast{number: 1, readyAt: time.Now().Add(time.Hour)}
	silence := defs.NPCSkill(214)
	s.addEffect(silence, 0)
	if s.cast != nil {
		t.Fatal("silence did not interrupt prepared magic")
	}
	hp := s.char.CurrentHP
	s.handleUseSkill(packet.SkillUseNoTarget, proto.UseSkillPacket(1, s.char.ID, true))
	s.resolveSkill(s.char.ID, 1, packet.SkillUseNoTarget, nil)
	if s.cast != nil || s.char.CurrentHP != hp || len(s.skillCooldowns) != 0 {
		t.Fatal("silenced healing cast succeeded")
	}
	// A rank-one player buff with the same ID coexists with NPC rank 100.
	player := &db.SkillDef{SkillID: 214, SkillLevel: 1, TypeEffect: typeEffectBuff, KeepSeconds: 10}
	s.addEffect(player, 0, s.char.ID)
	if len(s.buffs) != 2 {
		t.Fatal("NPC debuff replaced/refused same-numbered player buff")
	}
}
