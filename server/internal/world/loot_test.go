package world

import (
	"testing"
	"time"
)

func lootGroup(ids ...uint32) (*party, []*ConnectedPlayer) {
	pt := &party{id: 1}
	var members []*ConnectedPlayer
	now := time.Now()
	for _, id := range ids {
		p := &ConnectedPlayer{CharacterID: id}
		p.party = pt
		pt.members = append(pt.members, p)
		pt.lootJoin(id, now)
		members = append(members, p)
	}
	return pt, members
}

// Everyone is served once before anyone is served twice.
func TestLootRoundsServeEveryoneBeforeRepeating(t *testing.T) {
	pt, members := lootGroup(1, 2, 3)
	now := time.Now()
	for round := 0; round < 4; round++ {
		seen := map[uint32]int{}
		for range members {
			seen[pt.lootPick(members, members, now).CharacterID]++
		}
		if len(seen) != 3 {
			t.Fatalf("round %d served %v", round, seen)
		}
	}
}

// A member who joins a round in progress waits for the next one.
func TestLootLateJoinerWaitsForNextRound(t *testing.T) {
	pt, members := lootGroup(1, 2)
	now := time.Now()
	first := pt.lootPick(members, members, now)
	late := &ConnectedPlayer{CharacterID: 9}
	late.party = pt
	pt.members = append(pt.members, late)
	pt.lootJoin(9, now)
	all := append(append([]*ConnectedPlayer(nil), members...), late)
	second := pt.lootPick(all, all, now)
	if second == first || second == late {
		t.Fatalf("second item went to %d (first %d, late joiner 9)", second.CharacterID, first.CharacterID)
	}
}

// Leaving and returning cannot be used to be served again, and a short
// disconnection keeps the place for ten minutes.
func TestLootLeaveAndReturnKeepsPlace(t *testing.T) {
	pt, members := lootGroup(1, 2, 3)
	now := time.Now()
	got := pt.lootPick(members, members, now)
	pt.lootLeave(got.CharacterID, now)
	pt.lootJoin(got.CharacterID, now.Add(9*time.Minute))
	if !pt.lootEntryOf(got.CharacterID).served {
		t.Fatal("returning inside the grace period reset the member's turn")
	}
	// Someone unserved who steps out and back is still waiting for their turn.
	var waiting *ConnectedPlayer
	for _, m := range members {
		if !pt.lootEntryOf(m.CharacterID).served {
			waiting = m
		}
	}
	pt.lootLeave(waiting.CharacterID, now)
	pt.lootJoin(waiting.CharacterID, now.Add(5*time.Minute))
	if pt.lootEntryOf(waiting.CharacterID).served {
		t.Fatal("a disconnection cost the member their turn")
	}
	// After the grace period the member is new, and a round is under way.
	other := pt.lootPick(members, members, now) // someone else is served: the round has begun
	if other == got {
		t.Fatal("served the same member twice in a round")
	}
	pt.lootLeave(got.CharacterID, now)
	pt.lootJoin(got.CharacterID, now.Add(11*time.Minute))
	if !pt.lootEntryOf(got.CharacterID).served {
		t.Fatal("a returning member after the grace period must wait for the next round")
	}
}

// A member whose bag is full keeps their turn while the others are served.
func TestLootFullBagKeepsTurn(t *testing.T) {
	pt, members := lootGroup(1, 2)
	now := time.Now()
	for i := 0; i < 5; i++ {
		got := pt.lootPick(members, members[1:], now) // member 1 cannot carry
		if got.CharacterID != 2 {
			t.Fatalf("item went to %d", got.CharacterID)
		}
	}
	if pt.lootEntryOf(1).served {
		t.Fatal("the member with a full bag lost their turn")
	}
	if got := pt.lootPick(members, members, now); got.CharacterID != 1 {
		t.Fatalf("the waiting member must be next, got %d", got.CharacterID)
	}
}

func TestLootDamageLedgerSumsPerAttacker(t *testing.T) {
	var m MapMob
	m.recordLootDamage(5, 10)
	m.recordLootDamage(6, 30)
	m.recordLootDamage(5, 25)
	m.recordLootDamage(0, 99)
	if len(m.lootDamage) != 2 || m.lootDamage[0].Amount != 35 || m.lootDamage[1].Amount != 30 {
		t.Fatalf("ledger %+v", m.lootDamage)
	}
}
