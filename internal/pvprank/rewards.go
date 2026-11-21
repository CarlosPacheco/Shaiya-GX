package pvprank

// GX uses all 31 stages from the EP5 client's verified tables by explicit
// user request. This extends the older EP4.5 server's 16-stage limit.
const RewardCount = 31

// CounterCap is the final threshold in the client's rank table.
const CounterCap = 1000000

var statRewards = [31]uint16{
	1, 3, 5, 7,
	12, 18, 20, 22,
	24, 26, 26, 26,
	27, 27, 27, 28,
	28, 28, 28, 28, 28,
	29, 29, 29, 29, 29,
	30, 30, 30, 30, 30,
}

var deathMoney = [31]uint32{100, 200, 500, 1000, 5000, 10000, 20000, 50000, 100000, 200000, 300000, 400000, 500000, 650000, 800000, 1000000,
	1200000, 1400000, 1600000, 1800000, 2000000, 2200000, 2400000, 2600000, 2800000, 3000000, 3200000, 3400000, 3600000, 3800000, 4000000}

// Reward returns the amount for a one-based claim stage.
func Reward(stage uint8) (count uint32, points uint16, money uint32, ok bool) {
	if stage < 1 || stage > RewardCount {
		return 0, 0, 0, false
	}
	i := stage - 1
	return thresholds[i], statRewards[i], deathMoney[i], true
}
