package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// The original advances animation cursors at a rate that depends on the actor's
// speed indices (docs/animation-transitions.md, attackspeed/animation.go).

// weaponAttackTime is the weapon definition's own AttackTime, the base the
// character's attack index is measured against (Game.exe 0x41b300).
func (a *App) weaponAttackTime(typ, id uint8) (uint8, bool) {
	if typ == 0 {
		return 0, false
	}
	def := a.itemsItem(typ, id)
	if def == nil {
		return 0, false
	}
	return def.AttackTime, true
}

func (a *App) setSelfAnimationRates(c *scene.Character) {
	sp := a.state.Self.Speed
	if sp == nil {
		return
	}
	weapon := a.state.Inventory[[2]uint8{0, 5}]
	base, have := a.weaponAttackTime(weapon.Type, weapon.TypeID)
	c.SetSpeedTiers(sp.Move, sp.Attack, base, have)
}

func (a *App) setMobAnimationRate(e *game.Entity, actor *scene.CreatureActor) {
	if !e.MobSpeedKnown {
		return
	}
	actor.SetSpeedIndices(e.MobMoveSpeed, e.MobAttackSpeed)
}
