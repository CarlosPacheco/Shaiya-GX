package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"math"
)

// GX rule: six safe units; half the previous quadratic damage, capped at max HP.
func fallDamage(drop float32, maxHP uint32) uint32 {
	if drop <= 6 || math.IsNaN(float64(drop)) || math.IsInf(float64(drop), 0) {
		return 0
	}
	fraction := float64(drop-6) / 24
	return uint32(math.Ceil(float64(maxHP) * min(.5*fraction*fraction, 1)))
}
func (s *Session) applyFallDamage(drop float32) {
	if !config.FallDamageEnabled() || s.char == nil || s.dead || s.invulnerable() {
		return
	}
	st := s.stats()
	damage := min(st.CurrentHP, fallDamage(drop, st.MaxHP))
	if damage == 0 {
		return
	}
	s.commitMovement()
	s.cancelMove()
	s.breakStealth()
	s.char.CurrentHP = satSub(s.char.CurrentHP, damage)
	s.persist()
	w := packet.New(packet.Opcode(proto.GXFallDamage))
	w.U32(s.char.ID)
	w.U32(damage)
	frame := w.MustFinish()
	s.send(frame, "GX fall damage")
	s.sendToViewers(frame, "GX fall damage (view)")
	current := s.stats()
	s.send(buildCurrentHitpoints(&current), "current hp after fall")
	if current.CurrentHP > 0 {
		s.wake()
		return
	}
	s.damageLedger = nil
	state, death := buildCharacterDeathState(s.char.ID), buildCharacterDeath(s.char.ID, deathTypeMob, 0)
	s.send(state, "fall death state")
	s.send(death, "fall death")
	s.sendToViewers(state, "fall death state (view)")
	s.sendToViewers(death, "fall death (view)")
	if s.player != nil {
		s.player.SetAlive(false)
		duelLeave(s.player, 3, true)
		guildDuelLeave(s.player, nil, 3)
	}
	s.dead, s.autoAttack, s.cast = true, false, nil
	s.clearBuffsOnDeath()
	// Environmental deaths grant no PvP credit, item/XP loss or Ultimate deletion.
	s.deathLoss = -1
}
