package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"sort"
)

func equipmentOJLimits(defs *db.Defs) []proto.OJLimit {
	out := make([]proto.OJLimit, 0, len(defs.Items))
	for _, d := range defs.Items {
		if rt := realType(d.Type); rt < 1 || rt > 25 {
			continue
		}
		limit := proto.OJLimit{Type: d.Type, TypeID: d.TypeID}
		if d.Server > 0 && d.ReqWis > 0 {
			pool := uint8(9)
			if rollsStatsOnly(d.Type) {
				pool = 6
			}
			limit.Stats, limit.Value = min(d.Server, pool), uint8(min(d.ReqWis, 99))
		}
		out = append(out, limit)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		return a.Type < b.Type || a.Type == b.Type && a.TypeID < b.TypeID
	})
	return out
}
func (s *Session) sendOJLimits() {
	limits := equipmentOJLimits(db.MustDefs())
	for len(limits) > 0 {
		n := min(len(limits), 256)
		payload := proto.OJLimitsPacket(limits[:n])
		w := packet.New(packet.Opcode(proto.GXOJLimits))
		w.Bytes(payload[2:])
		s.send(w.MustFinish(), "GX equipment OJ limits")
		limits = limits[n:]
	}
}
