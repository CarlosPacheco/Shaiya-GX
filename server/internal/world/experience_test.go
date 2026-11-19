package world

import (
	"testing"
	"time"

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
	p.QueueExp(200)
	s.drainRewards(p)
	if _, xp, _ := p.peekRewardTotals(0); xp != 0 {
		t.Fatal("capped reward was not discarded")
	}
	select {
	case packet := <-packets:
		t.Fatalf("capped reward emitted packet %x", packet)
	case <-time.After(25 * time.Millisecond):
	}
}
