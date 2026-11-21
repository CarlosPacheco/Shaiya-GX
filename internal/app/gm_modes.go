package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type gmModeInfo struct {
	name, command, description string
}

// The three switches mirror the original admin commands /char and /attack
// (0xFA01/0xFA02 and 0xFA05/0xFA06) plus GX's turbo.
var gmModeInfos = [ui.GMModeCount]gmModeInfo{
	{"Char", "/char", "Stealth mode: your character is transparent and other players cannot see you. Same as /char on and /char off."},
	{"Attack", "/attack", "Untouchable mode: nothing can attack or target you. Same as /attack on and /attack off."},
	{"Turbo", "/turbo", "Sprinter mode: movement speed is multiplied by 12 and fall damage is disabled. Same as /turbo."},
}

func (a *App) gmModesVisible() bool {
	return a.state != nil && a.state.Self.IsGM && (a.world != nil || a.gmSend != nil)
}

func (a *App) gmModeActive(i int) bool {
	switch i {
	case 0:
		return a.state.GMConcealed
	case 1:
		return a.state.GMImmune
	}
	return a.state.GMTurbo
}

// toggleGMMode sends the command that flips mode i; the server answers with the
// new state, so nothing changes locally until it does.
func (a *App) toggleGMMode(i int) {
	if !a.gmModesVisible() || i < 0 || i >= ui.GMModeCount {
		return
	}
	// Wait for the server's answer before accepting another click on a mode:
	// a burst of toggles would only flood the world with visibility changes.
	if a.clock < a.gmModePendingUntil[i] {
		return
	}
	a.gmModePendingUntil[i] = a.clock + 1
	text := gmModeInfos[i].command
	if i < 2 {
		if a.gmModeActive(i) {
			text += " off"
		} else {
			text += " on"
		}
	}
	if a.gmSend != nil {
		a.gmSend(text)
		return
	}
	if err := a.world.Send(proto.ChatPacket(proto.ChannelNormal, false, text)); err != nil {
		a.backToLogin("Unable to send GM command.")
	}
}

func (a *App) drawGMModes(w, h int) {
	if !a.gmModesVisible() {
		return
	}
	x, y := a.gmModes.Origin(w, h)
	ui.DrawGMModesFrame(a.sprites, a.assets.Texture("interface/QuickPotion.png"), x, y)
	for i := 0; i < ui.GMModeCount; i++ {
		cx, cy := a.gmModes.Cell(w, h, i)
		tint := white
		if !a.gmModeActive(i) {
			tint = [4]float32{.32, .32, .32, 1} // off modes are grey
		}
		if file, uv, ok := ui.SkillIcon(ui.GMModeIcons[i]); ok {
			if tex := a.assets.Texture("interface/" + file); tex != nil {
				a.sprites.Draw(tex, float32(cx), float32(cy), 32, 32, uv.U0, uv.V0, uv.U1, uv.V1, tint)
				continue
			}
		}
		a.sprites.Draw(a.white, float32(cx), float32(cy), 32, 32, 0, 0, 1, 1, tint)
	}
}

func (a *App) drawGMModesTooltip(w, h int) {
	if !a.gmModesVisible() || a.windowDrag.position != nil {
		return
	}
	i := a.gmModes.Hit(w, h, a.lastMouseX, a.lastMouseY)
	if i < 0 {
		return
	}
	state := "off"
	if a.gmModeActive(i) {
		state = "on"
	}
	a.tooltip = []itemDetailLine{{gmModeInfos[i].name + " (" + state + ")", [4]float32{1, 1, 0, 1}}}
	a.tooltipItem = false
	a.tooltipSockets = nil
	a.tooltipDescription = gmModeInfos[i].description
}
