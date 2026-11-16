package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skillrange"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"math"
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
	defs.Mobs = map[uint16]*db.MobDef{64001: {MobID: 64001, MaxHP: 100, Country: 2}}
	s.skills = []db.CharacterSkill{{SkillID: d.SkillID, Level: 1, Number: 1}}
	s.skillCooldowns = map[uint16]time.Time{}
	sh := shard(s.char.MapID)
	sh.mu.Lock()
	sh.entities = MapEntities{Mobs: []MapMob{{ID: 64001, MobID: 64001, PosX: 10, CurrentHP: 100, MaxHP: 100}}}
	sh.mu.Unlock()
	t.Cleanup(func() { sh.mu.Lock(); sh.entities = MapEntities{}; sh.mu.Unlock() })
	return s, d, sh
}
func TestPreparedSpellValidatesLaunchWithoutReleaseDistanceLimit(t *testing.T) {
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
	move(150)
	s.cast.readyAt = time.Now().Add(-time.Second)
	s.processPendingCast()
	if !s.skillCooldowns[d.SkillID].After(time.Now()) {
		t.Fatal("original prepared release incorrectly rechecked range")
	}
	// The fixture fast-forwards a full preparation without sleeping.
	s.lastActionAt = time.Time{}
	s.skillCooldowns = map[uint16]time.Time{}
	move(10)
	cast()
	if s.cast == nil {
		t.Fatal("second valid cast failed")
	}
	move(14.7)
	s.cast.readyAt = time.Now().Add(-time.Second)
	s.processPendingCast()
	if !s.skillCooldowns[d.SkillID].After(time.Now()) {
		t.Fatal("prepared release was not honored")
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
	if !s.userSkillReach(p, d, true) {
		t.Fatal("caster distance incorrectly invalidated prepared release")
	}
	s.char.PosX = 0
	p.character.PosX, p.character.PosY = 0, 15
	if !s.userSkillReach(p, d, true) {
		t.Fatal("prepared release incorrectly repeated 3D reach")
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
	if time.Until(s.attackMoveUntil) < 700*time.Millisecond {
		t.Fatal("friendly skill did not stop movement for 800ms")
	}
	if len(p.TakeFriendly()) != 1 {
		t.Fatal("friendly cast did not honor release grace")
	}
}

func TestFriendlyEnemyFallbackAndOneSecondReuse(t *testing.T) {
	s, d, _ := spellCastRangeFixture(t)
	d.TypeEffect = typeEffectHeal
	if got := s.friendlySkillRecipient(d, 64001); got != s.char.ID {
		t.Fatal("enemy was not replaced by self", got)
	}
	if got := s.friendlySkillRecipient(d, 64002); got != 64002 {
		t.Fatal("stale ally silently replaced", got)
	}
	d.TypeEffect = typeEffectAttack
	if got := s.friendlySkillRecipient(d, 64001); got != 64001 {
		t.Fatal("attack recipient changed", got)
	}
	for _, seconds := range []uint16{0, 1, 2, 10} {
		d.ResetTimeSeconds = seconds
		s.startCooldown(s.skills[0], d)
		if got := s.skillReady(d.SkillID); got != (seconds <= 1) {
			t.Fatal("wrong authoritative reuse", seconds, got)
		}
		if time.Until(s.attackMoveUntil) < 700*time.Millisecond {
			t.Fatal("rapid chaining removed movement stop")
		}
	}
}

func TestPreparedHealKeepsSelfRecipientAfterEnemyDisappears(t *testing.T) {
	s, d, sh := spellCastRangeFixture(t)
	s.player = RegisterPlayer(*s.char, nil, false, 0, s.out, s.logger)
	defer s.player.Unregister()
	d.TypeEffect, d.TypeDetail, d.HealHP = typeEffectHeal, detailHeal, 5
	s.char.CurrentHP = 1
	s.handleUseSkill(packet.SkillUse, proto.UseSkillPacket(1, 64001, false))
	if s.cast == nil || s.cast.targetMobID != s.char.ID {
		t.Fatal("prepared heal retained enemy recipient", s.cast)
	}
	sh.mu.Lock()
	sh.entities.Mobs = nil
	sh.mu.Unlock()
	s.cast.readyAt = time.Now().Add(-time.Second)
	s.processPendingCast()
	if s.char.CurrentHP <= 1 {
		t.Fatal("enemy disappearance cancelled self heal")
	}
}

func TestFriendlyMobCastRefusedBeforeCostsAndCooldown(t *testing.T) {
	s, d, sh := spellCastRangeFixture(t)
	def := db.MustDefs().Mobs[64001]
	def.Country = s.faction
	d.MPCost, d.SPCost = 10, 10
	s.char.CurrentMP, s.char.CurrentSP = 100, 100
	wire, packets := gmWireSession(t, *s.char, false)
	s.out, s.aesSend, s.expanded = wire.out, wire.aesSend, true
	before := *s.char
	s.handleUseSkill(packet.SkillUse, proto.UseSkillPacket(1, 64001, false))
	if s.cast != nil || len(s.skillCooldowns) != 0 || *s.char != before {
		t.Fatal("friendly cast mutated character or cooldown")
	}
	sh.mu.Lock()
	mob := sh.entities.Mobs[0]
	mob.def = def
	sh.mu.Unlock()
	spawn, err := proto.DecodeMobEnter(buildMobEnter(&mob, false)[2:])
	if err != nil || !spawn.CountryKnown || spawn.Country != s.faction || spawn.HeightKnown {
		t.Fatalf("friendly spawn metadata %+v: %v", spawn, err)
	}
	select {
	case p := <-packets:
		r := proto.NewReader(p)
		if proto.Op(p) != proto.GXSkillFailure || r.U16() != d.SkillID || r.U16() != 317 {
			t.Fatalf("wrong rejection %x", p)
		}
	case <-time.After(time.Second):
		t.Fatal("wrong-target SysMsg317 not sent")
	}
}

func TestMovementOnlyInterruptsCastAfterValidationAndWithoutNativeFlag(t *testing.T) {
	s, _ := forgeRuleSession(t)
	physics, now := playerPhysicsFixture(t)
	s.char.MapID, s.char.PosX, s.char.PosY, s.char.PosZ = physics.char.MapID, physics.char.PosX, physics.char.PosY, physics.char.PosZ
	s.movement = physics.movement
	s.lastPositionSave = now
	s.cast = &pendingCast{number: 1, readyAt: now.Add(time.Minute)}
	s.speedKnown = true
	s.lastSpeeds = playerSpeeds{attack: 255, move: 2}
	s.handleCharacterMove([]byte{1})
	if s.cast == nil {
		t.Fatal("malformed movement cancelled cast")
	}
	s.handleCharacterMove(proto.MovePacket(1, 0, float32(math.NaN()), 0, 5))
	if s.cast == nil {
		t.Fatal("rejected movement cancelled cast")
	}
	s.handleCharacterMove(proto.MovePacket(proto.MotionFlag80, 0, s.char.PosX, s.char.PosY, s.char.PosZ))
	if s.cast == nil {
		t.Fatal("native non-interrupting movement cancelled cast")
	}
	s.handleCharacterMove(proto.MovePacket(0, 0, s.char.PosX, s.char.PosY, s.char.PosZ))
	if s.cast == nil {
		t.Fatal("idle refresh interrupted preparation")
	}
	s.handleCharacterMove(proto.MovePacket(1, 0, s.char.PosX, s.char.PosY, s.char.PosZ))
	if s.cast != nil || s.lastSpeeds.attack == 255 {
		t.Fatal("accepted interrupt did not clear cast and restore attack speed", s.lastSpeeds)
	}
}

func TestIdlePositionUpdatesDoNotLosePreparedSpell(t *testing.T) {
	s, d, sh := spellCastRangeFixture(t)
	// Use the real Blizzard definition and costs, not an instant or zero-cost skill.
	mobDef := db.MustDefs().Mobs[64001]
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	defs.Mobs[64001] = mobDef
	d = defs.Skill(71, 3)
	if d == nil || d.ReadyTime != 8 {
		t.Fatal("missing original Blizzard preparation")
	}
	s.skills = []db.CharacterSkill{{SkillID: 71, Level: 3, Number: 1}}
	s.char.Wis = 60000
	s.setPools(false)

	for _, tri := range collision.BlockerBox(collision.Vec{-100, -2, -100}, collision.Vec{100, 0, 100}, 0) {
		s.movement.geometry.Add(tri)
	}
	for attempt := 0; attempt < 20; attempt++ {
		s.lastActionAt = time.Time{}
		s.skillCooldowns = map[uint16]time.Time{}
		sh.mu.Lock()
		sh.entities.Mobs[0].CurrentHP = 100
		sh.mu.Unlock()
		s.handleUseSkill(packet.SkillUse, proto.UseSkillPacket(1, 64001, false))
		if s.cast == nil {
			t.Fatal("valid cast did not start", attempt)
		}
		for update := 0; update < 5; update++ {
			// GX sends these while standing: stop/state refresh and vertical settling.
			s.handleCharacterMove(proto.MovePacket(0, uint16(update*100), s.char.PosX, s.char.PosY+.01, s.char.PosZ))
			if s.cast == nil {
				t.Fatalf("idle update %d lost stationary cast %d", update, attempt)
			}
		}
		beforeMP := s.char.CurrentMP
		s.cast.readyAt = time.Now().Add(-time.Second)
		s.processPendingCast()
		// Misses are valid outcomes of the original hit roll. MP and the
		// release marker distinguish execution from a discarded preparation.
		if s.char.CurrentMP >= beforeMP || s.lastSkillID != 71 || s.cast != nil {
			t.Fatal("stationary Blizzard did not release", attempt, beforeMP, s.char.CurrentMP, s.lastSkillID)
		}
	}
}

func TestIdleCastUpdatesCannotHideHorizontalWalking(t *testing.T) {
	s, _ := forgeRuleSession(t)
	physics, now := playerPhysicsFixture(t)
	s.char.MapID, s.char.PosX, s.char.PosY, s.char.PosZ = physics.char.MapID, physics.char.PosX, physics.char.PosY, physics.char.PosZ
	s.movement = physics.movement
	s.lastPositionSave = now
	s.movement.budgetAt = now.Add(-time.Second)
	s.cast = &pendingCast{number: 1, readyAt: now.Add(time.Minute)}
	x, z := s.char.PosX, s.char.PosZ
	s.handleCharacterMove(proto.MovePacket(0, 0, x+1, s.char.PosY, z))
	if s.cast == nil || s.char.PosX != x || s.char.PosZ != z {
		t.Fatal("idle packet moved caster or lost cast", s.char.PosX, s.char.PosZ)
	}
	s.movement.budgetAt = now.Add(-time.Second)
	s.handleCharacterMove(proto.MovePacket(1, 0, x+.1, s.char.PosY, z))
	if s.cast != nil {
		t.Fatal("real walking did not interrupt preparation")
	}
}
