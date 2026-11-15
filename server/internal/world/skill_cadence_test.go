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
	s.char.CurrentHP = 1
	return s
}
func TestGlobalSkillCadenceRejectsAlternatingPacketFloodWithoutExtendingIt(t *testing.T) {
	s := cadenceSession(t, 0)
	cast := func(slot uint8) {
		s.handleUseSkill(packet.SkillUseNoTarget, proto.UseSkillPacket(slot, s.char.ID, true))
	}
	before := time.Now()
	cast(1)
	if s.char.CurrentHP <= 1 || s.nextSkillAt.Before(before.Add(skilltiming.CastInterval)) {
		t.Fatal("accepted skill did not start global cadence")
	}
	if !s.attackMoveUntil.After(s.nextSkillAt) {
		t.Fatal("movement pause was shortened to the cast cadence")
	}
	hp := s.char.CurrentHP
	// Hold a deterministic future boundary: the flood cannot bypass or bank it.
	deadline := time.Now().Add(time.Hour)
	s.nextSkillAt = deadline
	for i := range 100 {
		cast(uint8(i%2 + 1))
	}
	if s.char.CurrentHP != hp || s.nextSkillAt != deadline {
		t.Fatal("alternating flood bypassed or extended global cadence")
	}
	s.nextSkillAt = time.Now().Add(-time.Nanosecond)
	cast(2)
	if s.char.CurrentHP <= hp {
		t.Fatal("different skill did not resume when cadence expired")
	}
}
func TestShortPreparedCastCompletesInsideItsRequestCadence(t *testing.T) {
	s := cadenceSession(t, 1)
	s.handleUseSkill(packet.SkillUseNoTarget, proto.UseSkillPacket(1, s.char.ID, true))
	if s.cast == nil || !s.nextSkillAt.After(time.Now()) {
		t.Fatal("prepared cast not started")
	}
	s.cast.readyAt = time.Now().Add(-time.Second)
	s.processPendingCast()
	if s.cast != nil || s.char.CurrentHP <= 1 {
		t.Fatal("request cadence incorrectly rejected its own cast completion")
	}
	if !s.nextSkillAt.After(time.Now()) {
		t.Fatal("release did not renew practical cadence")
	}
}
