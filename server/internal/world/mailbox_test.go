package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"net"
	"testing"
)

func TestRewardsHaveBoundedMemoryAndPreserveRounding(t *testing.T) {
	for mode := range 4 {
		p := &ConnectedPlayer{}
		for range 10000 {
			p.QueueExp(1)
			p.QueueGold(1)
		}
		exp, gold := p.takeRewardTotals(mode)
		if exp != 10000*experienceScaled(1, mode) || gold != 10000 {
			t.Fatal(mode, exp, gold)
		}
		if e, g := p.takeRewardTotals(mode); e != 0 || g != 0 {
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
