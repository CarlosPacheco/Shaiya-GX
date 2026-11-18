package effect

// PeriodicHit reports the skills that hurt on their own every few moments
// (Fire Thorn, Fire Bush). Each of their hits plays the damage effect of the
// skill's element on the target: DA_target_Adamage, one sequence per element
// (Game.exe 0x577b40 maps AttribType to the sequence), a plain flame for fire.
func PeriodicHit(skill uint16) bool { return skill == 47 || skill == 48 }

// AttributeDamageLibrary holds the five element damage effects.
const AttributeDamageLibrary = "Effect/DA_target_Adamage.EFT"

// AttributeDamageSequence is Game.exe 0x577b40: the sequence of the element
// damage library for an AttribType (fire 1 is sequence 2); no element is 4.
func AttributeDamageSequence(attribute uint8) int {
	switch attribute {
	case 1:
		return 2
	case 2:
		return 0
	case 3:
		return 3
	case 4:
		return 1
	}
	return 4
}
