package proto

import "fmt"

// GXKillNotice tells a user that it killed another player: the victim's name
// (21 bytes). The client says it in red with SysMsg 8, "Killed <t>".
const GXKillNotice Opcode = 0x7F1A

func DecodeKillNotice(p []byte) (string, error) {
	if len(p) != 23 {
		return "", fmt.Errorf("invalid GX kill notice length")
	}
	r := NewReader(p)
	name := r.Fixed(21)
	return name, r.Err()
}
