package world

import (
	"slices"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestPartySummonRaidScope(t *testing.T) {
	leader, a, b, c := &ConnectedPlayer{}, &ConnectedPlayer{}, &ConnectedPlayer{}, &ConnectedPlayer{}
	pt := &party{
		members: []*ConnectedPlayer{leader, a, b, c},
		kind:    raidKindRaid, slots: []uint16{0, 5, 6, 10},
	}
	for _, p := range pt.members {
		p.party = pt
	}
	for _, tc := range []struct {
		caster *ConnectedPlayer
		want   []*ConnectedPlayer
	}{
		{leader, []*ConnectedPlayer{a, b, c}},
		{a, []*ConnectedPlayer{b}},
		{c, nil},
	} {
		if group, got := partySummonTargets(tc.caster); group != pt || !slices.Equal(got, tc.want) {
			t.Fatalf("summon scope: got %v want %v", got, tc.want)
		}
	}
	pt.kind = raidKindNormal
	if _, got := partySummonTargets(a); !slices.Equal(got, []*ConnectedPlayer{leader, b, c}) {
		t.Fatal("normal party is filtered by raid slot")
	}
}

func TestPartySummonDeclineAndExpiry(t *testing.T) {
	p := &ConnectedPlayer{}
	s := Session{player: p, char: &db.Character{ID: 9, CurrentHP: 100}}
	p.summonOffer = &partySummonOffer{id: 7, expires: time.Now().Add(time.Minute)}
	s.handleSummonRune(proto.SummonAnswerPacket(6, true))
	if p.summonOffer == nil {
		t.Fatal("stale reply discarded the current offer")
	}
	s.handleSummonRune(proto.SummonAnswerPacket(7, false))
	if p.summonOffer != nil {
		t.Fatal("declined offer is still pending")
	}
	p.summonOffer = &partySummonOffer{id: 8, expires: time.Now().Add(-time.Second)}
	s.handleSummonRune(proto.SummonAnswerPacket(8, true))
	if p.summonOffer != nil || s.char.MapID != 0 {
		t.Fatal("expired offer permitted movement")
	}
}

func TestPartySummonInterruptsOnDamage(t *testing.T) {
	s, give := forgeRuleSession(t)
	left := give(db.Item{Bag: 1, Slot: 23, Type: 100, TypeID: 45, Count: 1})
	s.summonCastEnd = time.Now().Add(summonCastTime)
	s.takeMobDamage(pendingDamage{MobGlobalID: 42}, 0, nil)
	if s.summonCastEnd.IsZero() {
		t.Fatal("zero damage interrupted the summon")
	}
	s.takeMobDamage(pendingDamage{MobGlobalID: 42}, 1, nil)
	if !s.summonCastEnd.IsZero() {
		t.Fatal("damage did not interrupt the summon")
	}
	s.tickPartySummon(time.Now().Add(summonCastTime))
	if i := findItem(s.items, left.Bag, left.Slot); i < 0 || s.items[i].Count != 1 {
		t.Fatal("interrupting the cast changed the already charged inventory")
	}
}
