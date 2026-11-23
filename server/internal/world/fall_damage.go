package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"math"
	"time"
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
	// Only GM turbo cancels fall damage; "attack on" does not protect from it.
	if !config.FallDamageEnabled() || s.char == nil || s.dead || s.turboActive() {
		return
	}
	st := s.stats()
	s.applyEnvironmentalDamage(fallDamage(drop, st.MaxHP), proto.GXFallDamage, "fall")
}

func (s *Session) applyEnvironmentalDamage(amount uint32, opcode proto.Opcode, cause string) {
	// Environmental damage ignores GM immunity: "attack on" never stops drowning.
	if s.char == nil || s.dead {
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
	if opcode == proto.GXDrowningDamage {
		notice := packet.New(packet.Opcode(proto.GXDrowningDeath))
		notice.U32(s.char.ID)
		frame := notice.MustFinish()
		s.send(frame, "GX drowning death")
		s.sendToViewers(frame, "GX drowning death (view)")
		s.floatDrownedBody()
	}
}

// drownedFloatY is the surface swimming pose: a drowned body and its spirit
// rise to it, steadily, instead of staying where the breath ran out.
const (
	drownedFloatY    = float32(-1.5)
	drownedRiseSpeed = float32(1.2) // units per second
)

// floatDrownedBody starts the rise; tickDrownedFloat advances it.
func (s *Session) floatDrownedBody() {
	if s.char != nil && s.char.PosY < drownedFloatY {
		s.drownedRiseAt = time.Now()
	}
}

func (s *Session) tickDrownedFloat(now time.Time) {
	if s.drownedRiseAt.IsZero() {
		return
	}
	ch := s.char
	if ch == nil || !s.dead || ch.PosY >= drownedFloatY {
		s.drownedRiseAt = time.Time{}
		return
	}
	dt := float32(now.Sub(s.drownedRiseAt).Seconds())
	s.drownedRiseAt = now
	ch.PosY = min(drownedFloatY, ch.PosY+drownedRiseSpeed*dt)
	s.movement.pos[1] = ch.PosY
	if s.player != nil {
		s.player.UpdateCharacter(*ch)
	}
	move := buildCharacterMove(ch, 0, 0)
	s.send(move, "drowned body rises")
	s.sendToViewers(move, "drowned body rises (view)")
	s.positionDirty = true
	if ch.PosY >= drownedFloatY {
		s.drownedRiseAt = time.Time{}
	}
}
