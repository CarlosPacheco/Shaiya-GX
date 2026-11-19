package proto

import "fmt"

// BlessInit (0x0211) is faction:u8, count:u32, remaining milliseconds:u32;
// 0x020F updates the faction count without a timer.
type BlessingUpdate struct {
	Country            uint8
	Count, RemainingMS uint32
}

func DecodeBlessing(p []byte) (BlessingUpdate, error) {
	r := NewReader(p)
	b := BlessingUpdate{Country: r.U8(), Count: r.U32()}
	if Op(p) == BlessInit {
		b.RemainingMS = r.U32()
	}
	if err := r.Err(); err != nil {
		return b, err
	}
	if b.Country > 1 || r.Remaining() != 0 {
		return b, fmt.Errorf("invalid blessing country or trailing bytes")
	}
	return b, nil
}

// GXBlessingSnapshot carries the server's presentation data independently of
// the native count packets. Entry snapshots restore the timer without a burst.
const GXBlessingSnapshot Opcode = 0x7F1B

type BlessingSnapshot struct {
	Country                     uint8
	Entry                       bool
	Count, Maximum, RemainingMS uint32
	Bonuses                     [27]uint16
}

func DecodeBlessingSnapshot(p []byte) (BlessingSnapshot, error) {
	r := NewReader(p)
	country, flags := r.U8(), r.U8()
	b := BlessingSnapshot{Country: country, Entry: flags == 1, Count: r.U32(), Maximum: r.U32(), RemainingMS: r.U32()}
	for i := range b.Bonuses {
		b.Bonuses[i] = r.U16()
	}
	if err := r.Err(); err != nil {
		return b, err
	}
	if country > 1 || flags > 1 || b.Maximum == 0 || b.Count > b.Maximum || b.RemainingMS > 600000 ||
		b.RemainingMS != 0 && b.Count != b.Maximum || r.Remaining() != 0 {
		return b, fmt.Errorf("invalid blessing snapshot")
	}
	return b, nil
}
