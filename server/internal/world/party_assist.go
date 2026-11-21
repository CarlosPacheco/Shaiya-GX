package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func (s *Session) handlePartyAssist(frame []byte) {
	me := s.player
	if me == nil || len(frame) != 7 || !me.IsAlive() {
		return
	}
	r := proto.NewReader(frame)
	action, id := r.U8(), r.U32()
	switch action {
	case proto.PartyAssistSelection:
		if id != 0 && !me.knowsAssistTarget(id) {
			id = 0
		}
		me.mu.Lock()
		me.assistTarget = id
		me.mu.Unlock()
	case proto.PartyAssistRequest:
		ally := FindPlayerByID(id)
		if ally == nil || ally == me || !ally.IsAlive() || !samePartyIDs(me.CharacterID, id) ||
			!me.knows(viewUser, id) || !me.canSee(ally) {
			return
		}
		ally.mu.Lock()
		target, sameMap := ally.assistTarget, ally.mapID == s.char.MapID
		ally.mu.Unlock()
		if sameMap && (target == 0 || ally.knowsAssistTarget(target) && me.knowsAssistTarget(target)) {
			reply := packet.New(packet.Opcode(proto.GXPartyAssist))
			reply.U32(id)
			reply.U32(target)
			s.send(reply.MustFinish(), "GX party assist")
		}
	}
}

func (p *ConnectedPlayer) knowsAssistTarget(id uint32) bool {
	if id == p.CharacterID || p.knows(viewNpc, id) || p.knows(viewMob, id) {
		return true
	}
	if !p.knows(viewUser, id) {
		return false
	}
	other := FindPlayerByID(id)
	return other != nil && other.MapID() == p.MapID() && p.canSee(other)
}
