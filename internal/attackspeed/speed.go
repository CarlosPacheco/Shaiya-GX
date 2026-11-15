// Package attackspeed describes the original server attack indices and delays.
package attackspeed

const Maximum uint8 = 9
const Blocked uint8 = 255

// Tier adds equipment, mastery and applied ability 16 to Items.AttackTime.
// InitEquipment starts at zero; ItemEquipmentAdd adds the weapon field directly.
func Tier(base uint8, modifier int32) uint8 {
	return uint8(min(max(int64(base)+int64(modifier), 0), int64(Maximum)))
}

// Milliseconds is ps_game.exe g_ConstUserAttackSpeed at 0x5424e0.
func Milliseconds(tier uint8) (uint32, bool) {
	if tier > Maximum {
		return 0, false
	}
	return [...]uint32{4000, 3000, 2750, 2500, 2250, 2000, 1750, 1500, 1250, 1000}[tier], true
}

// Message is the native status lookup (Game.exe 0x506b57).
func Message(tier uint8) int {
	if tier > Maximum {
		return 5281
	}
	return 5271 + int(tier)
}
