package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestExperienceUsesSharedCapAndCreditsOnlyRemainingXP(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	db.SetDefsForTest(&db.Defs{Exp: map[uint16]uint32{1: 200, 2: 600, 3: 600}, LevelCap: 3})
	for mode := uint8(0); mode <= 3; mode++ {
		ch := db.Character{Level: 1, Mode: mode, Exp: 100}
		if levels, xp := applyExpGain(&ch, 1000); levels != 2 || xp != 500 || ch.Level != 3 || ch.Exp != 600 {
			t.Fatalf("mode %d: levels=%d xp=%d char level=%d exp=%d", mode, levels, xp, ch.Level, ch.Exp)
		}
		if levels, xp := applyExpGain(&ch, 900); levels != 0 || xp != 0 || ch.Exp != 600 {
			t.Fatalf("mode %d awarded exp at cap", mode)
		}
	}
	s, packets := gmWireSession(t, db.Character{Level: 3, Exp: 600}, false)
	p := &ConnectedPlayer{}
	p.QueueExp(200, false)
	s.drainRewards(p)
	if snapshot := p.peekRewardTotals(); snapshot.experience.scaled(0) != 0 || len(snapshot.perfect) != 0 {
		t.Fatal("capped reward was not discarded")
	}
	select {
	case packet := <-packets:
		t.Fatalf("capped reward emitted packet %x", packet)
	case <-time.After(25 * time.Millisecond):
	}
}

func experienceMember(id uint32, mapID, level uint16, x float32) *ConnectedPlayer {
	p := &ConnectedPlayer{CharacterID: id, mapID: mapID, character: db.Character{ID: id, MapID: mapID, Level: level, PosX: x}}
	p.SetAlive(true)
	return p
}

func TestPartyExperienceRangeTruncationAndRaidFloor(t *testing.T) {
	credit := experienceMember(1, 42, 20, 1000)
	near := experienceMember(2, 42, 12, 1100)
	far := experienceMember(3, 42, 20, 1101)
	if got, perfect := killRewards(credit, 42, 20, 1000); perfect || len(got) != 1 || got[0].exp != 1440 {
		t.Fatalf("solo reward: %v, perfect=%t", got, perfect)
	}
	pt := &party{members: []*ConnectedPlayer{credit, near, far}}
	credit.party = pt
	pt.refreshLevel()
	// 64 * .73 is truncated to 46 before the .8 level share is applied.
	if got, perfect := killRewards(credit, 42, 20, 23); perfect || len(got) != 2 || got[1] != (killReward{near, 36}) {
		t.Fatalf("party range or truncation: %v, perfect=%t", got, perfect)
	}
	near.SetAlive(false)
	if got, _ := killRewards(credit, 42, 20, 23); len(got) != 1 || got[0].member != credit {
		t.Fatalf("dead or far member got a reward: %v", got)
	}
	near.SetAlive(true)
	near.character.MapID, near.mapID = 43, 43
	for len(pt.members) < partyMax {
		pt.members = append(pt.members, experienceMember(uint32(len(pt.members)+1), 42, 20, 1200))
	}
	if got, perfect := killRewards(credit, 42, 20, 1000); !perfect || len(got) != 1 || got[0].exp != 468 {
		t.Fatalf("seven-member normal party: %v, perfect=%t", got, perfect)
	}
	pt.kind = raidKindRaid
	for len(pt.members) < raidMax {
		pt.members = append(pt.members, experienceMember(uint32(len(pt.members)+1), 42, 20, 1200))
	}
	members := pt.members
	for _, size := range []int{2, 7, 18, 30, raidMax} {
		pt.members = members[:size]
		if got, perfect := killRewards(credit, 42, 20, 1000); perfect || len(got) != 1 || got[0].exp != 12 {
			t.Fatalf("raid of %d: %v, perfect=%t", size, got, perfect)
		}
	}
}

func TestPerfectPartyAwardThresholds(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	db.SetDefsForTest(&db.Defs{Exp: map[uint16]uint32{1: 200, 2: 600, 3: 600}, LevelCap: 3})
	for _, tc := range []struct {
		start, amount uint32
		level         uint16
		perfect       bool
		want          experienceCredit
		newLevel      uint16
	}{
		{100, 40, 1, false, experienceCredit{40, 0}, 1},
		{100, 40, 1, true, experienceCredit{40, 40}, 1},
		{150, 40, 1, true, experienceCredit{40, 40}, 2},
		{160, 40, 1, true, experienceCredit{40, 0}, 2},
		{550, 40, 2, true, experienceCredit{40, 10}, 3},
		{590, 40, 2, true, experienceCredit{10, 0}, 3},
		{600, 40, 3, true, experienceCredit{}, 3},
	} {
		ch := db.Character{Level: tc.level, Exp: tc.start}
		if got := applyExperienceAward(&ch, tc.amount, tc.perfect); got != tc.want || ch.Level != tc.newLevel || ch.Exp != tc.start+got.base+got.perfect {
			t.Fatalf("%+v: award %+v, resulting character %+v", tc, got, ch)
		}
	}
}

func TestRaidLevelRefreshesExperienceAndSendsNativeNotice(t *testing.T) {
	s, _ := gmWireSession(t, db.Character{ID: 1, MapID: 42, Level: 21}, false)
	peer, packets := gmWireSession(t, db.Character{ID: 2, MapID: 42, Level: 13}, false)
	s.player, peer.player = experienceMember(1, 42, 21, 0), experienceMember(2, 42, 13, 0)
	s.player.out, peer.player.out = s.out, peer.out
	pt := &party{kind: raidKindRaid, members: []*ConnectedPlayer{s.player, peer.player}, maxLevel: 20}
	s.player.party = pt
	s.partyLevel()
	if pt.maxLevel != 21 {
		t.Fatalf("raid maximum stayed at level %d", pt.maxLevel)
	}
	select {
	case p := <-packets:
		r := proto.NewReader(p)
		if proto.Op(p) != proto.RaidLevel || r.U32() != 1 || r.U16() != 21 || r.Err() != nil {
			t.Fatalf("invalid raid level notice %x", p)
		}
	case <-time.After(time.Second):
		t.Fatal("no raid level notice")
	}
}

func TestDifferentExperienceBuffsAddTheirAuthoredPercentages(t *testing.T) {
	s := &Session{buffs: []ActiveBuff{
		{SkillID: 222, Abilities: [3]buffAbility{{35, 100}}},
		{SkillID: 250, Abilities: [3]buffAbility{{35, 50}}},
		{SkillID: 251, Abilities: [3]buffAbility{{35, 50}}},
	}}
	if got := experienceScaled(101, s.experienceBonus()); got != 303 {
		t.Fatalf("101 base EXP with 200%% bonus yielded %d", got)
	}
	if got := experienceScaled(100, 300); got != 400 {
		t.Fatalf("100 base EXP with 300%% bonus yielded %d", got)
	}
	if got := experienceScaled(^uint32(0), ^uint32(0)); got != ^uint32(0) {
		t.Fatalf("EXP overflow: %d", got)
	}
}
