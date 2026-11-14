// Package forgecost shares the original static blacksmith gold table.
package forgecost

// Gold is GetGemMoneyByGrade (0x0045f6e0), indexed by lapis/lapisia ReqIg.
func Gold(reqIg uint8) uint32 {
	low := [...]uint32{1000, 4095, 11250, 22965, 41280, 137900, 365000, 480000, 627000, 814000, 1040000}
	high := [...]uint32{2500000, 3500000, 5500000, 7500000, 9800000}
	switch {
	case reqIg >= 30 && reqIg <= 40:
		return low[reqIg-30]
	case reqIg >= 96 && reqIg <= 100:
		return high[reqIg-96]
	}
	return 0
}
