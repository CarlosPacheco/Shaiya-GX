package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *App) loadBlessing() {
	if a.blessingLoaded || a.arc == nil {
		return
	}
	a.blessingLoaded = true
	raw, err := a.arc.ReadFile("Character/KillStatus.SData")
	if err == nil {
		a.blessingTiers, err = sdata.ParseBlessing(raw)
	}
	if err != nil {
		log.Printf("blessing data: %v", err)
		return
	}
	for c := range a.blessingMax {
		a.blessingMax[c] = sdata.BlessingMaximum(a.blessingTiers, uint8(c))
	}
}

// Original 0055fc90 starts ten minutes only on a maximum 020F broadcast.
// The 0211 entry snapshot restores a remaining time without activation feedback.
func (a *App) confirmedBlessing(op proto.Opcode) {
	if op == proto.Logout {
		a.blessingBannerUntil = 0
		a.blessingFeedbackActive = false
		return
	}
	if op != proto.CountryKillCount && op != proto.BlessInit || a.state == nil {
		return
	}
	a.loadBlessing()
	b := &a.state.Blessing
	if b.Country > 1 || b.Country != a.state.Faction.Faction || a.blessingMax[b.Country] == 0 {
		return
	}
	maximum := b.Count >= a.blessingMax[b.Country]
	if op == proto.BlessInit {
		a.blessingFeedbackActive = maximum
		return
	}
	if !maximum {
		a.blessingFeedbackActive = false
		return
	}
	// TCP normally delivers one activation. A repeated maximum notification
	// must not replay the burst or extend the countdown, including after reentry.
	if a.blessingFeedbackActive {
		return
	}
	a.blessingFeedbackActive = true
	b.RemainingMS = 600000
	a.blessingBannerUntil = a.clock + 5
	if a.sound != nil {
		name := a.audioPath("ui_etainbless01.wav", false)
		a.audioError(name, a.sound.Play(name, 1))
	}
	a.startBlessingEffect(b.Country)
}

// Original 0055fc90 attaches type 15, sequence country, to the local actor.
// Every winning-country client does this for itself; it is never replicated
// onto remote rigs. Allow this rare event over the ordinary 32-impact budget
// so a busy PvP frame cannot swallow the faction activation.
func (a *App) startBlessingEffect(country uint8) {
	if a.wv == nil || a.assets == nil || a.state.SelfID == 0 {
		return
	}
	if x := a.loadSkillEffect("Effect/bless.EFT", int(country)); x != nil {
		if m, ok := a.effectTargetMatrix(a.state.SelfID); ok {
			x.SetTransform(m)
		}
		a.wv.impacts = append(a.wv.impacts, skillImpact{target: a.state.SelfID, instance: x, authoredLifetime: true})
	}
}

// Native 00484be0 lists the nonzero cumulative types in numeric order using
// SysMsg 5127..5153. It does not print amounts or an invented title.
func (a *App) blessingLines() []string {
	if a.state == nil || !a.state.Blessing.Known {
		return nil
	}
	a.loadBlessing()
	b := a.state.Blessing
	values := sdata.BlessingBonuses(a.blessingTiers, b.Country, b.Count)
	var lines []string
	for typ := 1; typ < len(values); typ++ {
		if values[typ] != 0 {
			lines = append(lines, a.msg.Get(5126+typ))
		}
	}
	return lines
}

func (a *App) drawBlessingTooltip(w, h int) {
	if a.bottom == nil || !a.bottom.BlessingAt(a.lastMouseX, a.lastMouseY, w, h) || a.menuOpen || a.state == nil || a.state.Self.Dead {
		return
	}
	for _, dialog := range []*ui.Message{a.accountMessage, a.partyMessage, a.guildMessage, a.guildConfirm, a.guildDuelMessage, a.duelMessage, a.gemConfirm, a.gateConfirm, a.questConfirm, a.exchangeInvite} {
		if dialog != nil && dialog.Visible {
			return
		}
	}
	if a.appearanceEdit != nil || a.shop.quantity != nil || a.exchange.quantity != nil || a.exchange.moneyOpen || a.duel.quantity != nil || a.duel.moneyOpen || a.blacksmith.quantity {
		return
	}
	if a.barCovered(&platform.Input{MouseX: a.lastMouseX, MouseY: a.lastMouseY}, w, h) {
		return
	}
	lines := a.blessingLines()
	if !a.state.Blessing.Known || a.state.Blessing.Country > 1 || a.blessingMax[a.state.Blessing.Country] == 0 {
		return
	}
	// Original panel: 200px, 15px per line plus one row of top padding,
	// ARGB CD1D1F24, no border. Position follows the pointer above the bar.
	height := (len(lines) + 1) * 15
	left := max(0, min(a.lastMouseX, w-200)-10)
	top := max(0, a.lastMouseY-height)
	a.sprites.Draw(a.white, float32(left), float32(top), 200, float32(height), 0, 0, 1, 1, [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 205.0 / 255})
	f := a.tooltipFont
	if f == nil {
		f = a.font
	}
	for i, line := range lines {
		f.Draw(a.sprites, line, left+10, top+15+i*15, white, 0)
	}
}

func (a *App) drawBlessingBanner(w, h int) {
	if a.assets == nil || a.clock >= a.blessingBannerUntil {
		return
	}
	if tex := a.assets.Texture("interface/worldname/event.png"); tex != nil {
		remaining := a.blessingBannerUntil - a.clock
		alpha := float32(min(1, remaining, 5-remaining))
		a.sprites.Draw(tex, float32(w/2-128), float32(h/3-64), 256, 64, 0, 0, 1, 1, [4]float32{1, 1, 1, alpha})
	}
}
