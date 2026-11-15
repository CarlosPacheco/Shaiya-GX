package sdata

import (
	"encoding/binary"
	"os"
	"testing"
)

func TestBlessingOriginalTableCumulativeAndFactionSpecific(t *testing.T) {
	raw, err := os.ReadFile("../../../server/data/game/KillStatus.SData")
	if err != nil {
		t.Fatal(err)
	}
	tiers, err := ParseBlessing(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(tiers) != 60 || BlessingMaximum(tiers, 0) != 12288 || BlessingMaximum(tiers, 1) != 12288 {
		t.Fatal("original thresholds")
	}
	if got := BlessingBonuses(tiers, 0, 149); got != [28]uint16{} {
		t.Fatal("bonus below first threshold", got)
	}
	l, f := BlessingBonuses(tiers, 0, 450), BlessingBonuses(tiers, 1, 450)
	if l[10] != 5 || l[11] != 5 || l[12] != 5 || l[1] != 0 || f[1] != 6 || f[11] != 6 {
		t.Fatal("country inheritance", l, f)
	}
	l, f = BlessingBonuses(tiers, 0, 12288), BlessingBonuses(tiers, 1, 12288)
	if l[4] != 10 || l[11] != 10 || l[16] != 30 || l[24] != 5 || l[25] != 100 || l[26] != 10 || l[27] != 4 || f[1] != 9 || f[2] != 11 || f[20] != 7 {
		t.Fatal("final cumulative effects", l, f)
	}
}

func TestBlessingRejectsMalformedTables(t *testing.T) {
	row := make([]byte, 29)
	binary.LittleEndian.PutUint32(row, 1)
	binary.LittleEndian.PutUint32(row[5:], 150)
	for _, name := range []string{"country", "zero threshold", "truncated", "trailing", "unordered"} {
		t.Run(name, func(t *testing.T) {
			b := append([]byte{}, row...)
			switch name {
			case "country":
				b[4] = 2
			case "zero threshold":
				binary.LittleEndian.PutUint32(b[5:], 0)
			case "truncated":
				b = b[:28]
			case "trailing":
				b = append(b, 0)
			case "unordered":
				b = append(b, row[4:]...)
				binary.LittleEndian.PutUint32(b, 2)
			}
			if _, err := ParseBlessing(b); err == nil {
				t.Fatal("accepted malformed table")
			}
		})
	}
}
