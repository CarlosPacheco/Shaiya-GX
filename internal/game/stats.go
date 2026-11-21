package game

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

// Keep the last confirmed derived stats until the server sends a fresh 0x0526.
func (s *State) applyStatPoints(d proto.StatPoints) {
	s.LastStatPoints = d
	s.Details.StatPoint -= uint16(min(uint32(s.Details.StatPoint), d.Total()))
	for i, c := range s.Slots {
		if c.ID != s.SelfID {
			continue
		}
		addAttributes(&c, d)
		s.Slots[i] = c
	}
}

func addAttributes(c *proto.CharacterSlot, d proto.StatPoints) {
	values := []*uint16{&c.Str, &c.Dex, &c.Rec, &c.Int, &c.Wis, &c.Luc}
	for j, p := range values {
		*p = uint16(min(uint32(*p)+uint32(d[j]), 65535))
	}
}
