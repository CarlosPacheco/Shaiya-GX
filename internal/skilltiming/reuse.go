// Package skilltiming shares GX player-skill reuse rules between prediction and authority.
package skilltiming

import (
	"github.com/Spelunkern/Shaiya-GX/internal/attackspeed"
	"os"
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

// AnimationBound makes the shared action interval follow the basic-attack clip
// at the character's animation rate, like the original client (which waits for
// the action animation, 0x418030/0x40ff40) instead of a fixed table. The table
// stays the ceiling. Client and server read the same default; set
// GX_ANIMATION_CADENCE=off in both to restore the pure table of ps_game 0x5424e0.
var AnimationBound = os.Getenv("GX_ANIMATION_CADENCE") != "off"

// AttackClipFrames is the length of the longest basic-attack clips (26-30).
const AttackClipFrames = 30

// IntervalFor is the interval between two actions for a character whose attack
// index is tier and whose weapon has the given AttackTime. Without a weapon the
// animation runs at the neutral rate.
func IntervalFor(tier, weaponBase uint8, haveWeapon bool) (time.Duration, bool) {
	table, ok := Interval(tier)
	if !ok || !AnimationBound {
		return table, ok
	}
	fps := attackspeed.CharacterAttackFPS(tier, weaponBase, haveWeapon)
	clip := time.Duration(float64(AttackClipFrames) / float64(fps) * float64(time.Second))
	return min(table, clip), true
}
