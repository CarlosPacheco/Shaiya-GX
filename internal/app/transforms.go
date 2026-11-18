package app

import (
	"log"

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

func (a *App) prepareTransforms() {
	v := a.wv
	if v == nil {
		return
	}
	for id := range v.transforms {
		if _, ok := a.state.Transforms[id]; !ok {
			if ch := a.transformedCharacter(id); ch != nil {
				ch.Hidden = false
			}
			delete(v.transforms, id)
		}
	}
	budget := 2
	for id, mob := range a.state.Transforms {
		ch := a.transformedCharacter(id)
		if ch == nil {
			continue
		}
		t := v.transforms[id]
		if t == nil || t.typeID != mob {
			actor, err := a.assets.BuildMonster(mob)
			if err != nil || actor == nil {
				log.Printf("transformation into monster %d: %v", mob, err)
				continue
			}
			if v.transforms == nil {
				v.transforms = map[uint32]*transformActor{}
			}
			t = &transformActor{typeID: mob, actor: actor}
			v.transforms[id] = t
		}
		ch.Hidden = true
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
		t.actor.PrepareEffects(float32(a.frameDT), a.clientSettings().SkillEffects, true, &budget)
	}
}

func (a *App) drawTransforms() {
	v := a.wv
	if v == nil || len(v.transforms) == 0 {
		return
	}
	frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.VisibilityProjection())
	for _, t := range v.transforms {
		if t.actor.Visible(frustum, a.scene3d.ClipPlane) {
			t.actor.Draw(a.skinner, a.scene3d)
		}
	}
}
