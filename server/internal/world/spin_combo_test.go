package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"testing"
	"time"
)

func TestSpinComboExecutesEarlyButRetainsOwnCooldownAndCosts(t *testing.T) {
	s := cadenceSession(t, 0)
	defs := db.MustDefs()
	delete(defs.Skills, db.SkillKey{SkillID: 64001, Level: 1})
	delete(defs.Skills, db.SkillKey{SkillID: 64002, Level: 1})
	for i, id := range []uint16{38, 156} {
		defs.Skills[db.SkillKey{SkillID: id, Level: 1}] = &db.SkillDef{SkillID: id, SkillLevel: 1, TypeEffect: 1, TypeDetail: 21, TargetType: 6, ResetTimeSeconds: 10, SPCost: 5}
		s.skills[i] = db.CharacterSkill{SkillID: id, Level: 1, Number: uint8(i + 1)}
	}
	s.char.MaxSP, s.char.CurrentSP = 100, 100
	cast := func(n uint8) { s.handleUseSkill(packet.SkillUse, proto.UseSkillPacket(n, 0, false)) }
	cast(1)
	if s.lastSkillID != 38 || s.char.CurrentSP != 95 {
		t.Fatal("first spin not committed", s.lastSkillID, s.char.CurrentSP)
	}
	cast(2)
	if s.lastSkillID != 38 || s.char.CurrentSP != 95 || s.pendingCombo == nil {
		t.Fatal("early follow-up not retained without spending costs")
	}
	s.lastActionAt = time.Now().Add(-skilltiming.SpinContact - time.Millisecond)
	s.lastSkillAt = s.lastActionAt
	s.pendingCombo.due = time.Now().Add(-time.Millisecond)
	s.pendingCombo.previous = s.lastActionAt
	s.processSpinCombo()
	s.processSpinCombo()
	if s.pendingCombo != nil {
		t.Fatal("committed combo remained queued")
	}

	if s.lastSkillID != 156 || s.char.CurrentSP != 90 {
		t.Fatal("second spin did not abbreviate", s.lastSkillID, s.char.CurrentSP)
	}
	s.lastActionAt = time.Now().Add(-skilltiming.SpinContact - time.Millisecond)
	s.lastSkillAt = s.lastActionAt
	cast(1)
	if s.lastSkillID != 156 || s.char.CurrentSP != 90 {
		t.Fatal("combo bypassed Wind Spin reuse")
	}
	if s.skillActionReady(&db.SkillDef{SkillID: 40, TypeEffect: 4, TargetType: 6}, time.Now()) {
		t.Fatal("buff got combo permission")
	}
}
