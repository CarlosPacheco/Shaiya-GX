package app

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

func oxygenFraction(v proto.OxygenStatus, seconds float64) (float64, bool) {
	if v.Phase == proto.OxygenHidden {
		return 0, false
	}
	left := float64(v.RemainingMS) * .001
	switch v.Phase {
	case proto.OxygenDraining:
		left -= max(0, seconds)
	case proto.OxygenRefilling:
		left += max(0, seconds) * proto.OxygenRefillRate
	}
	fraction := min(1, max(0, left/proto.OxygenCapacity.Seconds()))
	return fraction, v.Phase != proto.OxygenRefilling || fraction < 1
}

func (a *App) drawCastAndOxygen(w, h int) {
	v := a.wv
	if v == nil || a.state.Self.Dead {
		return
	}
	var left, total float32
	var casting bool
	if v.self != nil {
		left, total, casting = v.self.CastTiming()
	}
	fraction, breathing := oxygenFraction(a.state.Oxygen, a.clock-a.oxygenAt)
	if !casting && !breathing {
		return
	}
	x, y, projected := castBarOrigin(v.pos, a.scene3d.View, a.scene3d.Proj, w, h)
	if casting && projected {
		a.hud.DrawCast(a.sprites, a.white, x, y, left, total)
	}
	if breathing {
		if !projected || x < 0 || x+13 >= w || y < 0 || y+64 >= h {
			x, y = w/2+24, h/2-32
		} else if casting {
			x += 8
		}
		a.hud.DrawOxygen(a.sprites, a.white, x, y, fraction)
	}
}
