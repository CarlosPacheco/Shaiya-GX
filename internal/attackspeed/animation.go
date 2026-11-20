package attackspeed

// Animation playback rates. The original clients advance an action's frame
// cursor by seconds times a frames-per-second value chosen from the actor's
// speed indices instead of a fixed 30 (Game.exe 0x41b300 for characters,
// 0x432520 for monsters). The blend clock between actions stays at 30.

// BaseFPS is the rate of every other action.
const BaseFPS = 30

// CharacterMoveFPS is the walk/run/step rate for the character's move index
// (0x41b300, state 1: +0x281). Index 255 and unknown indices keep 30.
func CharacterMoveFPS(tier uint8) float32 {
	if tier <= 4 {
		return [...]float32{24, 27, 30, 33, 36}[tier]
	}
	return BaseFPS
}

// CharacterAttackFPS is the basic-attack rate (0x41b300, state 2). The index
// is the character's attack index minus the weapon's own AttackTime, plus 5;
// anything outside 0..9 plays at 50. Without a weapon definition it is 30.
func CharacterAttackFPS(tier, weaponBase uint8, haveWeapon bool) float32 {
	if !haveWeapon || tier == Blocked {
		return BaseFPS
	}
	i := int(tier) - int(weaponBase) + 5
	if i < 0 || i > 9 {
		return 50
	}
	return [...]float32{15, 18, 21, 24, 27, 30, 34, 38, 42, 45}[i]
}

// MobMoveFPS is a walking or chasing monster's rate: (move index + 1) * 10.
func MobMoveFPS(move uint8) float32 { return (float32(move) + 1) * 10 }

// MobActionFPS is every other monster action (idle, attack, damage, death),
// selected by the monster's attack index; 3 and unknown indices give 30.
func MobActionFPS(attack uint8) float32 {
	if attack <= 6 {
		return [...]float32{15, 20, 25, 30, 35, 40, 45}[attack]
	}
	return BaseFPS
}
