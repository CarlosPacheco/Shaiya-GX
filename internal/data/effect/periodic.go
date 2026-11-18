package effect

// PeriodicLibrary is Game.exe 0x576f50: the library of skills whose effect
// repeats on its own. It returns the catalogue index of the library; index
// 0x4d is the object loaded from DA_self_fire.EFT by 0x4246c0 (the objects
// are 0x12c bytes apart, DA_target_slumpshot being 0x4f). Fire Thorn plays it
// on every target its pulse hurts, and so does Fire Bush's curtain of fire.
func PeriodicLibrary(skill uint16) string {
	switch skill {
	case 47, 48:
		return "Effect/DA_self_fire.EFT"
	}
	return ""
}

// PeriodicParts are the effects of that library the periodic hit really shows
// on each target: the simple flame ("화염연기", effects 2 and 15), not the whole
// composite (shield, light rays and rising columns) that the same library plays
// for a cast. A level's sequence holds 15 and, from level 2, 2 as well.
func PeriodicParts(skill uint16) []int {
	switch skill {
	case 47, 48:
		return []int{2, 15}
	}
	return nil
}
