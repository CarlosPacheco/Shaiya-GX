package sdata

import (
	"encoding/binary"
	"testing"
)

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
