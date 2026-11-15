//go:build integration

package app

import (
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestPvPRewardsAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PVP_REWARDS") != "1" {
		t.Skip("requires disposable PvP reward fixture")
	}
	st, _, send, wait := testworld.Enter(t)
	wait(func() bool { return st.PvPCountsKnown == [4]bool{true, true, true, true} })
	if st.Details.Kills != 300 || st.Details.Deaths != 300 {
		t.Fatal("fixture needs 300 kills/deaths")
	}
	points, money := st.Details.StatPoint, st.Details.Money
	query := func(kill, death uint8) {
		t.Helper()
		st.PvPRewards.Known = false
		send(proto.NewWriter(proto.PvPRewardStatus).Bytes())
		wait(func() bool { return st.PvPRewards.Known })
		if st.PvPRewards.KillStage != kill || st.PvPRewards.DeathStage != death {
			t.Fatal("unexpected server claim stages", st.PvPRewards)
		}
	}
	query(1, 1)
	for i, want := range []uint32{1, 3, 5} {
		st.LastPvPReward = proto.PvPRewardResult{}
		send(proto.NewWriter(proto.PvPRewardKill).Bytes())
		wait(func() bool { return st.LastPvPReward.OK })
		if st.LastPvPReward.Value != want || st.PvPRewards.KillStage != uint8(i+2) {
			t.Fatal("hard-mode native stat reward", st.LastPvPReward)
		}
	}
	for _, want := range []uint32{100, 200, 500} {
		st.LastPvPReward = proto.PvPRewardResult{}
		send(proto.NewWriter(proto.PvPRewardDeath).Bytes())
		wait(func() bool { return st.LastPvPReward.OK })
		if st.LastPvPReward.Value != want {
			t.Fatal("native death benefit", st.LastPvPReward)
		}
	}
	if st.Details.StatPoint != points+9 || st.Details.Money != money+800 {
		t.Fatal("confirmed reward totals", st.Details.StatPoint, st.Details.Money)
	}
	for _, op := range []proto.Opcode{proto.PvPRewardKill, proto.PvPRewardDeath} {
		st.LastPvPReward = proto.PvPRewardResult{OK: true, Value: 123456}
		send(proto.NewWriter(op).Bytes())
		wait(func() bool { return !st.LastPvPReward.OK })
		if st.LastPvPReward.Value != 0 || st.Details.StatPoint != points+9 || st.Details.Money != money+800 {
			t.Fatal("unearned/replayed claim changed state")
		}
	}
	query(4, 4)
	// A real native reset stone clears only kill claims, preserving earned
	// counters and death benefits; the original requires reclaiming PvP points.
	if item, ok := st.Inventory[[2]uint8{1, 23}]; !ok || item.Type != 100 || item.TypeID != 48 {
		t.Fatal("fixture reset stone missing")
	}
	send(proto.UseItemPacket(1, 23))
	wait(func() bool { return st.Details.StatPoint == 39*7 && st.PvPRewards.KillStage == 1 })
	query(1, 4)
	for range 3 {
		st.LastPvPReward = proto.PvPRewardResult{}
		send(proto.NewWriter(proto.PvPRewardKill).Bytes())
		wait(func() bool { return st.LastPvPReward.OK })
	}
	points = 39 * 7
	if st.Details.StatPoint != points+9 || st.Details.Money != money+800 || st.Details.Kills != 300 || st.Details.Deaths != 300 {
		t.Fatal("reset/reclaim changed earned counters or death benefits")
	}
	query(4, 4)
	id := st.SelfID
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	send(proto.SelectCharacterPacket(id))
	wait(func() bool { return st.Phase == game.PhaseInWorld && st.InventoryReady })
	if st.Details.StatPoint != points+9 || st.Details.Money != money+800 {
		t.Fatal("PvP rewards did not persist on re-entry")
	}
	query(4, 4)
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	t.Log("encrypted native 0218/0219/021A: separate stages, no replay, real reset stone/reclaim, preserved death benefits, persistence and re-entry")
}
