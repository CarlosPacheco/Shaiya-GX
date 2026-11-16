package app

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/skillrange"
)

func (a *App) skillLaunchReach(d sdata.Skill, e *game.Entity) float32 {
	size := float32(0)
	if e != nil && e.Kind == game.KindMob && a.assets != nil {
		if def, err := a.assets.MonsterDefinition(e.TypeID); err == nil {
			size = float32(def.Size)
		}
	}
	return skillrange.Reach(float32(d.AttackRange), size)
}
func (a *App) skillRequestReach(d sdata.Skill, target uint32) bool {
	if a.wv == nil || target == 0 || target == a.state.SelfID || d.Target == 6 {
		return true
	}
	e := a.state.Entities[target]
	// A party member outside the local view is validated by the server.
	if e == nil {
		return true
	}
	goal := a.skillTargetPosition(e)
	if !a.wv.world.SpellLineOfSight(a.wv.pos, goal) {
		a.systemMessage(1, 458, nil)
		return false
	}
	if !skillrange.Within(skillrange.Position(a.wv.pos), skillrange.Position(goal), a.skillLaunchReach(d, e)) {
		a.systemMessage(1, 312, nil)
		return false
	}
	return true
}

// Approach on the horizontal plane while reserving reach for target height.
func (a *App) skillApproachReach(d sdata.Skill, e *game.Entity) float32 {
	reach := a.skillLaunchReach(d, e)
	height := a.skillTargetPosition(e)[1] - a.wv.pos[1]
	return max(.5, float32(math.Sqrt(float64(max(0, reach*reach-height*height)))))
}

// Native 0601 and stationary GX spawns may omit ground height. Their wire
// Y=0 is unknown, not a target under the terrain. Keep current network XZ.
func (a *App) skillTargetPosition(e *game.Entity) render.V3 {
	p := render.V3{e.X, e.Y, e.Z}
	if e.Kind == game.KindMob && !e.MobHeightKnown && a.wv != nil {
		if m := a.wv.mobs[e.ID]; m != nil && m.entity == e && m.position.initialized {
			p[1] = m.pos[1]
		} else if a.wv.world != nil && a.wv.world.HasTerrain() {
			p[1] = a.wv.world.Height(e.X, e.Z)
		}
	}
	return p
}
