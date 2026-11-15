package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"sort"
)

func chestCloakFits(defs *db.Defs) []proto.ChestCloakFit {
	rows := make([]proto.ChestCloakFit, 0)
	for _, d := range defs.Items {
		if realType(d.Type) != 17 {
			continue
		}
		fit := uint8(0)
		if d.ReqVg >= 1 && d.ReqVg <= 8 {
			fit = uint8(d.ReqVg - 1)
		}
		rows = append(rows, proto.ChestCloakFit{Type: d.Type, TypeID: d.TypeID, Fit: fit})
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		return a.Type < b.Type || a.Type == b.Type && a.TypeID < b.TypeID
	})
	return rows
}
func (s *Session) sendChestCloakFits() {
	rows := chestCloakFits(db.MustDefs())
	for len(rows) > 0 {
		n := min(len(rows), 256)
		payload := proto.ChestCloakFitsPacket(rows[:n])
		w := packet.New(packet.Opcode(proto.GXChestCloakFits))
		w.Bytes(payload[2:])
		s.send(w.MustFinish(), "GX chest cloak fits")
		rows = rows[n:]
	}
}
