package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// transformActor is the monster a transformed character (Degeneration) is shown
// as. The character's own model is hidden and the creature takes its place:
// same position and heading, walking while the character moves.
type transformActor struct {
	typeID  uint16
	actor   *scene.CreatureActor
	last    render.V3
	movedAt float64
}

func (a *App) transformedCharacter(id uint32) *scene.Character {
	v := a.wv
	if id == a.state.SelfID {
		return v.self
	}
	if p := v.actors[id]; p != nil {
		return p.character
	}
	return nil
}

func (a *App) formMonster(id uint32) uint16 {
	if mob := a.state.Transforms[id]; mob != 0 {
		return mob
	}
	if form := a.state.PlayerForms[id]; form.Kind == proto.FormEvolution {
		if d, ok := a.skillDefinition(form.Skill, form.Rank); ok {
			return d.ChangeLevel // the visual model belongs to the client catalog
		}
	}
	return 0
}

func (a *App) prepareTransforms() {
	v := a.wv
	for id := range v.transforms {
		if a.formMonster(id) == 0 {
			if ch := a.transformedCharacter(id); ch != nil {
				ch.Hidden = false
			}
			delete(v.transforms, id)
		}
	}
	budget := 2
	for id, mob := range a.state.Transforms {
		a.prepareTransform(id, mob, &budget)
	}
	for id, form := range a.state.PlayerForms {
		if form.Kind == proto.FormEvolution {
			a.prepareTransform(id, a.formMonster(id), &budget)
		}
	}
}

func (a *App) prepareTransform(id uint32, mob uint16, budget *int) {
	ch := a.transformedCharacter(id)
	if ch == nil || mob == 0 {
		return
	}
	v := a.wv
	t := v.transforms[id]
	if t == nil || t.typeID != mob {
		actor, err := a.assets.BuildMonster(mob)
		if err != nil {
			log.Printf("transformation into monster %d: %v", mob, err)
		}
		if v.transforms == nil {
			v.transforms = map[uint32]*transformActor{}
		}
		t = &transformActor{typeID: mob, actor: actor}
		v.transforms[id] = t
	}
	ch.Hidden = t.actor != nil
	if t.actor == nil {
		return
	}
	pos := render.V3{ch.World[12], ch.World[13], ch.World[14]}
	if pos.Sub(t.last).Len() > .005 {
		t.movedAt = a.clock
	}
	t.last = pos
	t.actor.SetTransform(ch.World)
	action := 6 // idle
	if a.clock-t.movedAt < .2 {
		action = 0 // walk
	}
	if err := t.actor.SetAction(action); err != nil {
		a.assets.Missing[err.Error()] = true
	}
	t.actor.Update(float32(a.frameDT))
	t.actor.PrepareEffects(float32(a.frameDT), a.clientSettings().SkillEffects, true, budget)
}

func (a *App) playFormAttack(id uint32, slot uint8) bool {
	if t := a.wv.transforms[id]; t != nil && t.actor != nil {
		t.actor.PlayOnce(t.actor.AttackAction(slot))
		return true
	}
	return false
}

func (a *App) evolutionSkillEffects(r proto.SkillResult, animate bool) bool {
	form := a.state.PlayerForms[r.Source]
	if a.wv == nil || r.Level != 100 || form.Kind != proto.FormEvolution {
		return false
	}
	if animate && r.Result <= 4 {
		a.faceSkillTarget(r.Source, r.Target)
		for _, attack := range form.Attacks {
			if attack.Skill == r.ID {
				a.playFormAttack(r.Source, attack.Slot)
				break
			}
		}
	}
	a.startMobSkillEffects(r, false)
	return true
}

func (a *App) drawTransforms() {
	v := a.wv
	if v == nil || len(v.transforms) == 0 {
		return
	}
	frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.VisibilityProjection())
	for _, t := range v.transforms {
		if t.actor != nil && t.actor.Visible(frustum, a.scene3d.ClipPlane) {
			t.actor.Draw(a.skinner, a.scene3d)
		}
	}
}
