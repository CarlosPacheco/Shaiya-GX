package proto

import "fmt"

// PortalResult is the server's refusal of a 0x020A portal request (Game.exe
// 0x597a30): the portal index and a result code that selects a SysMsg.
type PortalResult struct{ Index, Code uint8 }

// PortalRequestPacket is the 0x020A request Game.exe sends once when the
// character first touches a portal volume (0x598490); the byte is the portal's
// index in the map's portal list.
func PortalRequestPacket(index uint8) []byte {
	w := NewWriter(CharacterEnteredPortal)
	w.U8(index)
	return w.Bytes()
}

func DecodePortalResult(p []byte) (PortalResult, error) {
	r := NewReader(p)
	v := PortalResult{Index: r.U8(), Code: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("invalid portal result length")
	}
	return v, nil
}

// PortalMessageID is the SysMsg Game.exe shows for a refusal code (jump table
// 0x569750); 0 is no message (code 5 opens the ranking-battle confirmation).
func PortalMessageID(code uint8) int {
	ids := [...]int{8038, 8039, 8040, 8041, 8042, 0, 8044, 8047, 8045, 6600, 6601, 6602, 6603}
	if int(code) < len(ids) {
		return ids[code]
	}
	return 0
}
