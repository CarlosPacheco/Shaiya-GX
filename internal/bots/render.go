package bots

import (
	"slices"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type visualEffect struct {
	instance *scene.Effect
	point    render.V3
	expires  float32
}

type damageNumber struct {
	point            render.V3
	at               float32
	amount           uint32
	critical, missed bool
}

// DamageNumbers supplies local combat annotations without using the world
// session's queue. A number remains at the point where the hit landed.
func (a *Arena) DamageNumbers(eye render.V3, visit func(point render.V3, age float32, amount uint32, critical, missed bool)) {
	for i := range a.actors {
		n := a.actors[i].number
		age := a.time - n.at
		if age < 1 && (n.amount > 0 || n.missed) && eye.Sub(n.point).Len() < 50 {
			visit(n.point, age, n.amount, n.critical, n.missed)
		}
	}
}

func (a *Arena) effect(file string, sequence int) *scene.Effect {
	if file == "" || a.failedEffects[file] {
		return nil
	}
	fx, err := scene.LoadEffect(a.assets, file, sequence)
	if err != nil {
		a.failedEffects[file] = true
	}
	return fx
}

func (a *Arena) impact(file string, at render.V3, sequence int, duration float32) {
	if fx := a.effect(file, sequence); fx != nil {
		fx.SetTransform(render.Basis(at, render.V3{0, 0, 1}, up))
		a.effects = append(a.effects, visualEffect{fx, at, a.time + duration})
	}
}

func (a *Arena) updateEffects(dt float32, enabled bool) {
	if !enabled {
		clear(a.effects)
		a.effects = a.effects[:0]
		return
	}
	live := a.effects[:0]
	for _, fx := range a.effects {
		if a.time < fx.expires && !fx.instance.Done() {
			a.effectUpdates = append(a.effectUpdates, scene.EffectUpdate{Instance: fx.instance, Delta: dt})
			live = append(live, fx)
		}
	}
	clear(a.effects[len(live):])
	a.effects = live
	scene.UpdateEffects(a.effectUpdates)
	clear(a.effectUpdates)
	a.effectUpdates = a.effectUpdates[:0]
}

func (a *Arena) Draw(s *render.Scene3D, skinner *render.Skinner, effects bool) {
	frustum := render.ViewFrustum(s.View, s.VisibilityProjection())
	a.visible, a.visibleCloth = a.visible[:0], a.visibleCloth[:0]
	for i := range a.actors {
		a.visible = append(a.visible, a.actors[i].avatar)
	}
	scene.CullCrowdCharacters(a.visible, frustum, s)
	a.visible = slices.DeleteFunc(a.visible, func(c *scene.Character) bool { return c == nil })
	for _, character := range a.visible {
		if character.HasCloak() {
			a.visibleCloth = append(a.visibleCloth, character)
		}
	}
	scene.PrepareCrowdCloth(a.visibleCloth, s)
	a.characterBatch.Draw(skinner, s, a.visible)
	if !effects {
		return
	}
	for i := range a.actors {
		hit := &a.actors[i].attack
		if hit.flight != nil && effectVisible(s, frustum, hit.flight.Origin) {
			a.visibleEffects = append(a.visibleEffects, hit.flight)
		}
	}
	for _, fx := range a.effects {
		if effectVisible(s, frustum, fx.point) {
			a.visibleEffects = append(a.visibleEffects, fx.instance)
		}
	}
	scene.DrawEffects(s, a.visibleEffects)
	clear(a.visibleEffects)
	a.visibleEffects = a.visibleEffects[:0]
}

// DrawStats returns the counts from the most recent camera pass.
func (a *Arena) DrawStats() scene.CrowdDrawStats { return a.characterBatch.Stats }

func effectVisible(s *render.Scene3D, frustum render.Frustum, point render.V3) bool {
	// Keep a generous radius for effects that rise or extend around targets.
	return frustum.SphereVisible(point, 12) && render.ClipSphereVisible(s.ClipPlane, point, 12)
}

// Nameplates visits nearby living bots for the client's existing label pass.
func (a *Arena) Nameplates(eye render.V3, visit func(name string, head render.V3, faction int, health float32)) {
	for i := range a.actors {
		f := &a.actors[i]
		if f.avatar == nil || f.hp == 0 || eye.Sub(f.position(a.time)).Len() > 50 {
			continue
		}
		if head, ok := f.avatar.NameAnchor(); ok {
			visit(f.name, head, i%2, float32(f.hp)/maxHP)
		}
	}
}
