package proto

import "fmt"

// Native target-effect readers: 0x5667a0, rendered by 0x50b6a0.
const (
	TargetEffects      Opcode = 0x030B
	TargetEffectAdd    Opcode = 0x030C
	TargetEffectRemove Opcode = 0x030D
	// GX requests a selected target's native snapshot; no native request is claimed.
	GXTargetEffectsQuery Opcode = 0x7F0F
)
const (
	TargetEffectUser uint8 = 1
	TargetEffectMob  uint8 = 2
)

type TargetEffect struct {
	Skill        uint16
	Level        uint8
	Remaining    uint32
	HasRemaining bool
}
type TargetEffectSnapshot struct {
	Kind    uint8
	ID      uint32
	Effects []TargetEffect
}

func TargetEffectsQuery(kind uint8, id uint32) []byte {
	w := NewWriter(GXTargetEffectsQuery)
	w.U8(kind)
	w.U32(id)
	return w.Bytes()
}
func DecodeTargetEffects(p []byte) (v TargetEffectSnapshot, err error) {
	r := NewReader(p)
	v.Kind, v.ID = r.U8(), r.U32()
	n := int(r.U8())
	if v.Kind != 1 && v.Kind != 2 || r.Remaining() != n*7 {
		return v, fmt.Errorf("invalid target effects snapshot")
	}
	v.Effects = make([]TargetEffect, n)
	for i := range v.Effects {
		v.Effects[i] = TargetEffect{Skill: r.U16(), Level: r.U8(), Remaining: r.U32(), HasRemaining: true}
	}
	return v, r.Err()
}
func DecodeTargetEffectChange(p []byte) (v TargetEffectSnapshot, e TargetEffect, err error) {
	if len(p) != 10 {
		return v, e, fmt.Errorf("invalid target effect change")
	}
	r := NewReader(p)
	v.Kind, v.ID = r.U8(), r.U32()
	e.Skill, e.Level = r.U16(), r.U8()
	if v.Kind != 1 && v.Kind != 2 {
		return v, e, fmt.Errorf("invalid target effect kind")
	}
	return v, e, r.Err()
}
