package effect

// PeriodicLibrary is Game.exe 0x576f50: the library of skills whose effect
// repeats on its own. It returns the catalogue index of the library; index
// 0x4d is the object loaded from DA_self_fire.EFT by 0x4246c0 (the objects
// are 0x12c bytes apart, DA_target_slumpshot being 0x4f). Fire Thorn plays it
// on every target its pulse hurts.
func PeriodicLibrary(skill uint16) string {
	switch skill {
	case 47:
		return "Effect/DA_self_fire.EFT"
	}
	return ""
}
