package proto

// GXPartyAssist shares selected targets with nearby members of the same group.
const GXPartyAssist Opcode = 0x7F28

const (
	PartyAssistSelection uint8 = iota
	PartyAssistRequest
)

func PartyAssistPacket(action uint8, id uint32) []byte {
	w := NewWriter(GXPartyAssist)
	w.U8(action)
	w.U32(id)
	return w.Bytes()
}
