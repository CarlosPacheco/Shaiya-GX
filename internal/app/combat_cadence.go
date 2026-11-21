package app

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
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
	return ok && (!a.skills.actionKnown || a.clock-a.skills.lastActionAt >= max(0, interval.Seconds()-early))
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
	a.skills.lastActionAt, a.skills.actionKnown = a.clock, true
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
