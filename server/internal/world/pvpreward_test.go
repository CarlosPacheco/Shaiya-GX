package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/pvprank"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestPvPRewardStagesCannotBeReplayedOrSkipped(t *testing.T) {
	for mode := uint8(0); mode <= 3; mode++ {
		ch := db.Character{Mode: mode, Level: 16}
		for stage := uint8(1); stage <= pvprank.RewardCount; stage++ {
			count, points, gold, _ := pvprank.Reward(stage)
			ch.Kills, ch.Deaths = count-1, count-1
			if _, _, ok := claimPvPReward(ch, false); ok {
				t.Fatal("kill reward before threshold")
			}
			if _, _, ok := claimPvPReward(ch, true); ok {
				t.Fatal("death reward before threshold")
			}
			ch.Kills, ch.Deaths = count, count
			before := ch.StatPoint
			next, value, ok := claimPvPReward(ch, false)
			if !ok || value != uint32(points) || next.KillLevel != stage || next.StatPoint != before+points || next.DeadLevel != ch.DeadLevel {
				t.Fatalf("kill stage %d mode %d: %+v value=%d", stage, mode, next, value)
			}
			ch = next
			if _, _, ok := claimPvPReward(ch, false); ok {
				t.Fatal("kill stage replay")
			}
			next, value, ok = claimPvPReward(ch, true)
			if !ok || value != gold || next.DeadLevel != stage || next.Money != ch.Money+gold || next.KillLevel != ch.KillLevel {
				t.Fatal("death reward mismatch")
			}
			ch = next
			if _, _, ok := claimPvPReward(ch, true); ok {
				t.Fatal("death stage replay")
			}
		}
		if ch.StatPoint != 734 {
			t.Fatal("redeemed stat total")
		}
		ch.StatPoint -= 4
		ch.Str, ch.Dex, ch.Rec, ch.Int, ch.Wis, ch.Luc = startingStats(ch.Race, ch.Class)
		ch.Str += 19 // fifteen automatic level points and four allocated rewards
		for range 2 {
			resetStats(&ch)
			if ch.StatPoint != 734 || ch.KillLevel != pvprank.RewardCount || ch.DeadLevel != pvprank.RewardCount || ch.Kills != pvprank.CounterCap {
				t.Fatal("reset lost points or claims", ch)
			}
			if _, _, ok := claimPvPReward(ch, false); ok {
				t.Fatal("reset allowed a previously claimed reward")
			}
		}
	}
	for _, ch := range []db.Character{{Mode: 0, Level: 15, Kills: 200000}, {Mode: 3, Level: 15, Kills: 200000}} {
		if _, _, ok := claimPvPReward(ch, false); ok {
			t.Fatal("ineligible character received reward")
		}
	}
	ch := db.Character{Mode: 1, Level: 16, Deaths: 1, Money: ^uint32(0) - 10}
	if next, _, ok := claimPvPReward(ch, true); !ok || next.Money != ^uint32(0) {
		t.Fatal("death reward gold overflow")
	}
}
