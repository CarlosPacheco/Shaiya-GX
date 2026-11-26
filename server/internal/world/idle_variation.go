package world

import (
	"math/rand/v2"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Game.exe 0x4191cd and 0x41c39c pick ALT 16 or 17 after 10000 ms
// of standing idle. World picks once and sends the same choice to viewers.
const idleVariationInterval = 10 * time.Second

func (s *Session) tickIdleVariation(now time.Time) {
	m := &s.movement
	if s.player == nil || m.geometry == nil || s.dead || s.mounted || s.autoAttack ||
		s.cast != nil || s.dash != nil || now.Before(s.attackMoveUntil) ||
		m.motion != 0 || m.jumping || m.ladder.Active || s.player.IsSitting() ||
		s.player.vehicleState().State != proto.VehicleOff || s.cannotAct() ||
		m.pos[1] <= -1.5 && m.geometry.DeepWater(m.pos) || !m.geometry.Supported(m.pos) {
		m.idleSince = time.Time{}
		return
	}
	if m.idleSince.IsZero() || s.lastCombatAt.After(m.idleSince) {
		m.idleSince = now
		return
	}
	if now.Sub(m.idleSince) <= idleVariationInterval {
		return
	}
	m.idleSince = now
	motion := proto.MotionIdle1 + uint8(rand.IntN(2))
	s.broadcastNear(buildCharacterMotion(s.char.ID, motion), "0x0506 idle variation", true)
}
