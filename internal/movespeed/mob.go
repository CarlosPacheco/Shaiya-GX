package movespeed

// MobMoveTier is the tier used for displacement, including during a move
// that was already in progress when World sent a blocked index (255).
func MobMoveTier(index uint8) uint8 {
	if index <= 4 {
		return index
	}
	return 2
}

// MobUnitsPerSecond is how fast the original client walks a monster toward its
// newest destination (Game.exe 0x4312a0): the monster's step plus the move
// index offset (-4, -2, 0, +2, +4) over its time in seconds, minus 0.3.
// Walking and running use their own step and time. It returns 0 when the
// monster has no usable time.
func MobUnitsPerSecond(step int, timeMS uint32, tier uint8) float64 {
	if timeMS == 0 {
		return 0
	}
	offset := [...]int{-4, -2, 0, 2, 4}[MobMoveTier(tier)]
	v := float64(step+offset)/(float64(timeMS)*0.001) - 0.3
	return max(v, 0.1)
}
