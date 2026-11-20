package proto

import "fmt"

// EmblemResult is the 15-byte 0x0803 reply of ps_game CUser::ItemCloakAdd
// (0x00464ad0): result 1 applied the pattern and charged the fee, result 0
// refused it for lack of gold and echoes the request.
type EmblemResult struct {
	Result, Bag, Slot uint8
	Pattern           [6]uint8
	Money             uint32
}

func DecodeEmblemResult(p []byte) (EmblemResult, error) {
	r := NewReader(p)
	v := EmblemResult{Result: r.U8(), Bag: r.U8(), Slot: r.U8()}
	for i := range v.Pattern {
		v.Pattern[i] = r.U8()
	}
	v.Money = r.U32()
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.Bag > 5 || v.Slot >= 24 {
		return v, fmt.Errorf("invalid emblem result")
	}
	return v, nil
}

// EmblemRemoved is the 0x0804 reply: the cloak's bag and slot.
type EmblemRemoved struct{ Bag, Slot uint8 }

func DecodeEmblemRemoved(p []byte) (EmblemRemoved, error) {
	r := NewReader(p)
	v := EmblemRemoved{Bag: r.U8(), Slot: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Bag > 5 || v.Slot >= 24 {
		return v, fmt.Errorf("invalid emblem removal")
	}
	return v, nil
}

// EmblemApplyPacket is the 0x0803 request: bag, slot and the six pattern bytes.
func EmblemApplyPacket(bag, slot uint8, pattern [6]uint8) []byte {
	w := NewWriter(CloakAdd)
	w.Raw([]byte{bag, slot})
	w.Raw(pattern[:])
	return w.Bytes()
}

// EmblemRemovePacket is the 0x0804 request.
func EmblemRemovePacket(bag, slot uint8) []byte {
	w := NewWriter(CloakRemove)
	w.Raw([]byte{bag, slot})
	return w.Bytes()
}
