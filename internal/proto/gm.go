package proto

import (
	"encoding/binary"
	"fmt"
	"unicode/utf8"
)

// GXGMState is a GX-only server response: self ID:u32, GM:u8, turbo:u8,
// message byte length:u16, UTF-8 message. Map teleport requests use GXGMMapTeleport; other requests use chat commands.
const GXGMState Opcode = 0x7F01

// GXVisibility: character ID:u32, concealed:u8. Sent only to authorized viewers.
const GXVisibility Opcode = 0x7F02

type GMState struct {
	ID        uint32
	GM, Turbo bool
	Message   string
}

func DecodeGMState(p []byte) (GMState, error) {
	if len(p) < 10 {
		return GMState{}, fmt.Errorf("short GX GM state")
	}
	n := int(binary.LittleEndian.Uint16(p[8:10]))
	if len(p) != 10+n || p[6] > 1 || p[7] > 1 || (p[6] == 0 && p[7] != 0) || !utf8.Valid(p[10:]) {
		return GMState{}, fmt.Errorf("invalid GX GM state")
	}
	return GMState{ID: binary.LittleEndian.Uint32(p[2:6]), GM: p[6] == 1, Turbo: p[7] == 1, Message: string(p[10:])}, nil
}

// GXItemCooldownReset clears the recipient’s item and mount cooldowns.
const GXItemCooldownReset Opcode = 0x7F1E

// GXGMModes tells a GM's own client which modes are active: self ID:u32,
// concealed:u8 (/char on), immune:u8 (/attack on), turbo:u8. Every byte is 0
// or 1 and the modes require GM. Sent on entry and after every change.
const GXGMModes Opcode = 0x7F29

type GMModes struct {
	ID                       uint32
	Concealed, Immune, Turbo bool
}

func DecodeGMModes(p []byte) (GMModes, error) {
	if len(p) != 9 || p[6] > 1 || p[7] > 1 || p[8] > 1 {
		return GMModes{}, fmt.Errorf("invalid GX GM modes")
	}
	return GMModes{ID: binary.LittleEndian.Uint32(p[2:6]), Concealed: p[6] == 1, Immune: p[7] == 1, Turbo: p[8] == 1}, nil
}
