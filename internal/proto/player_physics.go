package proto

import (
	"fmt"
	"math"
)

const GXPlayerCorrection Opcode = 0x7F08

type PlayerCorrection struct {
	ID             uint32
	X, Y, Z        float32
	Motion         uint8
	Direction      uint16
	Jumping        bool
	JumpFall, Fall float32
	SlopeFrames    float32
}

func DecodePlayerCorrection(p []byte) (c PlayerCorrection, err error) {
	if len(p) != 34 || p[21] > 1 {
		return c, fmt.Errorf("invalid player physics correction")
	}
	r := NewReader(p)
	c.ID = r.U32()
	c.X, c.Y, c.Z = r.F32(), r.F32(), r.F32()
	c.Motion, c.Direction, c.Jumping = r.U8(), r.U16(), r.Bool()
	c.JumpFall, c.Fall = r.F32(), r.F32()
	c.SlopeFrames = r.F32()
	if c.SlopeFrames < 0 {
		return PlayerCorrection{}, fmt.Errorf("negative slope pressure")
	}
	for _, v := range []float32{c.X, c.Y, c.Z, c.JumpFall, c.Fall, c.SlopeFrames} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return PlayerCorrection{}, fmt.Errorf("non-finite physics correction")
		}
	}
	return c, r.Err()
}
