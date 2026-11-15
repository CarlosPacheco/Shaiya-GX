package pvprank

// GX uses all 31 stages from the EP5 client's verified tables by explicit
// user request. This extends the older EP4.5 server's 16-stage limit.
const RewardCount = 31

// CounterCap is the final threshold in the client's rank table.
const CounterCap = 1000000

var statRewards = [31][3]uint16{
	{1, 1, 1}, {2, 3, 3}, {3, 5, 5}, {5, 7, 7},
	{10, 12, 12}, {15, 15, 18}, {16, 18, 20}, {16, 21, 22},
	{16, 21, 24}, {17, 22, 26}, {17, 22, 26}, {17, 22, 26},
	{18, 23, 27}, {18, 23, 27}, {18, 23, 27}, {19, 24, 28},
	{19, 24, 28}, {19, 24, 28}, {19, 24, 28}, {19, 24, 28}, {19, 24, 28},
	{20, 25, 29}, {20, 25, 29}, {20, 25, 29}, {20, 25, 29}, {20, 25, 29},
	{21, 26, 30}, {21, 26, 30}, {21, 26, 30}, {21, 26, 30}, {21, 26, 30},
}

var deathMoney = [31]uint32{100, 200, 500, 1000, 5000, 10000, 20000, 50000, 100000, 200000, 300000, 400000, 500000, 650000, 800000, 1000000,
	1200000, 1400000, 1600000, 1800000, 2000000, 2200000, 2400000, 2600000, 2800000, 3000000, 3200000, 3400000, 3600000, 3800000, 4000000}

// Reward returns a one-based claim stage. Easy mode cannot redeem rewards.
func Reward(mode, stage uint8) (count uint32, points uint16, money uint32, ok bool) {
	if mode < 1 || mode > 3 || stage < 1 || stage > RewardCount {
		return 0, 0, 0, false
	}
	return DisplayReward(mode, stage)
}

// DisplayReward is the EP5 client preview at 0x0070b164/1e4/264/364. Its
// 31 rows also supply GX's authoritative reward amounts.
func DisplayReward(mode, stage uint8) (count uint32, points uint16, money uint32, ok bool) {
	if mode < 1 || mode > 3 || stage < 1 || stage > 31 {
		return 0, 0, 0, false
	}
	i := stage - 1
	return thresholds[i], statRewards[i][mode-1], deathMoney[i], true
}
