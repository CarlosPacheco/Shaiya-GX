package proto

const (
	GXSummonRune  Opcode = 0x7F14
	SummonCast    uint8  = 1
	SummonCastEnd uint8  = 2
	SummonOffer   uint8  = 3
	SummonResult  uint8  = 4
	SummonAnswer  uint8  = 5
	SummonCancel  uint8  = 6
)

type SummonRuneEvent struct {
	Action           uint8
	Caster, Offer    uint32
	Map, RemainingMS uint16
	Name, Message    string
}

func DecodeSummonRune(p []byte) (v SummonRuneEvent, err error) {
	r := NewReader(p)
	v.Action = r.U8()
	switch v.Action {
	case SummonCast, SummonCastEnd:
		v.Caster, v.RemainingMS = r.U32(), r.U16()
	case SummonOffer:
		v.Offer, v.Caster, v.Map, v.RemainingMS = r.U32(), r.U32(), r.U16(), r.U16()
		v.Name = r.Fixed(21)
	case SummonResult:
		v.Message = r.Text()
	}
	return v, r.Err()
}

func SummonAnswerPacket(id uint32, accept bool) []byte {
	w := NewWriter(GXSummonRune)
	w.U8(SummonAnswer)
	w.U32(id)
	w.Bool(accept)
	return w.Bytes()
}

func SummonCancelPacket() []byte {
	w := NewWriter(GXSummonRune)
	w.U8(SummonCancel)
	return w.Bytes()
}
