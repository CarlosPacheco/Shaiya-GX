package world

import (
	"slices"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
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
		{a, []*ConnectedPlayer{leader}}, // slot 5 is the last of group 1 (slots 0-5)
		{b, []*ConnectedPlayer{c}},      // slots 6-11 are group 2
		{c, []*ConnectedPlayer{b}},
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

// The Movement Rune takes the user to a party member after its cast, dead or
// alive; a character outside the party is refused.
func TestMovementRuneGoesToAPartyMemberDeadOrAlive(t *testing.T) {
	user, _ := potentialSession(t)
	const mapID = uint16(1) // a field map: instance maps cannot be recorded or moved to
	user.char.MapID = mapID
	user.player.SetMap(mapID)
	old, had := mapConfig.Zones[mapID]
	mapConfig.Zones[mapID] = ZoneConfig{}
	t.Cleanup(func() {
		if had {
			mapConfig.Zones[mapID] = old
		} else {
			delete(mapConfig.Zones, mapID)
		}
	})
	floor := func() *mapgeometry.Geometry {
		g := &mapgeometry.Geometry{}
		g.Reindex()
		h := float64(float32(10) - mapgeometry.FootSkin)
		g.Add(collision.Triangle{{0, h, 0}, {0, h, 1000}, {1000, h, 0}})
		g.Add(collision.Triangle{{1000, h, 1000}, {1000, h, 0}, {0, h, 1000}})
		return g
	}
	user.char.PosX, user.char.PosY, user.char.PosZ = 100, 10, 100
	user.movement = playerMovement{geometry: floor(), mapID: mapID}
	friend, _ := gmWireSession(t, *user.char, false)
	friend.char.ID, friend.char.Name = user.char.ID+800, "RuneFriend"
	friend.char.PosX, friend.char.PosY, friend.char.PosZ = 600, 10, 450
	friend.player = RegisterPlayer(*friend.char, nil, false, friend.faction, friend.out, friend.logger)
	t.Cleanup(friend.player.Unregister)
	stranger, _ := gmWireSession(t, *user.char, false)
	stranger.char.ID, stranger.char.Name = user.char.ID+801, "RuneStranger"
	stranger.player = RegisterPlayer(*stranger.char, nil, false, stranger.faction, stranger.out, stranger.logger)
	t.Cleanup(stranger.player.Unregister)
	pt := &party{members: []*ConnectedPlayer{user.player, friend.player}}
	user.player.party, friend.player.party = pt, pt

	if p, _ := user.movementRuneTarget(stranger.char.ID); p != nil {
		t.Fatal("the rune accepted a character outside the party")
	}
	friend.player.SetAlive(false) // a dead member is still a destination
	if p, why := user.movementRuneTarget(friend.char.ID); p == nil {
		t.Fatalf("the party member was refused: %s", why)
	}
	user.moveRune = &movementRuneCast{target: friend.char.ID, ends: time.Now().Add(-time.Second)}
	user.tickMovementRune(time.Now())
	if user.moveRune != nil || user.char.PosX < 599 || user.char.PosX > 601 || user.char.PosZ < 449 || user.char.PosZ > 451 {
		t.Fatalf("the rune left the user at %v,%v, want about 600,450", user.char.PosX, user.char.PosZ)
	}
}

// Group boundaries follow CParty::GetGroup 0x445980: six slots per group.
func TestRaidGroupBoundaryIsSix(t *testing.T) {
	leader, a, b := &ConnectedPlayer{}, &ConnectedPlayer{}, &ConnectedPlayer{}
	pt := &party{kind: raidKindRaid, members: []*ConnectedPlayer{leader, a, b}, slots: []uint16{0, 5, 6}}
	for _, p := range pt.members {
		p.party = pt
	}
	if _, got := partySummonTargets(a); !slices.Equal(got, []*ConnectedPlayer{leader}) {
		t.Fatalf("slot 5 must stay with group 1: %v", got)
	}
}
