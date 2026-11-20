package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func TestSoulBurnLearnsForPriestsAndOracles(t *testing.T) {
	for name, race := range map[string]uint8{"Priest": 0, "Oracle": 2} {
		t.Run(name, func(t *testing.T) {
			s, _ := potentialSession(t)
			s.char.Level, s.char.Race, s.char.SkillPoint, s.char.Mode = 22, race, 5, growNormal
			if err := db.SaveCharacter(s.char); err != nil {
				t.Fatal(err)
			}
			s.char.Class = 4
			s.handleLearnSkill(proto.LearnSkillPacket(125, 1))
			if len(s.skills) != 0 {
				t.Fatal("a mage learned Soul Burn")
			}
			s.char.Class = 5
			s.handleLearnSkill(proto.LearnSkillPacket(125, 1))
			def := db.MustDefs().Skill(125, 1)
			if len(s.skills) != 1 || s.skills[0].SkillID != 125 || s.char.SkillPoint != 2 || def == nil ||
				def.TypeEffect != typeEffectOnce || def.SuccessType != 1 || def.SuccessValue != 100 || def.KeepSeconds != 0 {
				t.Fatalf("Soul Burn catalog or learning is incorrect for race %d: %+v, skills %+v", race, def, s.skills)
			}
			s.handleLearnSkill(proto.LearnSkillPacket(125, 2))
			if s.skills[0].Level != 1 {
				t.Fatal("legacy placeholder rank is still learnable")
			}
		})
	}
}

func TestSoulBurnRespawnsTheSameEnemyCorpseAfterItsImpact(t *testing.T) {
	caster, _ := potentialSession(t)
	const mapID = 60125
	RegisterMapGrid(mapID, 1024, nil)
	AddEntities(mapID, MapEntities{SpawnAreas: []MapSpawnArea{{Faction: 2, Boxes: []SpawnBox{{300, 10, 300, 310, 12, 310}}}}})
	caster.player.SetMap(mapID)
	caster.char.MapID, caster.char.PosX, caster.char.PosZ = mapID, 500, 500
	caster.syncPlayer()
	caster.skills = []db.CharacterSkill{{SkillID: 125, Level: 1, Number: 1}}
	ch := *caster.char
	ch.ID, ch.PosX, ch.PosY, ch.Mode = ch.ID+8125, 510, 11, growNormal
	victim, wire := gmWireSession(t, ch, false)
	victim.faction = FactionDark
	victim.player = RegisterPlayer(ch, nil, false, victim.faction, victim.out, victim.logger)
	t.Cleanup(victim.player.Unregister)
	cast := func() {
		caster.lastActionAt = time.Time{}
		caster.handleUseSkill(packet.SkillUseNoTarget, proto.UseSkillPacket(1, ch.ID, true))
	}
	finish := func() {
		t.Helper()
		if caster.cast == nil {
			t.Fatal("eligible Soul Burn cast never started")
		}
		caster.cast.readyAt = time.Now().Add(-time.Second)
		caster.processPendingCast()
	}
	cast()
	if caster.cast != nil {
		t.Fatal("Soul Burn prepared on a living player")
	}
	victim.dead, victim.char.CurrentHP = true, 0
	victim.player.SetAlive(false)
	victim.startRebirthTimer()
	victim.char.Mode = growUltimate
	victim.syncPlayer()
	cast()
	if caster.cast != nil {
		t.Fatal("Soul Burn prepared on an Ultimate corpse")
	}
	victim.char.Mode = growNormal
	victim.syncPlayer()
	caster.faction = FactionDark
	cast()
	if caster.cast != nil {
		t.Fatal("Soul Burn prepared on an allied corpse")
	}
	caster.faction = FactionLight
	before := caster.stats().CurrentMP
	cast()
	victim.player.SetAlive(true)
	victim.player.SetAlive(false) // a new death while the caster was preparing
	finish()
	victim.tickSkillOnce()
	if !victim.dead || !caster.skillReady(125) || caster.stats().CurrentMP != before {
		t.Fatal("the abandoned cast affected a later death or spent resources")
	}
	cast()
	finish()
	if !victim.dead || caster.stats().CurrentMP != before-430 || caster.skillReady(125) {
		t.Fatal("Soul Burn did not charge its authored cost and start reuse")
	}
	victim.tickSkillOnce()
	if victim.dead || !victim.player.IsAlive() || victim.char.PosX < 300 || victim.char.PosX >= 310 {
		t.Fatalf("the enemy did not return to their respawn point: dead %v, x %.2f", victim.dead, victim.char.PosX)
	}
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	impact, removed := false, false
	for {
		select {
		case frame := <-wire:
			switch proto.Op(frame) {
			case proto.SkillUseNoTarget:
				result, err := proto.DecodeSkillResult(frame)
				if err != nil || result.ID != 125 || result.Target != ch.ID || result.Result != 0 {
					t.Fatalf("wrong Soul Burn result: %+v, %v", result, err)
				}
				impact = true
			case proto.CharacterReviveInit:
				if !impact {
					t.Fatal("the corpse left before the impact notification")
				}
				removed = true
			case proto.CharacterResurrect:
				if !removed {
					t.Fatal("respawn packet appeared before corpse removal")
				}
				return
			}
		case <-timer.C:
			t.Fatal("missing the impact or respawn packets")
		}
	}
}
