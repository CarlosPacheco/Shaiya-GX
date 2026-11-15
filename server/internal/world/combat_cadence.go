package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"time"
)

// Both basic attack routes and skills recover from lastActionAt. Use the live
// equipment/buff tier so haste, slow and gear changes affect recovery immediately.
func (s *Session) combatActionReady(now time.Time) bool {
	interval, ok := skilltiming.Interval(s.speeds().attack)
	return ok && (s.lastActionAt.IsZero() || now.Sub(s.lastActionAt) >= interval)
}
