package world

import (
	"slices"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func TestRaidSkillCastScope(t *testing.T) {
	for _, tc := range []struct {
		name       string
		id         uint16
		targetType uint8
	}{
		{"raid attribute buff", 111, proto.SkillTargetRaid},
		{"raid periodic heal", 124, proto.SkillTargetRaid},
		{"raid direct heal", 102, proto.SkillTargetRaid},
		{"subgroup attribute buff", 111, targetTypeParty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster, casterWire := potentialSession(t)
			defs := db.MustDefs()
			def := *defs.Skill(tc.id, 1)
			def.TargetType, def.ApplyRange, def.ReadyTime = tc.targetType, 15, 4
			def.SPCost, def.MPCost, def.ResetTimeSeconds = 12, 25, 30
			defs.Skills[db.SkillKey{SkillID: tc.id, Level: 1}] = &def
			caster.skills = []db.CharacterSkill{{SkillID: tc.id, Level: 1, Number: 1}}
			caster.char.PosX, caster.char.CurrentHP = 100, 50
			caster.syncPlayer()

			member := func(offset uint32, x, y float32, sameMap bool) *Session {
				ch := *caster.char
				ch.ID += 800000 + offset
				ch.PosX, ch.PosY = x, y
				if !sameMap {
					ch.MapID++
				}
				s, _ := gmWireSession(t, ch, false)
				s.faction, s.items = caster.faction, caster.items
				s.player = RegisterPlayer(ch, nil, false, s.faction, s.out, s.logger)
				t.Cleanup(s.player.Unregister)
				return s
			}
			subgroup := member(1, 110, 25, true)
			otherGroup := member(2, 116, 25, true)
			outOfRange := member(3, 105, 0, true)
			otherMap := member(4, 105, 0, false)
			dead := member(5, 105, 0, true)
			dead.dead = true
			dead.player.SetAlive(false)
			outsider := member(6, 105, 0, true)
			pt := &party{kind: raidKindRaid, slots: []uint16{0, 5, 149, 6, 7, 8},
				members: []*ConnectedPlayer{subgroup.player, caster.player, otherGroup.player, outOfRange.player, otherMap.player, dead.player}}
			partyMu.Lock()
			for _, p := range pt.members {
				p.party = pt
			}
			partyMu.Unlock()
			t.Cleanup(func() {
				partyMu.Lock()
				defer partyMu.Unlock()
				for _, p := range pt.members {
					p.party = nil
				}
			})

			before := caster.stats()
			costSP, costMP := caster.skillCosts(&def, &before)
			caster.handleUseSkill(packet.SkillUseNoTarget, proto.UseSkillPacket(1, outsider.char.ID, true))
			if caster.cast == nil || caster.hasBuff(tc.id) {
				t.Fatal("party cast must prepare on the caster")
			}
			timeout := time.NewTimer(time.Second)
			defer timeout.Stop()
		prepared:
			for {
				select {
				case frame := <-casterWire:
					if proto.Op(frame) == proto.SkillCastStarted {
						cast, err := proto.DecodeSkillCast(frame)
						if err != nil || cast.ID != tc.id || cast.Target != caster.char.ID {
							t.Fatalf("wrong prepared cast target: %+v, %v", cast, err)
						}
						break prepared
					}
				case <-timeout.C:
					t.Fatal("missing prepared cast packet")
				}
			}
			otherGroup.char.PosX, outOfRange.char.PosX = 115, 116
			otherGroup.syncPlayer()
			outOfRange.syncPlayer()
			caster.cast.readyAt = time.Now().Add(-time.Second)
			caster.processPendingCast()
			if after := caster.stats(); after.CurrentSP != before.CurrentSP-costSP || after.CurrentMP != before.CurrentMP-costMP || caster.skillReady(tc.id) {
				t.Fatal("one cast must use resources and start reuse once")
			}

			assertRecipient := func(s *Session, reached bool) {
				t.Helper()
				s.tickFriendly()
				if def.TypeEffect == typeEffectBuff {
					if s.hasBuff(tc.id) != reached {
						t.Fatalf("character %d received buff %v, want %v", s.char.ID, s.hasBuff(tc.id), reached)
					}
					return
				}
				if healed := s.stats().CurrentHP > 50; healed != reached {
					t.Fatalf("character %d healed %v, want %v", s.char.ID, healed, reached)
				}
			}
			assertRecipient(caster, true)
			assertRecipient(subgroup, true)
			assertRecipient(otherGroup, tc.targetType == proto.SkillTargetRaid)
			for _, s := range []*Session{outOfRange, otherMap, dead, outsider} {
				assertRecipient(s, false)
			}

			partyMu.Lock()
			pt.kind = raidKindNormal
			partyMu.Unlock()
			targets, _, _ := caster.friendlyTargets(&def, 0)
			if !slices.Equal(targets, []*ConnectedPlayer{subgroup.player, caster.player, otherGroup.player}) {
				t.Fatal("normal party cast did not reach the in-range members")
			}
			partyMu.Lock()
			caster.player.party = nil
			partyMu.Unlock()
			if targets, _, _ := caster.friendlyTargets(&def, 0); !slices.Equal(targets, []*ConnectedPlayer{caster.player}) {
				t.Fatal("solo cast did not fall back to the caster")
			}
		})
	}
}
