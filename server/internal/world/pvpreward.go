package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/pvprank"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// CUser::PacketMaininterface 0x0218-0x021A requires non-easy mode and level16+.
// StatusKillResult/StatusDeathResult redeem exactly one unclaimed stage.
func (s *Session) handlePvPReward(op packet.Opcode, p []byte) {
	ch := s.char
	if len(p) != 2 || ch == nil || ch.Level < 16 {
		return
	}
	if op == packet.Opcode(proto.PvPRewardStatus) {
		s.send(buildPvPRewardStatus(ch), "0x0218 PvP reward stages")
		return
	}
	death := op == packet.Opcode(proto.PvPRewardDeath)
	next, value, ok := claimPvPReward(*ch, death)
	if ok {
		if err := db.SaveCharacter(&next); err != nil {
			s.log.Printf("GX Server World PvP reward save failed character_id=%d: %v", ch.ID, err)
			ok, value = false, 0
		} else {
			*ch = next
			s.syncPlayer()
		}
	}
	w := packet.New(op)
	if ok {
		w.U8(1)
	} else {
		w.U8(0)
	}
	if death {
		w.U32(value)
	} else {
		w.U8(uint8(value))
	}
	s.send(w.MustFinish(), "PvP reward result")
}

func buildPvPRewardStatus(ch *db.Character) []byte {
	w := packet.New(packet.Opcode(proto.PvPRewardStatus))
	w.U8(min(ch.KillLevel, pvprank.RewardCount) + 1)
	w.U8(min(ch.DeadLevel, pvprank.RewardCount) + 1)
	return w.MustFinish()
}

func claimPvPReward(ch db.Character, death bool) (db.Character, uint32, bool) {
	if ch.Level < 16 {
		return ch, 0, false
	}
	stage, count := ch.KillLevel+1, ch.Kills
	if death {
		stage, count = ch.DeadLevel+1, ch.Deaths
	}
	// Reject persisted out-of-range claim levels before the byte increment wraps.
	if (!death && ch.KillLevel >= pvprank.RewardCount) || (death && ch.DeadLevel >= pvprank.RewardCount) {
		return ch, 0, false
	}
	threshold, points, money, ok := pvprank.Reward(stage)
	if !ok || count < threshold {
		return ch, 0, false
	}
	if death {
		ch.DeadLevel = stage
		ch.Money = uint32(min(uint64(ch.Money)+uint64(money), uint64(^uint32(0))))
		return ch, money, true
	}
	ch.KillLevel = stage
	ch.StatPoint += points // Native stat pool is a u16.
	return ch, uint32(points), true
}
