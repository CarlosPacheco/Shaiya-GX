package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func (a *App) autoAttackNearest() bool {
	target := a.state.Target
	e := a.state.Entities[target]
	if e == nil || e.Dead || e.ID == a.state.SelfID || e.Kind != game.KindMob && e.Kind != game.KindUser {
		target = 0
		best := float32(50 * 50)
		for id, mob := range a.state.Entities {
			if mob.Kind != game.KindMob || mob.Dead || mob.HealthKnown && mob.HP == 0 {
				continue
			}
			dx, dz := mob.X-a.wv.pos[0], mob.Z-a.wv.pos[2]
			if d := dx*dx + dz*dz; d < best || d == best && (target == 0 || id < target) {
				target, best = id, d
			}
		}
	}
	if target == 0 {
		return true
	}
	if target != a.state.Target && a.state.Entities[target].Kind == game.KindMob {
		if err := a.world.Send(proto.TargetMobPacket(target)); err != nil {
			a.backToLoginID(12003)
			return false
		}
	}
	a.state.Target = target
	if !a.followOnly(a.state.Entities[target]) && a.deferAirAction(target, nil) {
		return true
	}
	a.wv.autoRun = false
	a.wv.approachSkill = nil
	a.wv.approachNPC = false
	a.wv.approachTarget, a.wv.approachLoot = target, false
	return a.updateActionApproach(&platform.Input{})
}

func (a *App) approachLoot(id uint32) bool {
	a.wv.autoRun = false
	a.wv.approachSkill = nil
	a.wv.approachNPC = false
	a.wv.approachTarget, a.wv.approachLoot = id, true
	return a.updateActionApproach(&platform.Input{})
}

// Approach is visual prediction only. The request is sent once in reach and
// server packets remain responsible for attacks, pickup and item removal.
func (a *App) updateActionApproach(in *platform.Input) bool {
	v := a.wv
	if v.approachTarget == 0 || in.SuppressAutoRun {
		return true
	}
	if in.Clicked {
		v.approachTarget = 0
		return true
	}
	if a.state.Self.Dead || in.Held('W') || in.Held('S') || in.Held('A') || in.Held('D') || in.Held('Q') || in.Held('E') || in.HasKey(platform.KeyNumLock) {
		v.approachTarget, v.travel = 0, false
		return true
	}
	var goal render.V3
	// Airborne: keep chasing, but the attack or skill waits for the landing.
	airborne := false
	reach := float32(1.5)
	if v.approachNPC {
		e := a.state.Entities[v.approachTarget]
		if e == nil || e.Kind != game.KindNpc {
			v.approachTarget, v.travel = 0, false
			return true
		}
		goal = a.visualEntityPosition(e)
		reach = 2
	} else if v.approachLoot {
		item, ok := a.state.GroundItems[v.approachTarget]
		if !ok {
			v.approachTarget, v.travel = 0, false
			return true
		}
		goal = render.V3{item.X, item.Y, item.Z}
		// Native 0x43bfa0 sends 0x0205 within one horizontal unit.
		reach = 1
	} else {
		e := a.state.Entities[v.approachTarget]
		if e == nil || e.Dead && (len(v.approachSkill) == 0 || e.Kind == game.KindMob) || a.state.Target != v.approachTarget {
			v.approachTarget, v.travel = 0, false
			return true
		}
		airborne = !a.attackMovementAllowed() && !a.followOnly(e)
		goal = render.V3{e.X, e.Y, e.Z}
		if e.Kind == game.KindMob {
			goal = a.visualEntityPosition(e)
		}
		reach = a.attackReachTo(e)
		if len(v.approachSkill) > 0 {
			var def sdata.Skill
			var ok bool
			if proto.Op(v.approachSkill) == proto.GXEvolutionCast {
				def, ok = a.evolutionCastDefinition(v.approachSkill)
			} else {
				learned := v.approachLearned
				current, known := a.state.Skills[learned.Number]
				def = a.skillDefs[sdata.SkillKey{ID: learned.ID, Level: learned.Level}]
				ok = known && current.ID == learned.ID && current.Level == learned.Level && a.skills.cooldowns[current.ID] <= a.clock
			}
			if !ok {
				v.approachTarget, v.travel = 0, false
				return true
			}
			reach = a.skillApproachReach(def, e)
		}
	}
	delta := goal.Sub(v.pos)
	delta[1] = 0
	if delta.Len() > reach {
		v.travel = true
		v.destination = goal.Sub(delta.Norm().Scale(reach * .97))
		if v.engagedID == v.approachTarget {
			v.engagedID = 0 // the target left reach: the chase attacks again on arrival
		}
		return true
	}
	if airborne {
		v.travel = false
		return true
	}

	if v.approachNPC {
		e := a.state.Entities[v.approachTarget]
		dx, dz := e.X-v.pos[0], e.Z-v.pos[2]
		if dx*dx+dz*dz >= 9.5*9.5 {
			v.travel = false
			return true
		}
	}
	// An attack clicked during the previous swing's movement pause stays
	// queued at the target until it can actually be requested.
	if !v.approachNPC && !v.approachLoot && len(v.approachSkill) == 0 && a.clock < v.attackMoveUntil {
		v.travel = false
		return true
	}
	plain := !v.approachNPC && !v.approachLoot && len(v.approachSkill) == 0
	if plain && v.engagedID == v.approachTarget {
		// Already attacking: keep following (0x43bfa0 state 3 stays active
		// while the target lives) and swing again only after it left reach.
		v.travel = false
		return true
	}
	if plain && a.clock-v.engagedAt < .25 {
		v.travel = false
		return true
	}
	id, loot := v.approachTarget, v.approachLoot
	v.approachTarget, v.travel = 0, false
	if v.approachNPC {
		v.approachNPC = false
		a.openNPC(id)
		return a.state != nil
	}
	if loot {
		if err := a.world.Send(proto.PickupPacket(id)); err != nil {
			a.backToLoginID(12004)
			return false
		}
		return true
	}
	a.state.Target = id
	if len(v.approachSkill) > 0 {
		packet := v.approachSkill
		v.approachSkill = nil
		if proto.Op(packet) == proto.GXEvolutionCast {
			if def, ok := a.evolutionCastDefinition(packet); ok && a.validSkillTarget(def, id) && a.skillResourcesAvailable(def) {
				return a.sendEvolutionCast(packet, id)
			}
			return true
		}
		def := a.skillDefs[sdata.SkillKey{ID: v.approachLearned.ID, Level: v.approachLearned.Level}]
		if !a.validSkillTarget(def, id) || !a.skillResourcesAvailable(def) {
			return true
		}
		if !a.faceRequestedTarget(id) {
			return false
		}
		if !a.prepareSkillMovement(packet) {
			return true
		}
		if err := a.world.Send(packet); err != nil {
			a.backToLoginID(12005)
			return false
		}
		a.requestedSkillPresentation(packet)
		return true
	}
	return a.attackTarget()
}

