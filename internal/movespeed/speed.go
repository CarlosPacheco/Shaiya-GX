// Package movespeed describes the original on-foot movement tiers shared by
// authoritative simulation and client prediction.
package movespeed

const (
	Normal  uint8 = 2
	Maximum uint8 = 4
	Blocked uint8 = 255
)

// Tier is CUser::GetMoveSpeedIndex (ps_game.exe 0x48a530). Equipment and
// applied ability 17 contribute signed offsets to the normal tier.
func Tier(modifier int32) uint8 {
	return uint8(min(max(int64(Normal)+int64(modifier), 0), int64(Maximum)))
}

// Units is Game.exe 0x43bfa0, with base 5 initialized at 0x409250.
// Unknown values retain the original base; 255 prohibits movement.
func Units(tier uint8) float32 {
	if tier == Blocked {
		return 0
	}
	if tier <= Maximum {
		return 3 + float32(tier)
	}
	return 5
}

// Message follows the original status switch at 0x506e12. The native display
// uses five selected SysMsg descriptions rather than consecutive text IDs.
func Message(tier uint8) int {
	if tier > Maximum {
		return 5282
	}
	return [...]int{5273, 5274, 5276, 5278, 5279}[tier]
}

// MountedUnits adds the original extremely-fast tier, reserved for vehicles.
func MountedUnits(tier uint8) float32 {
	if tier == 5 {
		return 8
	}
	return Units(tier)
}
