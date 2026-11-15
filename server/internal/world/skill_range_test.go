package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skillrange"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"testing"
	"time"
)

func spellCastRangeFixture(t *testing.T) (*Session, *db.SkillDef, *mapShard) {
	t.Helper()
	s, _ := forgeRuleSession(t)
	s.char.MapID, s.char.PosX, s.char.PosY, s.char.PosZ = 60044, 0, 0, 0
	g := &mapgeometry.Geometry{}
	g.Reindex()
	s.movement.geometry = g
	s.movement.mapID = s.char.MapID
	d := &db.SkillDef{SkillID: 64001, SkillLevel: 1, TypeEffect: typeEffectAttack, TypeDetail: detailAttack, TargetType: 3, AttackRange: 10, ReadyTime: 4, ResetTimeSeconds: 5, SuccessType: 1, SuccessValue: 100, DamageHP: 10}
	defs := db.MustDefs()
	defs.Skills = map[db.SkillKey]*db.SkillDef{{SkillID: d.SkillID, Level: 1}: d}
	defs.Mobs = map[uint16]*db.MobDef{64001: {MobID: 64001, MaxHP: 100}}
	s.skills = []db.CharacterSkill{{SkillID: d.SkillID, Level: 1, Number: 1}}
	s.skillCooldowns = map[uint16]time.Time{}
	sh := shard(s.char.MapID)
	sh.mu.Lock()
	sh.entities = MapEntities{Mobs: []MapMob{{ID: 64001, MobID: 64001, PosX: 10, CurrentHP: 100, MaxHP: 100}}}
	sh.mu.Unlock()
	t.Cleanup(func() { sh.mu.Lock(); sh.entities = MapEntities{}; sh.mu.Unlock() })
	return s, d, sh
}
func TestSpellCastValidatesStartAndCurrentReleaseRange(t *testing.T) {
	s, d, sh := spellCastRangeFixture(t)
	move := func(x float32) { sh.mu.Lock(); sh.entities.Mobs[0].PosX = x; sh.mu.Unlock() }
	cast := func() { s.handleUseSkill(packet.SkillUse, proto.UseSkillPacket(1, 64001, false)) }
	move(50)
	cast()
	if s.cast != nil || len(s.skillCooldowns) != 0 {
		t.Fatal("out-of-range mob started casting")
	}
	move(10)
	cast()
	if s.cast == nil {
		t.Fatal("valid cast failed")
	}
	move(15)
	s.cast.readyAt = time.Now().Add(-time.Second)
	s.processPendingCast()
	if len(s.skillCooldowns) != 0 {
		t.Fatal("cast release used old 100-unit escape allowance")
	}
	move(10)
	cast()
	if s.cast == nil {
		t.Fatal("second valid cast failed")
	}
	move(14.7)
	s.cast.readyAt = time.Now().Add(-time.Second)
	s.processPendingCast()
	if !s.skillCooldowns[d.SkillID].After(time.Now()) {
		t.Fatal("bounded release grace was not honored")
	}
}
func TestSpellReleaseAndAreaRejectWallsAndDifferentFloors(t *testing.T) {
	s, d, sh := spellCastRangeFixture(t)
	s.handleUseSkill(packet.SkillUse, proto.UseSkillPacket(1, 64001, false))
	if s.cast == nil {
		t.Fatal("valid cast failed")
	}
	for _, tri := range collision.BlockerBox(collision.Vec{4, -1, -2}, collision.Vec{5, 3, 2}, 0) {
		s.movement.geometry.Add(tri)
	}
	s.cast.readyAt = time.Now().Add(-time.Second)
	s.processPendingCast()
	if len(s.skillCooldowns) != 0 {
		t.Fatal("cast passed through new obstruction")
	}
	sh.mu.Lock()
	mob := sh.entities.Mobs[0]
	sh.mu.Unlock()
	if s.mobSkillReach(&mob, d, false) {
		t.Fatal("obstructed start accepted")
	}
	if s.skillAreaContains(skillrange.Position{}, skillrange.Position{10, 0, 0}, 20) {
		t.Fatal("area passed through wall")
	}
	if s.skillAreaContains(skillrange.Position{}, skillrange.Position{0, 21, 0}, 20) {
		t.Fatal("area radius ignored height")
	}
}

func TestUserSpellReachUsesCurrentCasterAndTarget(t *testing.T) {
	s, d, _ := spellCastRangeFixture(t)
	p := &ConnectedPlayer{character: db.Character{PosX: 14}}
	if s.userSkillReach(p, d, false) || !s.userSkillReach(p, d, true) {
		t.Fatal("user launch/release grace differs")
	}
	s.char.PosX = -1
	if s.userSkillReach(p, d, true) {
		t.Fatal("release ignored current caster position")
	}
	s.char.PosX = 0
	p.character.PosX, p.character.PosY = 0, 15
	if s.userSkillReach(p, d, true) {
		t.Fatal("user height ignored")
	}
	p.character.PosY = 0
	s.queueSkillHits(p, d, true, 1)
	p.character.PosX = 1000
	if len(p.TakeUserHits()) != 1 {
		t.Fatal("accepted hit revoked after departure")
	}
}

func TestPreparedFriendlySpellUsesUserReach(t *testing.T) {
	s, d, _ := spellCastRangeFixture(t)
	s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	c := *s.char
	c.ID++
	c.PosX = 10
	c.CurrentHP = 100
	p := RegisterPlayer(c, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(p.Unregister)
	d.TypeEffect = typeEffectHeal
	d.TypeDetail = detailHeal
	d.HealHP = 10
	s.handleUseSkill(packet.SkillUseNoTarget, proto.UseSkillPacket(1, c.ID, true))
	if s.cast == nil {
		t.Fatal("prepared friendly skill incorrectly looked for a mob")
	}
	c.PosX = 14
	p.UpdateCharacter(c)
	s.cast.readyAt = time.Now().Add(-time.Second)
	s.processPendingCast()
	if len(p.TakeFriendly()) != 1 {
		t.Fatal("friendly cast did not honor release grace")
	}
}
