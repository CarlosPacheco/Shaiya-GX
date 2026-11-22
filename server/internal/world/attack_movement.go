package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/attackspeed"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"time"
)

// Attacks and spell casts require footing. Only the attacker is checked.
func (s *Session) attackMovementAllowed() bool {
	m := &s.movement
	return !m.jumping && (m.ladder.Active || m.geometry == nil || m.geometry.Supported(m.pos) ||
		m.pos[1] <= -1.5 && m.geometry.DeepWater(m.pos))
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
