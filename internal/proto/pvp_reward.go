package proto

import "fmt"

const (
	PvPRewardStatus Opcode = 0x0218
	PvPRewardKill   Opcode = 0x0219
	PvPRewardDeath  Opcode = 0x021A
)

type PvPRewards struct {
	KillStage, DeathStage uint8
	Known                 bool
}

type PvPRewardResult struct {
	OK    bool
	Value uint32
}

func DecodePvPRewards(p []byte) (PvPRewards, error) {
	r := NewReader(p)
	v := PvPRewards{KillStage: r.U8(), DeathStage: r.U8(), Known: true}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 || v.KillStage < 1 || v.KillStage > 17 || v.DeathStage < 1 || v.DeathStage > 17 {
		return v, fmt.Errorf("invalid PvP reward stages")
	}
	return v, nil
}

// Native readers 0x00595d70/0x00595dc0 consume result+u8 points or result+u32 gold.
func DecodePvPRewardResult(p []byte, death bool) (PvPRewardResult, error) {
	r := NewReader(p)
	result := r.U8()
	v := PvPRewardResult{OK: result == 1}
	if death {
		v.Value = r.U32()
	} else {
		v.Value = uint32(r.U8())
	}
	if err := r.Err(); err != nil {
		return v, err
	}
	if result > 1 || r.Remaining() != 0 {
		return v, fmt.Errorf("invalid PvP reward result")
	}
	return v, nil
}
