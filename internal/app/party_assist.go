package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func (a *App) partyShortcutTarget(index int) uint32 {
	p := &a.state.Party
	if p.Raid {
		if index < 6 {
			for slot, id := range p.RaidSlots {
				if id == a.state.SelfID {
					return p.RaidSlots[slot/6*6+uint16(index)]
				}
			}
		}
		return 0
	}
	if len(p.Members) == 0 {
		return a.state.SelfID
	}
	if index == 7 {
		index = 0
	}
	if index < len(p.Members) {
		return p.Members[index].ID
	}
	return 0
}

func (a *App) selectGroupTarget(id uint32) {
	if id == a.state.Target {
		return
	}
	if a.world != nil {
		if e := a.state.Entities[id]; e != nil && e.Kind == game.KindMob {
			if err := a.world.Send(proto.TargetMobPacket(id)); err != nil {
				a.backToLoginID(12003)
				return
			}
		}
	}
	a.state.PreviousTarget, a.state.Target = a.state.Target, id
}

func (a *App) publishPartySelection() {
	if a.world == nil || a.state.Self.Dead {
		return
	}
	id := uint32(0)
	if len(a.state.Party.Members) > 0 {
		if target := a.state.Target; target == a.state.SelfID || a.state.Entities[target] != nil {
			id = target
		}
	}
	if id == a.wv.partySelection {
		return
	}
	if err := a.world.Send(proto.PartyAssistPacket(proto.PartyAssistSelection, id)); err != nil {
		a.backToLoginID(12158)
		return
	}
	a.wv.partySelection = id
}

func (a *App) requestPartyAssist() {
	id := a.state.Target
	if a.world == nil || id == a.state.SelfID || a.state.Party.Member(id) == nil {
		return
	}
	if err := a.world.Send(proto.PartyAssistPacket(proto.PartyAssistRequest, id)); err != nil {
		a.backToLoginID(12159)
	}
}

func (a *App) confirmPartyAssist(frame []byte) {
	if len(frame) != 10 || a.state == nil || a.wv == nil || a.state.Self.Dead {
		return
	}
	r := proto.NewReader(frame)
	ally, target := r.U32(), r.U32()
	if a.state.Target != ally || a.state.Party.Member(ally) == nil {
		return
	}
	if target != 0 && target != a.state.SelfID && a.state.Entities[target] == nil {
		return
	}
	a.selectGroupTarget(target)
}
