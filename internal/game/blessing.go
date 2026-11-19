package game

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

type BlessingState struct {
	Country            uint8
	Count, RemainingMS uint32
	Known              bool
	Maximum            uint32
	Bonuses            [27]uint16
	Entry              bool
	fractionalMS       float64
}

func (b *BlessingState) apply(op proto.Opcode, value proto.BlessingUpdate) {
	if op == proto.CountryKillCount {
		if b.Country != value.Country || value.Count == 0 {
			b.RemainingMS = 0
			b.fractionalMS = 0
		}
	} else {
		b.RemainingMS = value.RemainingMS
		b.fractionalMS = 0
	}
	b.Country, b.Count, b.Known = value.Country, value.Count, true
}

func (b *BlessingState) applySnapshot(value proto.BlessingSnapshot) {
	b.Country, b.Count, b.Known = value.Country, value.Count, true
	b.Maximum, b.Bonuses, b.Entry = value.Maximum, value.Bonuses, value.Entry
	b.RemainingMS, b.fractionalMS = value.RemainingMS, 0
}

// Advance only animates the native countdown; ownership/count resets remain
// authoritative packets. Keep sub-millisecond frame time instead of drifting.
func (b *BlessingState) Advance(seconds float64) {
	if seconds <= 0 || b.RemainingMS == 0 {
		return
	}
	ms := seconds*1000 + b.fractionalMS
	if ms >= float64(b.RemainingMS) {
		b.RemainingMS = 0
		b.fractionalMS = 0
		return
	}
	elapsed := uint32(ms)
	b.RemainingMS -= elapsed
	b.fractionalMS = ms - float64(elapsed)
}
