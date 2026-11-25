// Package skilltiming shares GX player-skill reuse rules between prediction and authority.
package skilltiming

import (
	"github.com/Spelunkern/Shaiya-GX/internal/attackspeed"
	"os"
	"time"
)

// Interval returns the original server delay at the zero-based attack index.
// Invalid and blocked indices have no interval.
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

// AnimationBound makes GX skill recovery follow the character's attack clip
// rate, capped by the original server interval. Client and server read the
// same default. GX_ANIMATION_CADENCE=off uses the table for skill recovery.
var AnimationBound = os.Getenv("GX_ANIMATION_CADENCE") != "off"

// AttackClipFrames is the length of the longest basic-attack clips (26-30).
const AttackClipFrames = 30

// IntervalFor returns GX skill recovery for the character's attack index and
// weapon baseline. Without a weapon, the animation runs at the neutral rate.
func IntervalFor(tier, weaponBase uint8, haveWeapon bool) (time.Duration, bool) {
	table, ok := Interval(tier)
	if !ok || !AnimationBound {
		return table, ok
	}
	fps := attackspeed.CharacterAttackFPS(tier, weaponBase, haveWeapon)
	clip := time.Duration(float64(AttackClipFrames) / float64(fps) * float64(time.Second))
	return min(table, clip), true
}
