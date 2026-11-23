package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"testing"
	"time"
)

func TestMissAcquiresIdleReactiveMobWithoutDamage(t *testing.T) {
	_, h := luaWorldFixture(t, 0)
	recordMobAIMiss(h.mapID, h.mob.ID, 99)
	if h.mob.AggroTarget != 99 || h.mob.CurrentHP != 1000 || !h.mob.lua.attacked {
		t.Fatal("miss did not enter combat without damage")
	}
	eventOK(t, h, "idle")
	eventOK(t, h, "combat", 100, 0)
	if len(h.tick.Attacks) != 1 {
		t.Fatal("reactive mob did not retaliate after miss")
	}
}

func TestBasicMissWakesMobWithoutAddingAggro(t *testing.T) {
	s, defs := lapisSession(t)
	id, md := lapisMob(t, s, defs)
	s.char.PosX, s.char.PosZ = 1, 1
	md.Dex = 65535
	if _, ok := ApplyMobEffect(s.char.MapID, id, s.char.ID, defs.NPCSkill(241)); !ok {
		t.Fatal("failed to put the mob to sleep")
	}
	combat := mobCombatFor(md)
	stats := s.stats()
	if chance := playerHitChance(&stats, &combat, attackMelee); chance != 0 {
		t.Fatalf("expected a guaranteed miss, got chance %d", chance)
	}
	if !s.autoAttackSwing(id) {
		t.Fatal("the attack on the sleeping mob was rejected")
	}
	mob, _ := MobByID(s.char.MapID, id)
	if mob.CurrentHP != md.MaxHP || mob.AggroTarget != 0 || len(mob.aggro) != 0 || mob.mods().Sleep != 0 {
		t.Fatalf("basic miss left HP=%d target=%d aggro=%v sleep=%d", mob.CurrentHP, mob.AggroTarget, mob.aggro, mob.mods().Sleep)
	}
}

func TestMagicSpotForcedMissReturnsPacketAndAcquiresMob(t *testing.T) {
	_, h := luaWorldFixture(t, 0)
	s, packets := gmWireSession(t, db.Character{ID: 99, Level: 40, MapID: h.mapID, PosX: 102, PosZ: 100, CurrentHP: 500}, false)
	s.player = RegisterPlayer(*s.char, nil, false, 0, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	def := &db.SkillDef{SkillID: 137, SkillLevel: 3, TypeEffect: 1, TypeDetail: 21, TargetType: 3, SuccessType: 1, SuccessValue: 0, Kind: 2}
	db.MustDefs().Skills = map[db.SkillKey]*db.SkillDef{{SkillID: 137, Level: 3}: def}
	if !s.skillHit(h.mob.ID, def, db.CharacterSkill{SkillID: 137, Level: 3}, true) {
		t.Fatal("miss removed live mob")
	}
	select {
	case p := <-packets:
		r, err := proto.DecodeSkillResult(p)
		if err != nil || proto.Op(p) != proto.SkillUse || r.Result != 3 || r.Target != h.mob.ID || r.ID != 137 {
			t.Fatalf("missing Magic Spot miss packet: %+v %v", r, err)
		}
	case <-time.After(time.Second):
		t.Fatal("miss produced no packet")
	}
	if h.mob.AggroTarget != 99 || h.mob.CurrentHP != 1000 {
		t.Fatal("miss lost aggro or inflicted damage")
	}
}
func TestRejectedSkillCooldownReturnsNativeNotice(t *testing.T) {
	s, _, _ := spellCastRangeFixture(t)
	// The fixture creates a valid learned skill; only its cooldown refuses use.
	s.skillCooldowns[64001] = time.Now().Add(time.Minute)
	// Use a dedicated wire to inspect the rejection without registering viewers.
	wire, packets := gmWireSession(t, *s.char, false)
	wire.skills = s.skills
	wire.skillCooldowns = s.skillCooldowns
	wire.handleUseSkill(packet.SkillUse, proto.UseSkillPacket(1, 64001, false))
	select {
	case p := <-packets:
		v, err := proto.DecodeSkillFailure(p)
		if err != nil || proto.Op(p) != proto.GXSkillFailure || v.Message != 215 {
			t.Fatalf("missing cooldown notice: %+v %v", v, err)
		}
	case <-time.After(time.Second):
		t.Fatal("silent cooldown rejection")
	}
}
