package catalogfiles

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/blessingrules"
	"github.com/Spelunkern/Shaiya-GX/internal/gameini"
	"github.com/Spelunkern/Shaiya-GX/internal/teleportrunes"
)

func ParseKillStatus(raw []byte) (blessingrules.Catalog, error) {
	sections, err := entries(raw, "Tier_", 8)
	if err != nil {
		return nil, err
	}
	var tiers blessingrules.Catalog
	for _, entry := range sections {
		r := gameini.Record{Fields: entry.Fields}
		t := blessingrules.Tier{Number: uint8(entry.Index), RequiredKills: uint32(r.Uint("RequiredKills", 32))}
		for i, effect := range blessingrules.Effects {
			if r.Text(effect.Column) != "" {
				t.Bonuses[i] = uint16(r.Uint(effect.Column, 16))
			}
		}
		if r.Err != nil {
			return nil, fmt.Errorf("tier %d: %w", entry.Index, r.Err)
		}
		tiers = append(tiers, t)
	}
	return tiers, tiers.Validate()
}

func EncodeKillStatus(tiers blessingrules.Catalog) []byte {
	var b strings.Builder
	b.WriteString("; Light and Fury use these same tiers with independent kill counters.\n")
	for _, tier := range tiers {
		fmt.Fprintf(&b, "\n[Tier_%d]\nRequiredKills=%d\n", tier.Number, tier.RequiredKills)
		for i, effect := range blessingrules.Effects {
			if tier.Bonuses[i] != 0 {
				fmt.Fprintf(&b, "%s=%d\n", effect.Column, tier.Bonuses[i])
			}
		}
	}
	return []byte(b.String())
}

func parsePoint(text string) (teleportrunes.Destination, error) {
	v := gameini.Fields(text)
	if len(v) != 4 {
		return teleportrunes.Destination{}, fmt.Errorf("expected (MapID, X, Y, Z)")
	}
	m, err := strconv.ParseUint(v[0], 10, 16)
	if err != nil {
		return teleportrunes.Destination{}, err
	}
	var coords [3]float64
	for i, value := range v[1:] {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil || !teleportrunes.ValidCoordinate(f) {
			return teleportrunes.Destination{}, fmt.Errorf("invalid coordinate %q", value)
		}
		coords[i] = f
	}
	return teleportrunes.Destination{MapID: uint16(m), X: coords[0], Y: coords[1], Z: coords[2]}, nil
}

func formatPoint(d teleportrunes.Destination) string {
	return fmt.Sprintf("(%d, %g, %g, %g)", d.MapID, d.X, d.Y, d.Z)
}

func ParseTeleportRunes(raw []byte) ([]teleportrunes.Rune, error) {
	sections, err := entries(raw, "Rune_", 15)
	if err != nil {
		return nil, err
	}
	runes := make([]teleportrunes.Rune, 0, len(sections))
	for _, section := range sections {
		light, err := parsePoint(section.Fields["LIGHT"])
		if err != nil {
			return nil, fmt.Errorf("rune %d Light: %w", section.Index, err)
		}
		fury, err := parsePoint(section.Fields["FURY"])
		if err != nil {
			return nil, fmt.Errorf("rune %d Fury: %w", section.Index, err)
		}
		runes = append(runes, teleportrunes.Rune{Index: int16(section.Index), Light: light, Fury: fury})
	}
	return runes, teleportrunes.Validate(runes)
}

func EncodeTeleportRunes(runes []teleportrunes.Rune) []byte {
	var b strings.Builder
	b.WriteString("; Rune index matches ReqVg. Destinations are (MapID, X, Y, Z).\n")
	for _, r := range runes {
		fmt.Fprintf(&b, "\n[Rune_%d]\nLight=%s\nFury=%s\n", r.Index, formatPoint(r.Light), formatPoint(r.Fury))
	}
	return []byte(b.String())
}

type Notice struct {
	Index     uint32
	Message   string
	Frequency uint32
	Fixed     [5]string
}

func ParseNotices(raw []byte) ([]Notice, error) {
	sections, err := entries(raw, "Notice_", 32)
	if err != nil {
		return nil, err
	}
	notices := make([]Notice, 0, len(sections))
	for _, section := range sections {
		r := gameini.Record{Fields: section.Fields}
		n := Notice{Index: uint32(section.Index), Message: r.Text("Message")}
		if r.Text("Frequency") != "" {
			n.Frequency = uint32(r.Uint("Frequency", 32))
		}
		if r.Err != nil || n.Index == 0 || strings.TrimSpace(n.Message) == "" || len(n.Message) > 127 || strings.ContainsAny(n.Message, "\x00\r\n") || n.Frequency > 5256000 {
			return nil, fmt.Errorf("notice %d has an invalid index, message or frequency", n.Index)
		}
		for i := range n.Fixed {
			n.Fixed[i] = r.Text(fmt.Sprintf("FixedHour%d", i+1))
		}
		notices = append(notices, n)
	}
	return notices, nil
}

func EncodeNotices(rows []Notice) []byte {
	var b strings.Builder
	b.WriteString("; Frequency is in minutes. Valid FixedHour entries (day/hour/minute, Sunday=0) take precedence.\n; Without a schedule, the index remains available to /notice indexN.\n")
	for _, n := range rows {
		fmt.Fprintf(&b, "\n[Notice_%d]\nMessage=%q\nFrequency=%d\n", n.Index, n.Message, n.Frequency)
		for i, v := range n.Fixed {
			if v != "" {
				fmt.Fprintf(&b, "FixedHour%d=%s\n", i+1, v)
			}
		}
	}
	return []byte(b.String())
}
