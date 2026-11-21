package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func (s *Session) sendPlayerFeedback(kind uint8) {
	p := proto.PlayerFeedbackPacket(s.char.ID, kind)
	w := packet.New(packet.Opcode(proto.GXPlayerFeedback))
	w.Bytes(p[2:])
	s.sendToViewers(w.MustFinish(), "GX player feedback")
}
