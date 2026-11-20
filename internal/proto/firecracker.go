package proto

import (
	"fmt"
	"math"
	"strings"
)

const GXFirecracker Opcode = 0x7F24

// Firecracker is a confirmed use localized to the launch position.
type Firecracker struct {
	Variant uint8
	X, Y, Z float32
	Name    string
}

func FirecrackerVariant(itemType, itemID uint8) uint8 {
	if itemType != 100 {
		return 0
	}
	switch itemID {
	case 115, 116, 117:
		return itemID - 114
	case 128:
		return 4
	default:
		return 0
	}
}

func FirecrackerPacket(v Firecracker) []byte {
	w := NewWriter(GXFirecracker)
	w.U8(v.Variant)
	w.F32(v.X)
	w.F32(v.Y)
	w.F32(v.Z)
	w.Text(v.Name)
	return w.Bytes()
}

func DecodeFirecracker(p []byte) (Firecracker, error) {
	r := NewReader(p)
	v := Firecracker{Variant: r.U8(), X: r.F32(), Y: r.F32(), Z: r.F32(), Name: r.Text()}
	if Op(p) != GXFirecracker || r.Err() != nil || r.Remaining() != 0 || p[len(p)-1] != 0 ||
		v.Variant < 1 || v.Variant > 4 || strings.TrimSpace(v.Name) == "" {
		return Firecracker{}, fmt.Errorf("invalid firecracker event")
	}
	for _, coordinate := range []float32{v.X, v.Y, v.Z} {
		if math.IsNaN(float64(coordinate)) || math.IsInf(float64(coordinate), 0) {
			return Firecracker{}, fmt.Errorf("invalid firecracker position")
		}
	}
	return v, nil
}
