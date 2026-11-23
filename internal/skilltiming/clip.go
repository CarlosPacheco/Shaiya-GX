package skilltiming

import "time"

// ClipFloor is how long a skill's own animation runs. The original client
// keeps the action busy until that animation ends (0x40ff40), so the next
// action cannot start earlier. Animations play at 30 fps whatever the attack
// speed, hence no tier argument. Skills with a cast time, weapon swings and
// anything not in the generated table have no floor.
func ClipFloor(skill uint16, level uint8) time.Duration {
	return time.Duration(clipMilliseconds[uint32(skill)<<8|uint32(level)]) * time.Millisecond
}
