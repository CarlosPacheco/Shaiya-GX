package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"net"
	"testing"
)

func TestRewardsHaveBoundedMemoryAndPreserveRounding(t *testing.T) {
	for _, bonus := range []uint32{0, 20, 50, 70, 200, 300} {
		p := &ConnectedPlayer{}
		for range 10000 {
			p.QueueExp(101, false)
			p.QueueGold(1)
		}
		exp, gold := p.takeRewardTotals(bonus)
		if exp != 10000*experienceScaled(101, bonus) || gold != 10000 {
			t.Fatal(bonus, exp, gold)
		}
		if e, g := p.takeRewardTotals(bonus); e != 0 || g != 0 {
			t.Fatal("replayed rewards")
		}
	}
	p := &ConnectedPlayer{}
	p.QueueGold(^uint32(0))
	p.QueueGold(1)
	_, gold := p.takeRewardTotals(0)
	if gold != ^uint32(0) {
		t.Fatal("reward overflow", gold)
	}
}
func TestCombatAndItemMailboxesAreBounded(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	p := &ConnectedPlayer{out: &connWriter{conn: a}}
	for range maxPendingCombat + 10 {
		p.QueueDamage(pendingDamage{})
	}
	if len(p.TakeDamage()) != maxPendingCombat {
		t.Fatal("unbounded combat queue")
	}
	for range maxPendingItemGrants {
		if !p.QueueItemGrant(db.Item{Count: 1}) {
			t.Fatal("early item refusal")
		}
	}
	if p.QueueItemGrant(db.Item{Count: 1}) {
		t.Fatal("unbounded item queue")
	}
	if len(p.TakeItemGrants()) != maxPendingItemGrants {
		t.Fatal("accepted items lost")
	}
}

func TestExperienceCommitRetainsRewardsQueuedWhileWriting(t *testing.T) {
	p := &ConnectedPlayer{}
	p.QueueExp(11, false)
	p.QueueExp(17, true)
	p.QueueExp(4, false)
	p.QueueGold(1)
	snapshot := p.peekRewardTotals()
	p.QueueExp(6, false)
	p.QueueExp(23, true)
	p.QueueExp(5, false)
	p.QueueGold(9)
	p.ackRewardTotals(snapshot)
	pending := p.peekRewardTotals()
	if len(pending.perfect) != 1 || pending.perfect[0].amount != 23 || pending.gold != 9 {
		t.Fatalf("new rewards lost after commit: %+v", pending)
	}
	for _, bonus := range []uint32{0, 20, 50, 70, 200, 300} {
		if pending.perfect[0].before.scaled(bonus) != experienceScaled(6, bonus) || pending.experience.scaled(bonus) != experienceScaled(5, bonus) {
			t.Fatalf("concurrent EXP replayed or disappeared at %d%%: %+v", bonus, pending)
		}
	}
}
