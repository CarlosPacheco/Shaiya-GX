package sdata

import "fmt"

type BlessingEffect struct {
	Type  uint8
	Value uint16
}
type BlessingTier struct {
	Country uint8
	Kills   uint32
	Level   uint16
	Effects [6]BlessingEffect
}

// ParseBlessing reads EP5 KillStatus.SData (the server uses 25-byte records).
func ParseBlessing(raw []byte) ([]BlessingTier, error) {
	b, err := Decrypt(raw)
	if err != nil {
		return nil, err
	}
	c := &cursor{b: b}
	n := c.count(25)
	out := make([]BlessingTier, 0, n)
	var last [2]uint32
	var counts [2]int
	for i := 0; i < n && c.err == nil; i++ {
		t := BlessingTier{Country: c.u8(), Kills: c.u32(), Level: c.u16()}
		for j := range t.Effects {
			t.Effects[j] = BlessingEffect{Type: c.u8(), Value: c.u16()}
		}
		if t.Country > 1 {
			return nil, fmt.Errorf("blessing: invalid country %d", t.Country)
		}
		if t.Kills == 0 || counts[t.Country] > 0 && t.Kills <= last[t.Country] {
			return nil, fmt.Errorf("blessing: unordered threshold for country %d", t.Country)
		}
		counts[t.Country]++
		if counts[t.Country] > 255 {
			return nil, fmt.Errorf("blessing: too many country tiers")
		}
		last[t.Country] = t.Kills
		out = append(out, t)
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(b) {
		return nil, fmt.Errorf("blessing: %d trailing bytes", len(b)-c.off)
	}
	return out, nil
}

func BlessingMaximum(tiers []BlessingTier, country uint8) uint32 {
	var result uint32
	for _, tier := range tiers {
		if tier.Country == country {
			result = max(result, tier.Kills)
		}
	}
	return result
}

// BlessingBonuses follows Game.exe 00479df0/00479d70: later rows retain
// the maximum value of each earlier type, independently for each country.
// Type zero is padding. Below the first threshold the native dummy row is empty.
func BlessingBonuses(tiers []BlessingTier, country uint8, count uint32) (out [28]uint16) {
	for _, tier := range tiers {
		if tier.Country != country || tier.Kills > count {
			continue
		}
		var row [28]uint16
		for _, effect := range tier.Effects {
			if effect.Type > 0 && int(effect.Type) < len(row) {
				row[effect.Type] = effect.Value
			}
		}
		for typ, value := range row {
			out[typ] = max(out[typ], value)
		}
	}
	return
}
