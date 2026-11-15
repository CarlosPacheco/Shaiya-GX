package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"testing"
	"time"
)

func cadenceSession(t *testing.T, ready uint8) *Session {
	s, d, _ := spellCastRangeFixture(t)
	s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	d.TypeEffect, d.TypeDetail, d.TargetType = typeEffectHeal, detailHeal, targetTypeSelf
	d.ReadyTime, d.ResetTimeSeconds, d.HealHP = ready, 1, 5
	second := *d
	second.SkillID++
	db.MustDefs().Skills[db.SkillKey{SkillID: second.SkillID, Level: 1}] = &second
	s.skills = append(s.skills, db.CharacterSkill{SkillID: second.SkillID, Level: 1, Number: 2})
	weapon := *db.MustDefs().Item(1, 1)
	weapon.AttackTime = 5
	db.MustDefs().Items[db.ItemKey{Type: 1, TypeID: 1}] = &weapon
	s.items = []db.Item{{Bag: 0, Slot: weaponSlot, Type: 1, TypeID: 1, Count: 1}}
	s.char.CurrentHP = 1
	return s
}
func TestSharedCadenceRejectsAlternatingSpellAndBasicFlood(t *testing.T) {
	s := cadenceSession(t, 0)
	cast := func(slot uint8) {
		s.handleUseSkill(packet.SkillUseNoTarget, proto.UseSkillPacket(slot, s.char.ID, true))
	}
	before := time.Now()
	cast(1)
	if s.char.CurrentHP <= 1 || s.lastActionAt.Before(before) {
		t.Fatal("accepted skill did not reserve shared action clock")
	}
	hp := s.char.CurrentHP
	held := time.Now().Add(time.Hour)
	s.lastActionAt = held
	for i := range 100 {
		cast(uint8(i%2 + 1))
		if !s.autoAttackSwing(987654) || !s.userAttackSwing(987654) {
			t.Fatal("basic attack reached target lookup during shared recovery")
		}
	}
	if s.char.CurrentHP != hp || s.lastActionAt != held {
		t.Fatal("alternating actions bypassed or extended recovery")
	}
	interval, _ := skilltiming.Interval(s.speeds().attack)
	s.lastActionAt = time.Now().Add(-interval)
	cast(2)
	if s.char.CurrentHP <= hp {
		t.Fatal("different skill did not resume after native recovery")
	}
}
func TestSharedCombatCadenceUsesEveryNativeTierAndLiveHaste(t *testing.T) {
	s := cadenceSession(t, 0)
	now := time.Now()
	s.lastActionAt = now
	for tier := uint8(0); tier <= 9; tier++ {
		s.buffs = []ActiveBuff{{Effect: typeEffectBuff, Abilities: [3]buffAbility{{Type: 16, Value: int32(tier) - 5}}}}
		interval, ok := skilltiming.Interval(tier)
		if !ok || s.speeds().attack != tier || s.combatActionReady(now.Add(interval-time.Nanosecond)) || !s.combatActionReady(now.Add(interval)) {
			t.Fatal("wrong shared recovery", tier, interval, s.speeds())
		}
	}
	s.buffs = nil
	if s.combatActionReady(now.Add(time.Second)) {
		t.Fatal("normal-speed recovery too short")
	}
	s.buffs = []ActiveBuff{{Effect: typeEffectBuff, Abilities: [3]buffAbility{{Type: 16, Value: 4}}}}
	if !s.combatActionReady(now.Add(time.Second)) {
		t.Fatal("live haste did not accelerate recovery")
	}
	s.buffs = []ActiveBuff{{Effect: typeEffectDebuff, Detail: detailStun}}
	if s.combatActionReady(now.Add(time.Hour)) {
		t.Fatal("blocked attack speed allowed a spell")
	}
}
func TestPreparedReleaseDoesNotChargeASecondAttackInterval(t *testing.T) {
	s := cadenceSession(t, 1)
	s.handleUseSkill(packet.SkillUseNoTarget, proto.UseSkillPacket(1, s.char.ID, true))
	if s.cast == nil || s.lastActionAt.IsZero() {
		t.Fatal("prepared cast not started")
	}
	start := s.lastActionAt
	s.cast.readyAt = time.Now().Add(-time.Second)
	s.processPendingCast()
	if s.cast != nil || s.char.CurrentHP <= 1 || s.lastActionAt != start {
		t.Fatal("prepared release restarted attack recovery")
	}
	interval, _ := skilltiming.Interval(s.speeds().attack)
	if !s.combatActionReady(start.Add(interval)) {
		t.Fatal("prepared cast charged two intervals")
	}
}
