//go:build integration

package app

import (
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestPvPFinalRankAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PVP_FINAL_RANK") != "1" {
		t.Skip("requires disposable one-million-kill fixture")
	}
	st, _, send, wait := testworld.Enter(t)
	wait(func() bool { return st.PvPCountsKnown == [4]bool{true, true, true, true} })
	if st.Details.Kills != 1000000 || st.Details.Deaths != 1000000 {
		t.Fatal("fixture needs the final rank counters")
	}
	points, money := st.Details.StatPoint, st.Details.Money
	query := func(stage uint8) {
		t.Helper()
		st.PvPRewards.Known = false
		send(proto.NewWriter(proto.PvPRewardStatus).Bytes())
		wait(func() bool { return st.PvPRewards.Known })
		if st.PvPRewards.KillStage != stage || st.PvPRewards.DeathStage != stage {
			t.Fatal("wrong final-rank claim stage", st.PvPRewards)
		}
	}
	query(31)
	for _, c := range []struct {
		op     proto.Opcode
		amount uint32
	}{{proto.PvPRewardKill, 26}, {proto.PvPRewardDeath, 4000000}} {
		st.LastPvPReward = proto.PvPRewardResult{}
		send(proto.NewWriter(c.op).Bytes())
		wait(func() bool { return st.LastPvPReward.OK })
		if st.LastPvPReward.Value != c.amount {
			t.Fatal("wrong client-table final reward", st.LastPvPReward)
		}
	}
	query(32)
	for _, op := range []proto.Opcode{proto.PvPRewardKill, proto.PvPRewardDeath} {
		st.LastPvPReward = proto.PvPRewardResult{OK: true}
		send(proto.NewWriter(op).Bytes())
		wait(func() bool { return !st.LastPvPReward.OK })
	}
	if st.Details.StatPoint != points+26 || st.Details.Money != money+4000000 {
		t.Fatal("final reward duplicated or lost")
	}
	id := st.SelfID
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	send(proto.SelectCharacterPacket(id))
	wait(func() bool { return st.Phase == game.PhaseInWorld && st.InventoryReady })
	query(32)
	if st.Details.StatPoint != points+26 || st.Details.Money != money+4000000 {
		t.Fatal("final reward did not persist")
	}
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	t.Log("1,000,000 kills/deaths: rank31 rewarded once, terminal stage32 accepted, persisted after re-entry")
}
