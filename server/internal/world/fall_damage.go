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
	if !config.FallDamageEnabled() || s.char == nil || s.dead || s.invulnerable() || s.turboActive() {
		return
	}
	st := s.stats()
	s.applyEnvironmentalDamage(fallDamage(drop, st.MaxHP), proto.GXFallDamage, "fall")
}

func (s *Session) applyEnvironmentalDamage(amount uint32, opcode proto.Opcode, cause string) {
	if s.char == nil || s.dead || s.invulnerable() {
		return
	}
	damage := min(s.char.CurrentHP, amount)
	if damage == 0 {
		return
	}
	s.cancelTravelCast()
	s.commitMovement()
	s.cancelMove()
	s.breakStealth()
	s.char.CurrentHP = satSub(s.char.CurrentHP, damage)
	s.persist()
	w := packet.New(packet.Opcode(opcode))
	w.U32(s.char.ID)
	w.U32(damage)
	frame := w.MustFinish()
	s.send(frame, "GX "+cause+" damage")
	s.sendToViewers(frame, "GX "+cause+" damage (view)")
	current := s.stats()
	s.send(buildCurrentHitpoints(&current), "current hp after "+cause)
	if current.CurrentHP > 0 {
		s.wake()
		return
	}
	s.damageLedger = nil
	state, death := buildCharacterDeathState(s.char.ID), buildCharacterDeath(s.char.ID, deathTypeMob, 0)
	s.send(state, cause+" death state")
	s.send(death, cause+" death")
	s.sendToViewers(state, cause+" death state (view)")
	s.sendToViewers(death, cause+" death (view)")
	if s.player != nil {
		s.player.SetAlive(false)
		duelLeave(s.player, 3, true)
		guildDuelLeave(s.player, nil, 3)
	}
	s.dead, s.autoAttack, s.cast = true, false, nil
	s.clearBuffsOnDeath()
	// Environmental deaths grant no PvP credit, item or XP loss.
	s.deathLoss = -1
}
