package app

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/bots"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func (a *App) botsCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "/bots") {
		return false
	}
	if a.state == nil {
		return true
	}
	if !a.state.Self.IsGM {
		a.questMessage("GM: access denied.")
		return true
	}
	if len(fields) == 2 && strings.EqualFold(fields[1], "off") {
		if a.wv != nil {
			a.wv.clearBots()
		}
		a.questMessage("Local PvP bots removed.")
		return true
	}
	if len(fields) == 2 {
		count, err := strconv.Atoi(fields[1])
		if err == nil && count >= 1 && count <= bots.MaxCount {
			if a.wv == nil || a.wv.world == nil || a.arc == nil {
				a.questMessage("Enter a map to start local PvP bots.")
				return true
			}
			a.wv.clearBots()
			a.wv.bots = bots.New(a.arc, a.wv.world, a.wv.pos, a.wv.facing, count, rand.Uint64())
			a.questMessage(fmt.Sprintf("Loading %d local PvP bots.", count))
			return true
		}
	}
	a.questMessage("Usage: /bots 1-1000 or /bots off")
	return true
}

func (v *worldView) clearBots() {
	if v.bots != nil {
		v.bots.Close()
		v.bots = nil
	}
}

func (a *App) updateBots() {
	v := a.wv
	if v.bots == nil {
		return
	}
	if a.state == nil || !a.state.Self.IsGM {
		v.clearBots()
		return
	}
	if v.bots.Update(float32(a.frameDT), a.clientSettings().SkillEffects) {
		a.questMessage(fmt.Sprintf("%d local PvP bots ready.", v.bots.Count()))
	}
}

func (a *App) drawBotNameplates(vp render.M4, w, h int, label func(string, render.V3, [4]float32, int, ...playerPlate) (int, int, bool)) {
	a.wv.bots.Nameplates(a.wv.pos, func(name string, pos render.V3, faction int, health float32) {
		color := [4]float32{.42, .74, 1, 1}
		if faction == 1 {
			color = [4]float32{1, .45, .43, 1}
		}
		if left, right, drawn := label(name, pos, color, 0); drawn && a.white != nil {
			_, y, _, _ := projectLabel(pos, vp, w, h)
			x, width := float32(left), float32(right-left)
			a.sprites.DrawFlat(a.white, x, float32(y), width, 3, [4]float32{0, 0, 0, .8})
			a.sprites.DrawFlat(a.white, x, float32(y), width*health, 2, color)
		}
	})
}
