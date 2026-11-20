package emblem

import (
	"encoding/binary"
	"testing"
)

func TestParseNamesReadsNativeTable(t *testing.T) {
	raw := make([]byte, 4+2*260)
	binary.LittleEndian.PutUint32(raw, 2)
	copy(raw[4:], "000_alien.dds")
	copy(raw[4+260:], "001_bearfoot.dds")
	names, err := ParseNames(raw)
	if err != nil || len(names) != 2 || names[0] != "000_alien.dds" || names[1] != "001_bearfoot.dds" {
		t.Fatalf("%v %v", names, err)
	}
	if _, err := ParseNames(raw[:len(raw)-1]); err == nil {
		t.Fatal("truncated table accepted")
	}
}

func TestPatternRoundTripAndPalette(t *testing.T) {
	b := [6]uint8{3, 4, 5, 6, 7, 8}
	if FromBytes(b).Bytes() != b || FromBytes(b).Empty() || !FromBytes([6]uint8{0, 0, 0, 9, 9, 9}).Empty() {
		t.Fatal("pattern bytes")
	}
	if Color(DefaultColor) != [3]uint8{255, 255, 255} || Color(200) != Palette[DefaultColor] || Palette[0] != [3]uint8{200, 0, 0} {
		t.Fatal("palette")
	}
}
