//go:build integration

package world

import (
	"fmt"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestPerfectPartyKillAndQuestExperiencePersistAndReachClient(t *testing.T) {
	fixture, _ := forgeRuleSession(t)
	defs := db.MustDefs()
	defs.Exp, defs.LevelCap = map[uint16]uint32{1: 10000, 2: 50000}, 2
	s, wire := gmWireSession(t, *fixture.char, false)
	s.player = experienceMember(s.char.ID, s.char.MapID, s.char.Level, 100)
	s.player.out = s.out
	pt := &party{members: []*ConnectedPlayer{s.player}}
	for index := 1; index < partyMax; index++ {
		member := experienceMember(uint32(index), s.char.MapID, s.char.Level, 10)
		member.out = s.out
		pt.members = append(pt.members, member)
	}
	s.player.party = pt
	pt.refreshLevel()
	mob := &MapMob{MobID: 1, Level: s.char.Level, Exp: 1000, PosX: 180}
	rewardKill(s.player, s.char.ID, s.char.MapID, mob, s.log)
	// This member is within 100 units of the credited player and 170 from the mob.
	if pending := pt.members[1].peekRewardTotals(); len(pending.perfect) != 1 || pending.perfect[0].amount != 468 {
		t.Fatalf("valid party member missed the mob reward: %+v", pending)
	}
	pool, err := db.Pool(db.GameData)
	if err != nil {
		t.Fatal(err)
	}
	trigger := fmt.Sprintf("CREATE TRIGGER perfect_exp_retry BEFORE UPDATE OF Exp ON PS_GameData_Chars WHEN OLD.CharID=%d BEGIN SELECT RAISE(ABORT,'injected EXP write failure'); END", s.char.ID)
	if _, err := pool.Exec(trigger); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec("DROP TRIGGER IF EXISTS perfect_exp_retry") })
	before := s.char.Exp
	s.drainRewards(s.player)
	if s.char.Exp != before || len(s.player.peekRewardTotals().perfect) != 1 || len(wire) != 0 {
		t.Fatal("failed write changed experience, lost its bonus or sent a packet")
	}
	if _, err := pool.Exec("DROP TRIGGER perfect_exp_retry"); err != nil {
		t.Fatal(err)
	}
	assertAward := func(base, bonus, total uint32) {
		t.Helper()
		deadline := time.NewTimer(time.Second)
		defer deadline.Stop()
		for {
			select {
			case p := <-wire:
				if proto.Op(p) != proto.ExpGain {
					continue
				}
				got, err := proto.DecodeExperienceGain(p)
				if err != nil || got.Base != base || got.Perfect != bonus {
					t.Fatalf("EXP wire: %+v, %v, want %d + %d", got, err, base, bonus)
				}
				saved, err := db.CharacterByID(s.char.ID)
				if err != nil || s.char.Exp != total || saved.Exp != total {
					t.Fatalf("live EXP %d, saved EXP %d, want %d; error %v", s.char.Exp, saved.Exp, total, err)
				}
				return
			case <-deadline.C:
				t.Fatal("no experience notice")
			}
		}
	}
	s.drainRewards(s.player)
	assertAward(468, 468, before+936)
	s.drainRewards(s.player)
	if len(s.player.peekRewardTotals().perfect) != 0 || s.char.Exp != before+936 {
		t.Fatal("successful write replayed XP")
	}
	q := &activeQuest{ID: 1, def: &questDef{ID: 1, Results: [3]questResult{{Exp: 25}}}}
	s.quests = []*activeQuest{q}
	if err := db.AddCharacterQuest(s.char.ID, db.CharQuest{QuestID: q.ID}); err != nil {
		t.Fatal(err)
	}
	s.awardQuest(q, 0, 0)
	assertAward(25, 25, before+986)
}
