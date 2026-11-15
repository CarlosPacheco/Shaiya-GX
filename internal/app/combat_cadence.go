package app

import "github.com/Spelunkern/Shaiya-GX/internal/skilltiming"

func (a *App) combatActionReady() bool {
	sp := a.state.Self.Speed
	if sp == nil {
		return !a.skills.actionKnown
	}
	interval, ok := skilltiming.Interval(sp.Attack)
	return ok && (!a.skills.actionKnown || a.clock-a.skills.lastActionAt >= interval.Seconds())
}
func (a *App) markCombatAction() {
	a.skills.lastActionAt, a.skills.actionKnown = a.clock, true
}
