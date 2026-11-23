package proto

import (
	"fmt"
	"time"
)

const (
	GXOxygen         Opcode = 0x7F26
	GXDrowningDamage Opcode = 0x7F27
	GXDrowningDeath  Opcode = 0x7F2A // target u32: the target died of drowning

	OxygenCapacity   = 30 * time.Second
	OxygenRefillRate = 10
)

type OxygenPhase uint8

const (
	OxygenHidden OxygenPhase = iota
	OxygenDraining
	OxygenRefilling
)

// OxygenStatus is a server snapshot. RemainingMS counts down underwater and
// increases at OxygenRefillRate while breathing.
type OxygenStatus struct {
	Phase       OxygenPhase
	RemainingMS uint16
}

func OxygenPacket(v OxygenStatus) []byte {
	w := NewWriter(GXOxygen)
	w.U8(uint8(v.Phase))
	w.U16(v.RemainingMS)
	return w.Bytes()
}

func DecodeOxygen(p []byte) (OxygenStatus, error) {
	r := NewReader(p)
	v := OxygenStatus{Phase: OxygenPhase(r.U8()), RemainingMS: r.U16()}
	if Op(p) != GXOxygen || r.Err() != nil || r.Remaining() != 0 ||
		v.Phase > OxygenRefilling || v.RemainingMS > uint16(OxygenCapacity.Milliseconds()) ||
		v.Phase == OxygenHidden && v.RemainingMS != 0 {
		return OxygenStatus{}, fmt.Errorf("invalid oxygen status")
	}
	return v, nil
}

func DecodeDrowningDamage(p []byte) (EnvironmentalDamage, error) {
	return decodeEnvironmentalDamage(p, "drowning")
}

// DecodeDrowningDeath reads the id of the character that drowned.
func DecodeDrowningDeath(p []byte) (uint32, error) {
	r := NewReader(p)
	id := r.U32()
	if Op(p) != GXDrowningDeath || r.Err() != nil || r.Remaining() != 0 {
		return 0, fmt.Errorf("invalid drowning death")
	}
	return id, nil
}
