package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const (
	specialFirecracker = 46
	firecrackerRadius  = 40
)

func (s *Session) useFirecracker(index int, def *db.ItemDef) bool {
	ch := s.char
	variant := proto.FirecrackerVariant(def.Type, def.TypeID)
	if variant == 0 || s.dead || ch.CurrentHP == 0 {
		return false
	}
	now := time.Now()
	delay := cooldownForReqIg(def.ReqIg)
	if delay > 0 && now.Sub(s.itemCooldowns[def.ReqIg]) < delay {
		return false
	}
	item, err := s.consumeOne(index)
	if err != nil {
		s.logErr("consume firecracker", err)
		return false
	}
	if delay > 0 {
		if s.itemCooldowns == nil {
			s.itemCooldowns = make(map[uint8]time.Time)
		}
		s.itemCooldowns[def.ReqIg] = now
	}
	s.send(buildUseItemResult(ch.ID, &item), "0x050A firecracker used")
	payload := proto.FirecrackerPacket(proto.Firecracker{Variant: variant, X: ch.PosX, Y: ch.PosY, Z: ch.PosZ, Name: ch.Name})
	w := packet.New(packet.Opcode(proto.GXFirecracker))
	w.Bytes(payload[2:])
	frame := w.MustFinish()
	s.send(frame, "local firecracker")
	for _, viewer := range PlayersNearExcluding(ch.MapID, ch.ID, ch.PosX, ch.PosZ, firecrackerRadius) {
		viewer.Send(frame, "local firecracker")
	}
	return true
}
