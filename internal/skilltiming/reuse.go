// Package skilltiming shares GX player-skill reuse rules between prediction and authority.
package skilltiming

import "time"

// CastInterval is a shared per-player cadence, independent of per-skill reuse
// and the longer movement pause. Alternating skills cannot bypass it.
const CastInterval = 300 * time.Millisecond

// ReuseSeconds treats the native one-second placeholder as no reuse delay.
// Longer cooldowns retain their authored duration; this does not shorten casting.
func ReuseSeconds(seconds uint16) uint16 {
	if seconds <= 1 {
		return 0
	}
	return seconds
}
