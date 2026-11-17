package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestTeleportStoneCastInterruptsOnDamage(t *testing.T) {
	s, give := forgeRuleSession(t)
	stone := give(db.Item{Bag: 1, Slot: 23, Type: 100, TypeID: 65, Count: 2})
	s.teleportStone = stone
	for _, tc := range []struct {
		name string
		hit  func(uint32)
	}{
		{"PvE", func(n uint32) { s.takeMobDamage(pendingDamage{MobGlobalID: 42}, n, nil) }},
		{"PvP", func(n uint32) { s.takeUserDamage(42, n) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s.teleportCast = &teleportStoneCast{ends: time.Now().Add(teleportStoneCastDuration)}
			tc.hit(0)
			if s.teleportCast == nil {
				t.Fatal("a hit with zero damage interrupted the cast")
			}
			tc.hit(1)
			if s.teleportCast != nil {
				t.Fatal("damage left the stone casting")
			}
			s.tickTeleportStone(time.Now().Add(teleportStoneCastDuration))
			i := findItem(s.items, stone.Bag, stone.Slot)
			if i < 0 || s.items[i].Count != stone.Count {
				t.Fatal("interrupted cast consumed a stone")
			}
		})
	}
}
