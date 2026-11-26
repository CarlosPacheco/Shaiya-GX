package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/movespeed"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func (a *App) sendLadderMode(mode uint8) {
	v := a.wv
	if err := a.world.Send(proto.LadderPacket(uint16(v.ladder.Index), mode)); err != nil {
		a.backToLoginID(12132)
		return
	}
	v.ladderSentMode, v.ladderSendAt = mode, a.clock+.25
}
func (a *App) updateLadderMovement(in *platform.Input, forward, back bool, strafe int, blocked bool, speed float32, dt float64) (bool, bool) {
	v := a.wv
	if v.ladderFinishing {
		if a.clock >= v.ladderSendAt {
			a.sendLadderMode(v.ladder.Mode)
		}
		return true, false
	}
	unavailable := a.state.Self.Dead || a.state.Self.Mounted || blocked || speed == 0
	if v.ladder.Active && (unavailable || in.HasKey(platform.KeySpace) || strafe != 0 || v.travel) {
		v.ladder.Active = false
		v.ladderRetryAt = a.clock + .4
		a.sendLadderMode(mapgeometry.LadderDetach)
		return false, false
	}
	if !v.ladder.Active && !v.jumpActive && !unavailable && !in.SuppressAutoRun && (forward || back) && a.clock >= v.ladderRetryAt && a.clock >= v.attackMoveUntil && !v.travel {
		approach := uint8(mapgeometry.LadderUp)
		if !forward {
			approach = mapgeometry.LadderDown
		}
		for index := range v.world.PlayerLadders() {
			m, p, ok := v.world.AttachLadder(v.pos, v.facing, index, approach)
			if !ok {
				continue
			}
			v.ladder = m
			v.jumping = false
			v.groundFall = -2.98
			v.slope.Frames = 0
			v.autoRun, v.travel, v.evading = false, false, false
			if v.self != nil {
				v.self.ReleaseAction()
				v.self.CancelCast()
			}
			// Flush the heading before requesting attachment: native direction remains authoritative.
			if err := a.world.Send(v.movePacket(0)); err != nil {
				a.backToLoginID(12133)
				return false, false
			}
			v.place(p)
			v.ladderSentMode = 255
			break
		}
	}
	if !v.ladder.Active {
		return false, false
	}
	mode := uint8(mapgeometry.LadderHold)
	if !in.SuppressAutoRun && a.clock >= v.attackMoveUntil {
		if forward {
			mode = mapgeometry.LadderUp
		} else if back {
			mode = mapgeometry.LadderDown
		}
	}
	v.ladder.Mode = mode
	if mode != v.ladderSentMode || a.clock >= v.ladderSendAt {
		a.sendLadderMode(mode)
	}
	old := v.pos
	if sp := a.state.Self.Speed; sp != nil {
		v.ladder.Tier = sp.Move
	} else {
		v.ladder.Tier = movespeed.Normal
	}
	next := v.world.StepLadder(&v.ladder, v.pos, float32(dt))
	v.place(next)
	l := v.world.PlayerLadders()[v.ladder.Index]
	v.facing = render.V3(l.Normal).Scale(-v.ladder.Side)
	if !v.ladder.Active {
		v.ladderFinishing = true
		// Let the server reach and confirm the same exit before resuming ground input.
	}
	return true, next != old
}
