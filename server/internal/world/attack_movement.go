package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/attackspeed"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"time"
)

const attackMotionFreshness = 150 * time.Millisecond

// Jumping alone is allowed. Only the attacker's actual horizontal travel is
// relevant; the victim's jump or movement never grants damage immunity.
func (s *Session) attackMovementAllowed(now time.Time) bool {
	m := &s.movement
	airborne := m.jumping || m.geometry != nil && !m.geometry.Supported(m.pos)
	return !airborne || m.horizontalAt.IsZero() || now.Sub(m.horizontalAt) >= attackMotionFreshness
}

func (s *Session) commitAttackStop(now time.Time) {
	s.attackMoveUntil = now.Add(attackspeed.MovementPauseMilliseconds * time.Millisecond)
	m := &s.movement
	if m.ladder.Active {
		m.ladder.Mode = 1
		s.notifyLadder()
	}
	m.credit = 0
	m.budgetAt = now
	m.motion = 0
	if m.jumping {
		m.motion = 4 | proto.MotionFlag80
	}
	if s.player != nil {
		s.player.UpdateMotion(m.motion, m.direction)
		s.sendToViewers(buildCharacterMove(s.char, m.motion, m.direction), "attack movement stop")
	}
}
