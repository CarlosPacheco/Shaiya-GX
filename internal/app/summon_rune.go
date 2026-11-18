package app

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type summonRuneView struct {
	id      uint32
	expires float64
	casts   map[uint32]bool
}

func (a *App) confirmedSummonRune(p []byte) {
	if proto.Op(p) == proto.UserLeave {
		delete(a.summonRune.casts, proto.NewReader(p).U32())
		return
	}
	r, err := proto.DecodeSummonRune(p)
	if err != nil {
		return
	}
	v := &a.summonRune
	switch r.Action {
	case proto.SummonCast:
		if a.wv == nil {
			return
		}
		actor := a.playerActor(r.Caster)
		if actor == nil && r.Caster != a.state.SelfID {
			return
		}
		if v.casts == nil {
			v.casts = make(map[uint32]bool)
		}
		v.casts[r.Caster] = true
		if actor != nil {
			actor.PlayCast(85, float32(r.RemainingMS)/1000)
			a.startMountCastEffects(r.Caster, float64(r.RemainingMS)/1000)
			if pos, ok := a.effectPosition(r.Caster); ok {
				a.playSound("ef_in_atrbu01.wav", pos[0], pos[2])
			}
		}
	case proto.SummonCastEnd:
		if v.casts[r.Caster] {
			delete(v.casts, r.Caster)
			if actor := a.playerActor(r.Caster); actor != nil {
				actor.CancelCast()
			}
			if a.wv != nil {
				a.stopCastEffects(r.Caster)
			}
		}
	case proto.SummonOffer:
		if a.summonMessage == nil || a.state.Self.Dead {
			return
		}
		prompt := a.msg.Format(818, map[string]string{"p": r.Name})
		if prompt == "" {
			prompt = fmt.Sprintf("Do you want to accept %s's party member summon?", r.Name)
		}
		a.summonMessage.OpenWithDetail(prompt, ui.MapName(a.msg, r.Map), [4]float32{1, 1, 0, 1}, a.font)
		v.id, v.expires = r.Offer, a.clock+float64(r.RemainingMS)/1000
	case proto.SummonResult:
		a.questMessage(r.Message)
	}
}

func (a *App) updateSummonPrompt(in *platform.Input, w, h int) bool {
	v := &a.summonRune
	if a.summonMessage == nil || !a.summonMessage.Visible {
		return false
	}
	if a.clock >= v.expires || a.state.Self.Dead {
		a.summonMessage.Visible = false
		return false
	}
	answer := a.summonMessage.Update(in, w, h)
	if answer != ui.MessageNone {
		if err := a.world.Send(proto.SummonAnswerPacket(v.id, answer == ui.MessageOK)); err != nil {
			a.backToLogin("Unable to answer the party summon.")
		}
	}
	return true
}

func (a *App) updateSummonCancel(in *platform.Input) bool {
	if !a.summonRune.casts[a.state.SelfID] || !in.HasKey(platform.KeyEscape) {
		return false
	}
	if err := a.world.Send(proto.SummonCancelPacket()); err != nil {
		a.backToLogin("Unable to cancel the party summon.")
	}
	return true
}