// attackReachTo is the horizontal centre distance at which the original stops
// chasing (0x43bfa0): bare hands 1.0; otherwise weapon range plus the mob's
// Size minus 0.3, or plus 0.5 against players.
func (a *App) attackReachTo(e *game.Entity) float32 {
	weapon := a.state.Inventory[[2]uint8{0, 5}]
	def := a.itemsItem(weapon.Type, weapon.TypeID)
	if a.followOnly(e) {
		return 1.5
	}
	if def == nil {
		return 1
	}
	reach := float32(def.Range)
	if e != nil && e.Kind == game.KindMob {
		if md, err := a.assets.MonsterDefinition(e.TypeID); err == nil {
			reach += float32(md.Size) - .3
		}
	} else {
		reach += .5
	}
	return max(reach, 1)
}

func (a *App) attackReach() float32 {
	reach := float32(2)
	weapon := a.state.Inventory[[2]uint8{0, 5}]
	if def := a.itemsItem(weapon.Type, weapon.TypeID); def != nil {
		reach = max(reach, float32(def.Range))
	}
	return reach
}

func (a *App) approachSkillPacket(packet []byte) bool {
	if a.wv == nil || a.state.Self.Mounted || len(packet) != 7 || proto.Op(packet) != proto.SkillUse && proto.Op(packet) != proto.SkillUseNoTarget {
		return false
	}
	r := proto.NewReader(packet)
	number, target := r.U8(), r.U32()
	if a.deferAirAction(target, packet) {
		return true
	}
	// A new command replaces any queued approach, including when the new
	// spell is already in range. Otherwise the old spell could fire later.
	a.wv.approachTarget, a.wv.approachSkill = 0, nil
	a.wv.approachNPC = false
	a.wv.travel, a.wv.autoRun = false, false
	if !a.faceRequestedTarget(target) {
		return true
	}
	if target == 0 || target == a.state.SelfID {
		return false
	}
	e := a.state.Entities[target]
	if e == nil {
		return false
	}
	learned, ok := a.state.Skills[number]
	if !ok {
		return false
	}
	def := a.skillDefs[sdata.SkillKey{ID: learned.ID, Level: learned.Level}]
	delta := render.V3{e.X - a.wv.pos[0], 0, e.Z - a.wv.pos[2]}
	if delta.Len() <= a.skillApproachReach(def, e) {
		return false
	}
	a.wv.autoRun = false
	a.wv.approachTarget, a.wv.approachLoot = target, false
	a.wv.approachSkill, a.wv.approachLearned = packet, learned
	a.updateActionApproach(&platform.Input{})
	return true
}
