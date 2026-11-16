package effect

// AreaLibrary is the original Game.exe 0x577020 detailed area selector.
// These libraries are independent of the target selector at 0x577390.
func AreaLibrary(skill uint16) string {
	switch skill {
	case 48:
		return "Effect/DA_area_fire.EFT"
	case 50:
		return "Effect/DA_area_rock.EFT"
	case 52:
		return "Effect/DA_area_gale.EFT"
	case 53:
		return "Effect/DA_area_hailstorm.EFT"
	case 68:
		return "Effect/DA_area_meteor.EFT"
	case 69:
		return "Effect/DA_area_earthquake.EFT"
	case 70:
		return "Effect/DA_area_lightning.EFT"
	case 71:
		return "Effect/DA_area_blizzard.EFT"
	case 72:
		return "Effect/DA_area_elemental.EFT"
	case 81, 174, 301:
		return "Effect/DA_target_valribomb.EFT"
	case 83:
		return "Effect/DA_area_earthpole.EFT"
	case 84:
		return "Effect/DA_area_turbulent.EFT"
	case 122:
		return "Effect/DA_area_disease.EFT"
	case 156, 158:
		return "Effect/DA_area_Mwind.EFT"
	case 162:
		return "Effect/DA_area_bloodshower.EFT"
	case 163:
		return "Effect/DA_area_shower.EFT"
	case 184:
		return "Effect/DA_self_kamainb.EFT"
	case 353, 355:
		return "Effect/ArrowBomberShot.EFT"
	case 356:
		return "Effect/MissileRage.EFT"
	}
	return ""
}
