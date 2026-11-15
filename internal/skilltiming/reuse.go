// Package skilltiming shares GX player-skill reuse rules between prediction and authority.
package skilltiming

import (
	"github.com/Spelunkern/Shaiya-GX/internal/attackspeed"
	"time"
)

// Interval uses the same native attack-speed table for basics and skills.
// Invalid/blocked indices cannot perform an action; Ext. Fast caps at 1000ms.
func Interval(tier uint8) (time.Duration, bool) {
	ms, ok := attackspeed.Milliseconds(tier)
	return time.Duration(ms) * time.Millisecond, ok
}

// ReuseSeconds treats the native one-second placeholder as no reuse delay.
// Longer cooldowns retain their authored duration; this does not shorten casting.
func ReuseSeconds(seconds uint16) uint16 {
	if seconds <= 1 {
		return 0
	}
	return seconds
}
