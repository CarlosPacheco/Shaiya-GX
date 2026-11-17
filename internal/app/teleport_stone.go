package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type teleportStoneView struct {
	open, pending, casting bool
	deadline               float64

	points [proto.TeleportPointCount]proto.TeleportPoint
}

func (a *App) loadRecordedPoints() bool {
	if a.recordedPoints == nil {
		var textures ui.Textures
		if a.assets != nil {
			textures = a
		}
		panel, err := ui.NewRecordedPoints(textures)
		if err != nil {
			a.questMessage("Recorded Point interface resources are missing.")
			return false
		}
		a.recordedPoints = panel
	}
	return true
}

func (a *App) confirmedTeleportStone(p []byte) {
	v := &a.teleportStone
	r, err := proto.DecodeTeleportStone(p)
	if err != nil {
		return
	}
	if r.Action == proto.TeleportOpen {
		a.inventory.pending = false
		*v = teleportStoneView{open: r.OK && a.loadRecordedPoints(), points: r.Points}
		if !r.OK {
			a.questMessage(r.Message)
		}
		return
	}
	if r.Action == proto.TeleportCasting && r.OK && v.pending {
		v.open, v.casting = false, true
		v.deadline = a.clock + float64(r.CastMS)/1000 + 5
		if actor := a.playerActor(a.state.SelfID); actor != nil {
			actor.PlayCast(85, float32(r.CastMS)/1000)
			a.startMountCastEffects(a.state.SelfID, float64(r.CastMS)/1000)
			a.playSound("ef_in_atrbu01.wav", a.state.Self.X, a.state.Self.Z)
		}
		return
	}
	if v.casting && r.Action == proto.TeleportTravel {
		a.stopTeleportCast()
		if !r.OK {
			a.questMessage(r.Message)
		}
		return
	}
	if v.open && v.pending {
		v.pending, v.points = false, r.Points
		if r.Message != "" {
			a.questMessage(r.Message)
		}
	}
}

func (a *App) stopTeleportCast() {
	if a.teleportStone.casting {
		if actor := a.playerActor(a.state.SelfID); actor != nil {
			actor.CancelCast()
		}
		if a.wv != nil {
			a.stopCastEffects(a.state.SelfID)
		}
	}
	a.teleportStone = teleportStoneView{}
}

func (a *App) updateTeleportCast(in *platform.Input) bool {
	v := &a.teleportStone
	if !v.casting {
		return false
	}
	if a.clock >= v.deadline {
		a.stopTeleportCast()
		a.questMessage("No teleportation response.")
		return false
	}
	if in.HasKey(platform.KeyEscape) {
		if err := a.world.Send(proto.TeleportStonePacket(proto.TeleportCancel, 0)); err != nil {
			a.backToLogin("Unable to cancel the teleportation.")
		}
		return true
	}
	return false
}

func (a *App) updateTeleportStone(in *platform.Input, w, h int) bool {
	v := &a.teleportStone
	if !v.open {
		return false
	}
	if v.pending && a.clock >= v.deadline {
		v.pending = false
		a.questMessage("No response. Use the stone again to refresh.")
	}
	if !a.loadRecordedPoints() {
		*v = teleportStoneView{}
		return true
	}
	action, slot, closed := a.recordedPoints.Update(in, w, h, v.pending, v.points)
	if closed {
		*v = teleportStoneView{}
		return true
	}
	if action == 0 {
		x, y := a.recordedPoints.Origin(w, h)
		return panelPointer(in, x, y, ui.RecordedPointsWidth, ui.RecordedPointsHeight)
	}
	if err := a.world.Send(proto.TeleportStonePacket(action, slot)); err != nil {
		a.backToLogin("Unable to use a recorded point.")
		return true
	}
	v.pending, v.deadline = true, a.clock+10
	return true
}

func (a *App) drawTeleportStone(w, h int) {
	v := &a.teleportStone
	if !v.open || !a.loadRecordedPoints() {
		return
	}
	x, y := a.recordedPoints.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.RecordedPointsWidth, ui.RecordedPointsHeight)
	a.recordedPoints.Draw(a.sprites, a.font, a.msg, w, h, v.pending, v.points)
}
