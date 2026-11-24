package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// portalRetrySeconds is how long a request waits for its answer before the
// character, still standing in the portal, asks again.
const portalRetrySeconds = 5

// portalContact returns the index of the portal volume holding the actor origin
// (Game.exe tests the position .9 above the feet), or -1.
func portalContact(portals []world.Box, x, y, z float32) int {
	y += .9
	for i, b := range portals {
		if x >= b.Min[0] && x <= b.Max[0] && y >= b.Min[1] && y <= b.Max[1] && z >= b.Min[2] && z <= b.Max[2] {
			return i
		}
	}
	return -1
}

// updatePortalContact is Game.exe 0x4435d5: when the character first touches a
// portal volume the client sends 0x020A with its index, once, and waits for the
// server's answer. Portals whose target needs a party, a schedule or a guild
// are decided only by that request.
func (a *App) updatePortalContact() {
	v := a.wv
	if v == nil || v.world == nil || v.world.WLD == nil || a.world == nil || a.state == nil || a.state.MapID != v.mapID {
		return
	}
	index := portalContact(v.world.WLD.Portals, v.pos[0], v.pos[1], v.pos[2])
	if index < 0 || index > 255 || a.state.Self.Dead {
		v.portalAt = 0
		return
	}
	if v.portalAt == index+1 && a.clock < v.portalRetryAt {
		return
	}
	if err := a.world.Send(proto.PortalRequestPacket(uint8(index))); err != nil {
		return
	}
	v.portalAt, v.portalRetryAt = index+1, a.clock+portalRetrySeconds
}

// confirmedPortalResult shows the refusal of a portal request (Game.exe 0x5696c0).
func (a *App) confirmedPortalResult(p []byte) {
	r, err := proto.DecodePortalResult(p)
	if err != nil || a.wv == nil {
		return
	}
	if id := proto.PortalMessageID(r.Code); id != 0 {
		a.systemMessage(7, id, nil)
	}
}
