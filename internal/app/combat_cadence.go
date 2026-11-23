package app

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
)

// skillSendEarly lets a skill leave slightly before the interval ends: the
// server keeps the latest early request and runs it exactly at its deadline, so
// network latency and its own tick no longer add to every cycle.
const skillSendEarly = .15

func (a *App) combatActionReady() bool { return a.actionReadyWithin(0) }

func (a *App) actionReadyWithin(early float64) bool {
	sp := a.state.Self.Speed
	if sp == nil {
		return !a.skills.actionKnown
	}
	interval, ok := a.actionInterval(sp.Attack)
	// The original keeps the action busy until its animation ends
	// (0x40ff40), so a skill's own clip is a floor the early send cannot cut.
	return ok && (!a.skills.actionKnown || a.clock-a.skills.lastActionAt >= max(0, interval.Seconds()-early, a.skills.actionFloor))
}

// actionInterval mirrors the server: the table, capped by the basic-attack clip
// at the animation rate of the equipped weapon.
func (a *App) actionInterval(tier uint8) (time.Duration, bool) {
	weapon := a.state.Inventory[[2]uint8{0, 5}]
	base, have := a.weaponAttackTime(weapon.Type, weapon.TypeID)
	return skilltiming.IntervalFor(tier, base, have)
}
func (a *App) markCombatAction() {
	a.skills.lastSkillID = 0
	a.skills.actionFloor = 0
	a.skills.lastActionAt, a.skills.actionKnown = a.clock, true
}

// skillClipFloor is how long the skill's own animation runs. Weapon swings are
// already bounded by the basic-attack cadence and timed casts by their cast.
func (a *App) skillClipFloor(d sdata.Skill) float64 {
	if d.ReadyTime > 0 || a.wv == nil || a.wv.self == nil {
		return 0
	}
	slot := scene.SkillAction(d.Animation, d.Attack, d.Effect, a.playerWeapon(a.state.SelfID), 0)
	if slot != 87 && slot != 90 && slot != 93 && (slot < 100 || slot > 110) {
		return 0
	}
	return float64(a.wv.self.ActionSeconds(slot))
}

func (a *App) skillActionReady(id uint16, d sdata.Skill) bool {
	if a.wv != nil && a.wv.dashes[a.state.SelfID].Source != 0 {
		return false
	}
	if a.actionReadyWithin(skillSendEarly) {
		return true
	}
	return a.state.Self.Speed != nil && a.state.Self.Speed.Attack != 255 && a.skills.lastSkillAt == a.skills.lastActionAt && a.clock-a.skills.lastSkillAt >= skilltiming.SpinContact.Seconds() && a.clock-a.skills.lastSkillAt < skilltiming.SpinDuration.Seconds() && skilltiming.AreaCombo(a.skills.lastSkillID, id, d.Target, d.Effect, d.ReadyTime)
}
