package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type evolutionView struct {
	generation uint32
	received   float64
}

func (a *App) confirmEvolutionBar() {
	next := a.state.Evolution.Generation
	if a.evolution.generation != next {
		a.skills.queuedSkill = nil
		if a.wv != nil {
			if len(a.wv.approachSkill) != 0 {
				a.wv.approachSkill, a.wv.approachTarget, a.wv.travel = nil, 0, false
			}
		}
	}
	a.evolution = evolutionView{generation: next, received: a.clock}
}

func (a *App) evolutionOrigin(w, h int) (int, int) {
	x, y := ui.QuickPotionOrigin(a.quickPotions.Position, w, h)
	if y >= ui.QuickPotionHeight+4 {
		return x, y - ui.QuickPotionHeight - 4
	}
	return x, min(y+ui.QuickPotionHeight+4, h-ui.QuickPotionHeight)
}

func (a *App) evolutionCooldown(action proto.EvolutionAction) float64 {
	return max(0, float64(action.LeftMS)/1000-(a.clock-a.evolution.received))
}

func (a *App) updateEvolutionBar(in *platform.Input, w, h int) bool {
	if a.state.Evolution.Generation == 0 || a.state.Self.Dead {
		return false
	}
	x, y := a.evolutionOrigin(w, h)
	if in.MouseX < x || in.MouseX >= x+ui.QuickPotionWidth || in.MouseY < y || in.MouseY >= y+ui.QuickPotionHeight {
		return false
	}
	cell := ui.QuickPotionCell(in.MouseX, in.MouseY, x, y)
	if !in.Clicked || cell < 0 || a.world == nil || a.state.Self.Mounted {
		return true
	}
	action := a.state.Evolution.Actions[cell]
	d, ok := a.skillDefinition(action.Skill, 100)
	if !ok || action.Skill == 0 || a.evolutionCooldown(action) > 0 || !a.skillResourcesAvailable(d) {
		return true
	}
	target := a.state.SelfID
	if d.Effect == 5 {
		target = 0
	} else if d.Effect != 2 {
		target = a.resolveSkillTarget(d)
		if !a.validSkillTarget(d, target) {
			return true
		}
	}
	packet := proto.EvolutionCastPacket(a.state.Evolution.Generation, uint8(cell), target)
	if a.wv != nil && a.approachEvolutionCast(packet, target, action.Range, d) {
		return true
	}
	a.sendEvolutionCast(packet, target)
	return true
}

func (a *App) evolutionCastDefinition(packet []byte) (sdata.Skill, bool) {
	r := proto.NewReader(packet)
	generation, cell := r.U32(), r.U8()
	if generation == 0 || generation != a.state.Evolution.Generation || cell >= 2 || a.state.Self.Mounted {
		return sdata.Skill{}, false
	}
	action := a.state.Evolution.Actions[cell]
	d, ok := a.skillDefinition(action.Skill, 100)
	d.AttackRange = action.Range
	return d, ok && a.evolutionCooldown(action) == 0
}

func (a *App) approachEvolutionCast(packet []byte, target uint32, reach uint8, d sdata.Skill) bool {
	v := a.wv
	v.approachTarget, v.approachSkill, v.approachNPC, v.approachLoot = 0, nil, false, false
	v.travel, v.autoRun = false, false
	e := a.state.Entities[target]
	if e == nil || target == 0 || target == a.state.SelfID {
		return false
	}
	d.AttackRange = reach
	delta := render.V3{e.X - v.pos[0], 0, e.Z - v.pos[2]}
	if delta.Len() <= a.skillApproachReach(d, e) {
		return false
	}
	v.approachTarget, v.approachSkill = target, packet
	// The world input loop will finish this approach through the usual reach path.
	a.updateActionApproach(&platform.Input{})
	return true
}

func (a *App) sendEvolutionCast(packet []byte, target uint32) bool {
	if a.world == nil || !a.faceRequestedTarget(target) {
		return false
	}
	if a.wv != nil && !a.stopForAttack() {
		return false
	}
	if err := a.world.Send(packet); err != nil {
		a.backToLogin("Unable to cast monster skill.")
		return false
	}
	return true
}

func (a *App) drawEvolutionBar(w, h int) {
	if a.state.Evolution.Generation == 0 || a.state.Self.Dead {
		return
	}
	var slots [2]ui.ActionSlot
	for i, action := range a.state.Evolution.Actions {
		if action.Skill == 0 {
			continue
		}
		d, ok := a.skillDefinition(action.Skill, 100)
		if !ok {
			continue
		}
		slot := &slots[i]
		slot.Cooldown, slot.CooldownDuration = a.evolutionCooldown(action), float64(action.CooldownMS)/1000
		if file, uv, ok := ui.SkillIcon(d.Icon); ok {
			slot.Icon, slot.UV = a.assets.Texture("interface/"+file), uv
		}
	}
	x, y := a.evolutionOrigin(w, h)
	ui.DrawEvolutionBar(a.sprites, a.font, a.assets.Texture("interface/QuickPotion.png"), a.white, x, y, slots)
	if cell := ui.QuickPotionCell(a.lastMouseX, a.lastMouseY, x, y); cell >= 0 {
		if id := a.state.Evolution.Actions[cell].Skill; id != 0 {
			a.queueSkillTooltip(sdata.SkillKey{ID: id, Level: 100})
		}
	}
}
