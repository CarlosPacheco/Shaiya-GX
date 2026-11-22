package app

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type deathView struct {
	initialized, party, waiting, pending bool
	waitUntil                            float64
}

func (a *App) initDeath() {
	d := &a.wv.death
	if d.initialized {
		return
	}
	d.initialized = true
	a.deathUI.Reset()
	d.party = a.state.Party.Leader != 0 && a.state.Party.Leader != a.state.SelfID
}
func (a *App) deathText(id int, fallback string) string {
	if s := a.msg[id]; s != "" {
		return s
	}
	return fallback
}
func (a *App) configureDeath() {
	d := &a.wv.death
	kind := ui.DeathSingle
	msg := a.tr(10067)
	if d.waiting {
		kind = ui.DeathWaiting
		msg = fmt.Sprintf("%s (%d)", a.tr(10068), max(0, int(math.Ceil(d.waitUntil-a.clock))))
	} else if d.pending {
		kind = ui.DeathWaiting
		msg = a.tr(12342)
	} else if d.party {
		kind = ui.DeathParty
	}
	a.deathUI.Set(kind, msg)
}
func (a *App) requestDeathRevival(mode proto.ReviveMode) {
	if a.world == nil || a.clock < a.wv.reviveAt {
		return
	}
	if err := a.world.Send(proto.ReviveChoicePacket(mode)); err != nil {
		a.backToLoginID(12030)
		return
	}
	a.wv.reviveAt = a.clock + 3
	if mode == proto.ReviveParty {
		a.wv.death.waiting = true
		a.wv.death.waitUntil = a.clock + proto.PartyReviveSeconds
	} else {
		a.wv.death.pending = true
	}
}

// deathChoice maps the two supported revival choices.
func deathChoice(d *deathView, choice int) (proto.ReviveMode, bool) {
	if choice == 0 || d.waiting || d.pending {
		return 0, false
	}
	if choice == 2 && d.party {
		return proto.ReviveParty, true
	}
	if choice != 1 {
		return 0, false
	}
	return proto.ReviveBind, true
}
func (a *App) updateDeath(in *platform.Input, w, h int) {
	if a.wv == nil || a.state == nil || a.deathUI == nil {
		return
	}
	a.initDeath()
	if a.wv.death.pending && a.clock >= a.wv.reviveAt {
		a.wv.death.pending = false
	}
	a.configureDeath()
	d := &a.wv.death
	choice := a.deathUI.Update(in, w, h)
	if !d.waiting && !d.pending {
		for _, r := range in.Chars {
			if r == 'r' || r == 'R' {
				a.requestDeathRevival(proto.ReviveBind)
				return
			}
		}
	}
	if mode, send := deathChoice(d, choice); send {
		a.requestDeathRevival(mode)
	}
}
func (a *App) drawDeath(w, h int) {
	if a.wv == nil || a.deathUI == nil {
		return
	}
	a.initDeath()
	a.configureDeath()
	a.deathUI.Draw(a.sprites, a.font, w, h)
}
