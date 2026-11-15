package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/pvprank"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestPvPRewardStagesCannotBeReplayedOrSkipped(t *testing.T) {
	for mode := uint8(1); mode <= 3; mode++ {
		ch := db.Character{Mode: mode, Level: 16}
		for stage := uint8(1); stage <= pvprank.RewardCount; stage++ {
			count, points, gold, _ := pvprank.Reward(mode, stage)
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
		if ch.StatPoint != [4]uint16{0, 208, 262, 299}[mode] {
			t.Fatal("redeemed stat total")
		}
		for range 2 {
			resetStats(&ch)
			base := 15 * [4]uint16{0, 5, 7, 9}[mode]
			if ch.StatPoint != base || ch.KillLevel != 0 || ch.DeadLevel != 16 || ch.Kills != 200000 {
				t.Fatal("native reset must clear only kill claims and retain earned counters/death claims", ch)
			}
			for range 16 {
				next, _, ok := claimPvPReward(ch, false)
				if !ok {
					t.Fatal("reset did not permit earned PvP rewards to be redeemed again")
				}
				ch = next
			}
			if ch.StatPoint != base+[4]uint16{0, 208, 262, 299}[mode] {
				t.Fatal("reclaim after reset duplicated or lost points")
			}
		}
	}
	for _, ch := range []db.Character{{Mode: 0, Level: 60, Kills: 200000}, {Mode: 1, Level: 15, Kills: 200000}, {Mode: 255, Level: 60, Kills: 200000}} {
		if _, _, ok := claimPvPReward(ch, false); ok {
			t.Fatal("ineligible character received reward")
		}
	}
	ch := db.Character{Mode: 1, Level: 16, Deaths: 1, Money: ^uint32(0) - 10}
	if next, _, ok := claimPvPReward(ch, true); !ok || next.Money != ^uint32(0) {
		t.Fatal("death reward gold overflow")
	}
}
