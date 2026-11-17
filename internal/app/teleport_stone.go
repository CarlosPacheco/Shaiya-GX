package app

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type teleportStoneView struct {
	open, pending, casting bool
	deadline               float64
	selected               uint8
	message                string
	points                 [proto.TeleportPointCount]proto.TeleportPoint
}

const teleportWidth, teleportHeight = 352, 258

func teleportOrigin(w, h int) (int, int) {
	return max(0, (w-teleportWidth)/2), max(0, (h-teleportHeight)/2)
}

func (a *App) confirmedTeleportStone(p []byte) {
	v := &a.teleportStone
	r, err := proto.DecodeTeleportStone(p)
	if err != nil {
		return
	}
	if r.Action == proto.TeleportOpen {
		a.inventory.pending = false
		*v = teleportStoneView{open: r.OK, points: r.Points}
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
		v.pending, v.points, v.message = false, r.Points, r.Message
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

func (a *App) updateTeleportStone(in *platform.Input, w, h int) {
	v := &a.teleportStone
	if v.pending && a.clock >= v.deadline {
		v.pending = false
		v.message = "No response. Use the stone again to refresh."
	}
	if in.HasKey(platform.KeyEscape) && !v.pending {
		*v = teleportStoneView{}
		return
	}
	if !in.Clicked || v.pending {
		return
	}
	x, y := teleportOrigin(w, h)
	mx, my := in.MouseX-x, in.MouseY-y
	if mx >= 312 && mx < 342 && my >= 8 && my < 33 {
		*v = teleportStoneView{}
		return
	}
	if mx >= 14 && mx < 338 && my >= 46 && my < 174 {
		v.selected = uint8((my - 46) / 32)
		v.message = ""
		return
	}
	if my < 211 || my >= 243 {
		return
	}
	var action uint8
	switch {
	case mx >= 14 && mx < 170:
		action = proto.TeleportSave
	case mx >= 182 && mx < 338 && v.points[v.selected].Saved:
		action = proto.TeleportTravel
	default:
		return
	}
	if err := a.world.Send(proto.TeleportStonePacket(action, v.selected)); err != nil {
		a.backToLogin("Unable to use a recorded point.")
		return
	}
	v.pending, v.deadline = true, a.clock+10
	v.message = ""
}

func (a *App) drawTeleportStone(w, h int) {
	v := &a.teleportStone
	if !v.open {
		return
	}
	x, y := teleportOrigin(w, h)
	a.clearTooltipUnderPanel(x, y, teleportWidth, teleportHeight)
	rect := func(dx, dy, width, height int, color [4]float32) {
		a.sprites.Draw(a.white, float32(x+dx), float32(y+dy), float32(width), float32(height), 0, 0, 1, 1, color)
	}
	label := func(text string, dx, dy, width int, color [4]float32) {
		a.font.Draw(a.sprites, text, x+dx, y+dy, color, width)
	}
	accent := [4]float32{.88, .73, .45, 1}
	muted := [4]float32{.67, .69, .70, 1}
	if frame := a.assets.Texture("interface/talk1_bg.png"); frame != nil {
		a.sprites.Draw(frame, float32(x), float32(y), teleportWidth, 44, 0, 0, 1, 80.0/512, white)
		a.sprites.Draw(frame, float32(x), float32(y+44), teleportWidth, 177, 0, 80.0/512, 1, 309.0/512, white)
		a.sprites.Draw(frame, float32(x), float32(y+221), teleportWidth, 37, 0, 309.0/512, 1, 386.0/512, white)
	} else {
		rect(0, 0, teleportWidth, teleportHeight, [4]float32{.12, .10, .08, .98})
	}
	if stone := a.itemsItem(100, 65); stone != nil {
		a.drawItemIcon(stone, x+35, y+8, 23, white)
	}
	label("Recorded Points", 95, 13, 210, white)
	label("×", 322, 12, 14, white)
	for i, p := range v.points {
		rowY := 46 + i*32
		color := [4]float32{.17, .14, .11, .92}
		if uint8(i) == v.selected {
			color = [4]float32{.34, .27, .16, .92}
		}
		rect(14, rowY, 324, 29, color)
		label(fmt.Sprint(i+1), 24, rowY+7, 16, accent)
		if p.Saved {
			label(ui.MapName(a.msg, p.Map), 50, rowY+7, 182, white)
			label(fmt.Sprintf("%.0f, %.0f", p.X, p.Z), 239, rowY+7, 93, muted)
		} else {
			label("Empty", 50, rowY+7, 182, muted)
		}
	}
	status := v.message
	if v.pending {
		status = "Waiting…"
	} else if status == "" {
		status = "Travel uses one Teleportation Stone."
	}
	label(status, 14, 182, 324, muted)
	button := func(text string, dx int, enabled bool) {
		color := [4]float32{.33, .26, .16, 1}
		ink := white
		if !enabled {
			color, ink = [4]float32{.17, .14, .11, .92}, muted
		}
		rect(dx, 211, 156, 32, color)
		label(text, dx+(156-a.font.Width(text))/2, 219, 152, ink)
	}
	save := "Save current point"
	if v.points[v.selected].Saved {
		save = "Replace point"
	}
	button(save, 14, !v.pending)
	button("Travel", 182, !v.pending && v.points[v.selected].Saved)
}
