package proto

import "fmt"

// GXPlayerFeedback carries a presentation event for an already visible player.
const GXPlayerFeedback Opcode = 0x7F25

const (
	PlayerSpawnFeedback uint8 = 1
	PlayerSkillFeedback uint8 = 2
)

func PlayerFeedbackPacket(id uint32, kind uint8) []byte {
	w := NewWriter(GXPlayerFeedback)
	w.U32(id)
	w.U8(kind)
	return w.Bytes()
}

func DecodePlayerFeedback(p []byte) (id uint32, kind uint8, err error) {
	r := NewReader(p)
	id, kind = r.U32(), r.U8()
	if Op(p) != GXPlayerFeedback || r.Err() != nil || r.Remaining() != 0 ||
		id == 0 || (kind != PlayerSpawnFeedback && kind != PlayerSkillFeedback) {
		err = fmt.Errorf("invalid player feedback")
	}
	return
}
