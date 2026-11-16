package skilltiming

import "time"

const (
	SpinDuration = 2 * time.Second
	// The original fighter spin first contacts at ALT frame 4, at 30 fps.
	SpinContact   = time.Second * 4 / 30
	StingerWindup = 10.0 / 30
	StingerSpeed  = 100.0
)

func Spin(id uint16) bool { return id == 38 || id == 156 }

// AreaCombo implements the requested early replacement of a spin by another
// distinct spin or an instant hostile area centred on the caster. Own reuse,
// target validation and costs still apply. Ordinary attacks/buffs cannot use it.
func AreaCombo(previous, next uint16, target, effect, ready uint8) bool {
	return Spin(previous) && previous != next && ready == 0 && (effect == 1 || effect == 3) && (Spin(next) || target == 6)
}

func StingerContact(race, gender uint8) float32 {
	if race == 3 {
		if gender == 1 {
			return 18
		}
		return 24
	}
	if gender == 1 {
		return 24
	}
	return 26
}
