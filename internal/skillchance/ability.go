// Package skillchance describes percentage-point skill modifiers shared by
// the world and client skill descriptions.
package skillchance

// Modifier limits each authored chance modifier to 100 percentage points.
// Zero means no modifier, and negative values keep their sign.
func Modifier(kind uint8, value int32) int32 {
	switch kind {
	case 18, 20, 21, 22, 29, 30, 31:
		return min(max(value, -100), 100)
	default:
		return value
	}
}
