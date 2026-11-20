package movespeed

// MobUnitsPerSecond is how fast the original client walks a monster toward its
// newest destination (Game.exe 0x4312a0): the monster's step plus the move
// index offset (-4, -2, 0, +2, +4) over its time in seconds, minus 0.3 so the
// drawn mob never outruns the server. Normal and chase moves use their own
// step and time. It returns 0 when the monster has no usable time.
func MobUnitsPerSecond(step int, timeMS uint32, tier uint8) float64 {
	if timeMS == 0 {
		return 0
	}
	offset := 0
	if tier <= 4 {
		offset = [...]int{-4, -2, 0, 2, 4}[tier]
	}
	v := float64(step+offset)/(float64(timeMS)*0.001) - 0.3
	return max(v, 0.1)
}
