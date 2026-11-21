package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"github.com/Spelunkern/Shaiya-GX/internal/attackspeed"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"testing"
	"time"
)

func TestNativeWeaponAccessoryLapisMasteryAndBuffSpeed(t *testing.T) {
	s, _ := forgeRuleSession(t)
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	ring, sonic, flash := defs.Item(22, 56), defs.Item(23, 57), defs.Item(30, 64)
	if ring.AttackTime != 1 || sonic.Speed != 1 || flash.AttackTime != 1 {
		t.Fatal("native bonus fields", ring, sonic, flash)
	}
	weapon := defs.Item(1, 1)
	s.items = []db.Item{{Bag: 0, Slot: 5, Type: 1, TypeID: 1, Count: 1}}
	check := func(bonus int32, move uint8) {
		t.Helper()
		sp := s.speeds()
		want := attackspeed.Tier(weapon.AttackTime, bonus)
		if sp.attack != want || sp.move != move {
			t.Fatalf("speed %+v expected attack %d move %d", sp, want, move)
		}
	}
	check(0, 2)
	s.items = append(s.items, db.Item{Bag: 1, Type: 22, TypeID: 56})
	check(0, 2)
	s.items[1].Bag, s.items[1].Slot = 0, 9
	check(1, 2)
	s.items = append(s.items, db.Item{Bag: 0, Slot: 8, Type: 23, TypeID: 57})
	check(1, 3)
	s.items[0].Gems[0] = 64
	check(2, 3)
	s.skills = []db.CharacterSkill{{SkillID: 15, Level: 1}}
	check(3, 3)
	haste := defs.Skill(129, 1)
	if haste == nil || haste.AbilityType[0] != 16 {
		t.Fatal("native attack haste")
	}
	s.addEffect(haste, 0, s.char.ID)
	check(5, 3)
	s.removeEffects(func(*ActiveBuff) bool { return true }, "expiry")
	check(3, 3)
	s.skills = nil
	s.items[0].Gems[0] = 0
	s.items = s.items[:1]
	check(0, 2)
	// Direct zero-based raw field: no premature clamp before a negative modifier.
	raw := *weapon
	raw.AttackTime = 10
	defs.Items[db.ItemKey{Type: 1, TypeID: 1}] = &raw
	s.buffs = []ActiveBuff{{Effect: typeEffectDebuff, Abilities: [3]buffAbility{{Type: 16, Value: 1}}}}
	if got := s.speeds().attack; got != 9 {
		t.Fatal("raw base clamped before modifiers", got)
	}
}

func TestAttackCadenceCannotBeBypassedByRestartOrTargetSwitch(t *testing.T) {
	skilltiming.AnimationBound = false
	t.Cleanup(func() { skilltiming.AnimationBound = true })
	s, _ := gmWireSession(t, db.Character{ID: 990070, MapID: 60094, CurrentHP: 100}, false)
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{{Type: 1, TypeID: 1}: {Type: 1, TypeID: 1, AttackTime: 5}}})
	s.items = []db.Item{{Bag: 0, Slot: 5, Type: 1, TypeID: 1}}
	for tier := uint8(0); tier <= 9; tier++ {
		s.buffs = []ActiveBuff{{Effect: typeEffectBuff, Abilities: [3]buffAbility{{Type: 16, Value: int32(tier) - 5}}}}
		ms, _ := attackspeed.Milliseconds(tier)
		if d, ok := weaponAttackDelay(tier, 5); !ok || d != time.Duration(ms)*time.Millisecond {
			t.Fatal("delay mismatch")
		}
		s.lastActionAt = time.Now().Add(-time.Duration(ms-100) * time.Millisecond)
		last := s.lastActionAt
		// Fake targets must not even reach target lookup while still cooling down.
		for i := range 200 {
			s.dispatch(packet.AutoAttackStop, proto.StopAttackPacket())
			s.dispatch(packet.CharacterMobAutoAttack, proto.MobAttackPacket(uint32(980000+i)))
			if s.lastActionAt != last || !s.autoAttack {
				t.Fatalf("tier %d restart bypassed cadence", tier)
			}
		}
	}
	s.cast = &pendingCast{}
	if s.speeds().attack != 255 || s.autoAttackSwing(980000) || s.userAttackSwing(980000) {
		t.Fatal("casting did not block normal attacks")
	}
	s.cast = nil
	s.buffs = []ActiveBuff{{Effect: typeEffectDebuff, Detail: detailStun}}
	if s.autoAttackSwing(980000) {
		t.Fatal("stun allowed attack")
	}
}

func TestNativeAttackCandyConsumesCoolsDownAndExpires(t *testing.T) {
	s, add := forgeRuleSession(t)
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	candy := defs.Item(25, 175)
	if candy == nil || candy.Range != 211 || candy.AttackTime != 1 {
		t.Fatal("native attack candy mapping")
	}
	add(db.Item{Bag: 0, Slot: 5, Type: 1, TypeID: 1, Count: 1})
	item := add(db.Item{Bag: 1, Slot: 0, Type: 25, TypeID: 175, Count: 3})
	s.itemCooldowns = map[uint8]time.Time{}
	s.handleUseItem(proto.UseItemPacket(1, 0))
	if sp := s.speeds(); sp.attack != 6 || sp.move != 2 || s.lastSpeeds.attack != 6 || len(s.buffs) != 1 {
		t.Fatal("attack candy did not apply", sp, s.buffs)
	}
	s.handleUseItem(proto.UseItemPacket(1, 0))
	if it := s.items[findItemByUID(s.items, item.UID)]; it.Count != 2 {
		t.Fatal("cooldown consumed another candy", it)
	}
	s.buffs[0].EndsAt = time.Now().Add(-time.Second)
	s.processBuffs()
	if sp := s.speeds(); sp.attack != 5 || s.lastSpeeds.attack != 5 {
		t.Fatal("attack candy expiry did not restore native base", sp)
	}
}
