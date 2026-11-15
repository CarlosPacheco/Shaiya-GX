// Package pvprank shares the original client's PvP rank selection between
// world nameplates and the Status (T) title.
package pvprank

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
)

// EP5.4 client SHA256 8e5ef14a...bb3263: thresholds at 0x0070b2e8,
// icon selection at 0x00443f90 and title selection at 0x00504285.
var thresholds = [...]uint32{
	1, 50, 300, 1000, 5000, 10000, 20000, 30000, 40000, 50000,
	70000, 90000, 110000, 130000, 150000, 200000, 250000, 300000, 350000, 400000,
	450000, 500000, 550000, 600000, 650000, 700000, 750000, 800000, 850000, 900000,
	1000000,
}

// Index returns the zero-based atlas cell and whether a rank exists.
// The client's 31 display ranks are independent of the server's kill cap.
func Index(kills uint32) (int, bool) {
	for i := len(thresholds) - 1; i >= 0; i-- {
		if kills >= thresholds[i] {
			return i, true
		}
	}
	return 0, false
}

// Title follows native 0x005042a1-0x00504325. Names and prefixes come from
// SysMsg, including the special final title; zero kills leaves Title blank.
func Title(kills uint32, messages text.SysMsg) string {
	i, ok := Index(kills)
	if !ok {
		return ""
	}
	if i == 30 {
		return messages.Get(5404)
	}
	prefix := 5450
	if i >= 15 {
		prefix = 5451
		i -= 15
	}
	return strings.TrimSpace(messages.Get(prefix) + " " + messages.Get(5389+i))
}
