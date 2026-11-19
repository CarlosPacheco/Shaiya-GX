package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type castEffect struct {
	source   uint32
	hand     int
	until    float64
	instance *scene.Effect
}

func (a *App) stopCastEffects(source uint32) {
	if a.wv == nil {
		return
	}
	for _, e := range a.wv.castEffects {
		if e.source == source {
			e.instance.Stop()
		}
	}
}

func (a *App) confirmedEffects(p []byte) {
	v := a.wv
	if v == nil {
		return
	}
	switch proto.Op(p) {
	case proto.MobEnter, proto.GXMobEnter:
		// Fresh spawns only: entering an existing population does not replay
		// its appearance. Phoenix's Spawn is native start.EFT, sequence zero.
		if m, err := proto.DecodeMobEnter(p); err == nil && m.New && len(v.impacts) < 32 {
			if x := a.loadSkillEffect("Effect/start.EFT", 0); x != nil {
				matrix := a.mobSpawnTransform(m)
				x.SetTransform(matrix)
				v.impacts = append(v.impacts, skillImpact{instance: x, authoredLifetime: true, fixedWorld: &matrix})
			}
		}
	case proto.MobSkillUse, proto.MobSkillDrain, proto.MobSkillOther:
		a.startMobSkillEffects(a.state.LastSkill)
	case proto.CharacterMove:
		if move, err := proto.DecodeUserMove(p); err == nil && proto.CastInterruptingMotion(move.Motion) {
			a.stopCastEffects(move.ID)
		}
	case proto.CharacterDeath:
		a.stopCastEffects(proto.NewReader(p).U32())
	case proto.SkillUse, proto.SkillUseNoTarget, proto.SkillUserDrain, proto.SkillMobDrain:
		a.stopCastEffects(a.state.LastSkill.Source)
		a.startSkillResultEffects(a.state.LastSkill)
	case proto.GXField:
		a.placeField(a.state.LastField)
	case proto.SkillPartyMember, proto.SkillMobOther:
		a.startSkillSecondaryEffect(a.state.LastSkill)
	case proto.SkillCastStarted:
		r := a.state.LastCast
		a.stopCastEffects(r.Source)
		a.loadSkills()
		d := a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}]
		file := effect.CastLibrary(r.ID)
		until := float64(0) // Unknown duration: wait for the authoritative result.
		if d.ReadyTime > 0 {
			until = a.clock + float64(d.ReadyTime)*.25
		}
		for hand := 0; hand < 2; hand++ {
			x := a.loadSkillEffect(file, 1)
			if x == nil {
				return
			}
			if len(v.castEffects) >= 16 {
				v.castEffects = v.castEffects[1:]
			}
			v.castEffects = append(v.castEffects, castEffect{r.Source, hand, until, x})
		}
	}
}

func (a *App) updateCastEffects() {
	v := a.wv
	a.updateRequestedSkillPresentation()
	for _, e := range v.castEffects {
		if e.until > 0 && a.clock >= e.until {
			e.instance.Stop()
		}
		actor := a.playerActor(e.source)
		matrix, ok := actor.CastHand(e.hand)
		if ok {
			e.instance.SetTransform(matrix)
		} else {
			e.instance.Stop()
		}
		v.effectUpdates = append(v.effectUpdates, scene.EffectUpdate{Instance: e.instance, Delta: float32(a.frameDT)})
	}
	v.updateQueuedEffects()
	alive := v.castEffects[:0]
	for _, e := range v.castEffects {
		if e.instance.Done() {
			continue
		}
		alive = append(alive, e)
	}
	clear(v.castEffects[len(alive):])
	v.castEffects = alive
}

func (a *App) drawCastEffects() {
	if !a.clientSettings().SkillEffects {
		return
	}
	for _, e := range a.wv.castEffects {
		a.wv.visibleEffects = append(a.wv.visibleEffects, e.instance)
	}
	a.wv.drawQueuedEffects(a.scene3d)
}

func (v *worldView) updateQueuedEffects() {
	scene.UpdateEffects(v.effectUpdates)
	clear(v.effectUpdates)
	v.effectUpdates = v.effectUpdates[:0]
}

func (v *worldView) drawQueuedEffects(s *render.Scene3D) {
	scene.DrawEffects(s, v.visibleEffects)
	clear(v.visibleEffects)
	v.visibleEffects = v.visibleEffects[:0]
}

// A spawn placement is a world event, not an attachment to the live monster.
func (a *App) mobSpawnTransform(m proto.MobSpawn) render.M4 {
	p := render.V3{m.X, m.Y, m.Z}
	var floor mobGroundCache
	p = floor.resolve(a.wv.world, p, m.HeightKnown, 0)
	matrix := render.Identity()
	matrix[12], matrix[13], matrix[14] = p[0], p[1], p[2]
	return matrix
}
