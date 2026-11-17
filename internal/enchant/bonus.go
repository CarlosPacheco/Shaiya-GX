// Package enchant shares compiled enchantment bonuses between server and tooltips.
package enchant

// WeaponAttack is the total attack bonus at each resulting enchantment level.
// Weapon enchantment also grants this amount to direct HP spell healing.
// Unconfigured levels above +20 retain their existing zero bonus.
var WeaponAttack = [50]uint32{0, 7, 14, 21, 31, 41, 51, 64, 77, 90, 106, 122, 138, 157, 176, 195, 217, 239, 261, 286, 311}
