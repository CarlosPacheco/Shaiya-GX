// Package emblem holds the cloak emblem data shared by the client and tests:
// the 36-colour palette (Game.exe 0x007091a8, three 32-bit values per entry),
// the pattern tables and the byte layout stored in the cloak's six gem bytes.
package emblem

import (
	"encoding/binary"
	"fmt"
)

// Fee is the gold the emblem NPC charges (0x35cc in the window object and
// ps_game CUser::ItemCloakAdd). Removing an emblem is free.
const Fee = 450000

// MinLevel is the character level below which the window refuses a cloak
// (Game.exe 0x47f4xx compares DAT_008655f8 with 15, SysMsg 5358).
const MinLevel = 15

// Pattern is the six cloak bytes. Shapes are one-based table indices (0 is
// none): Front, Back and Emblem. Colours are palette indices for the same
// three shapes, 35 (white) by default.
type Pattern struct {
	Front, Back, Emblem                uint8
	FrontColor, BackColor, EmblemColor uint8
}

// DefaultColor is the white entry the window uses when no shape is selected.
const DefaultColor = 35

// PaletteSize is the number of selectable colours (a 6x6 grid).
const PaletteSize = 36

// Palette holds RGB triples indexed by the stored colour byte.
var Palette = [PaletteSize][3]uint8{
	{200, 0, 0},
	{122, 0, 4},
	{211, 73, 0},
	{255, 142, 9},
	{255, 144, 116},
	{40, 0, 0},
	{251, 232, 6},
	{190, 201, 0},
	{120, 120, 0},
	{168, 161, 73},
	{243, 231, 171},
	{40, 40, 0},
	{0, 200, 0},
	{131, 238, 144},
	{51, 113, 48},
	{0, 122, 0},
	{0, 41, 0},
	{0, 43, 46},
	{1, 0, 207},
	{39, 120, 199},
	{36, 203, 195},
	{0, 120, 120},
	{0, 0, 114},
	{2, 0, 40},
	{190, 3, 216},
	{253, 196, 189},
	{194, 125, 190},
	{120, 1, 119},
	{119, 124, 182},
	{44, 0, 40},
	{0, 0, 0},
	{120, 120, 120},
	{200, 200, 200},
	{214, 214, 212},
	{39, 40, 122},
	{255, 255, 255},
}

// Color returns the palette entry; an out-of-range index reads as white.
func Color(i uint8) [3]uint8 {
	if int(i) >= PaletteSize {
		return Palette[DefaultColor]
	}
	return Palette[i]
}

// FromBytes reads the six gem bytes of a cloak.
func FromBytes(b [6]uint8) Pattern {
	return Pattern{b[0], b[1], b[2], b[3], b[4], b[5]}
}

// Bytes is the order sent by 0x0803 and stored in the item.
func (p Pattern) Bytes() [6]uint8 {
	return [6]uint8{p.Front, p.Back, p.Emblem, p.FrontColor, p.BackColor, p.EmblemColor}
}

// Empty reports a cloak without any shape.
func (p Pattern) Empty() bool { return p.Front == 0 && p.Back == 0 && p.Emblem == 0 }

// ParseNames reads EmblemList.dat, EmblemBack.dat or EmblemFront.dat: an
// int32 count followed by 260-byte NUL-padded texture names (0x55ac10).
func ParseNames(raw []byte) ([]string, error) {
	if len(raw) < 4 {
		return nil, fmt.Errorf("emblem: short table")
	}
	n := int(int32(binary.LittleEndian.Uint32(raw)))
	if n < 0 || n > 255 || len(raw) != 4+n*260 {
		return nil, fmt.Errorf("emblem: invalid table of %d entries in %d bytes", n, len(raw))
	}
	names := make([]string, n)
	for i := range names {
		field := raw[4+i*260 : 4+(i+1)*260]
		end := 0
		for end < len(field) && field[end] != 0 {
			end++
		}
		names[i] = string(field[:end])
	}
	return names, nil
}
