package proto

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestPlayerCorrectionRejectsMalformedVerticalState(t *testing.T) {
	w := NewWriter(GXPlayerCorrection)
	w.U32(42)
	w.F32(2)
	w.F32(5)
	w.F32(7)
	w.U8(0x80)
	w.U16(123)
	w.Bool(true)
	w.F32(-4)
	w.F32(-2.98)
	w.F32(0)
	p := w.Bytes()
	c, err := DecodePlayerCorrection(p)
	if err != nil || c.ID != 42 || !c.Jumping || c.JumpFall != -4 || c.Y != 5 {
		t.Fatal(c, err)
	}
	for _, size := range []int{0, 21, 29, 30, 33, 35} {
		malformed := make([]byte, size)
		copy(malformed, p)
		if _, err := DecodePlayerCorrection(malformed); err == nil {
			t.Fatalf("accepted length %d", size)
		}
	}
	for _, offset := range []int{6, 10, 14, 22, 26, 30} {
		malformed := append([]byte(nil), p...)
		binary.LittleEndian.PutUint32(malformed[offset:], math.Float32bits(float32(math.NaN())))
		if _, err := DecodePlayerCorrection(malformed); err == nil {
			t.Fatal("accepted non-finite state", offset)
		}
	}
	malformed := append([]byte(nil), p...)
	malformed[21] = 2
	if _, err := DecodePlayerCorrection(malformed); err == nil {
		t.Fatal("accepted invalid jump flag")
	}
}
