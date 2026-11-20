package scene

import "github.com/Spelunkern/Shaiya-GX/internal/attackspeed"

type rateKind uint8

const (
	rateNone rateKind = iota
	rateMove
	rateAttack
)

// playbackRate scales how fast the current clip's frame cursor advances, as the
// original does from the speed indices it receives (see attackspeed/animation.go).
// Without known indices every clip plays at 30 frames per second.
type playbackRate struct {
	kind, next   rateKind
	known        bool
	move, attack uint8
	weaponBase   uint8
	haveWeapon   bool
	creature     float32 // fixed multiplier set by monster actors; 0 = unused
}

func (r *playbackRate) factor() float32 {
	if r.creature > 0 {
		return r.creature
	}
	if !r.known {
		return 1
	}
	switch r.kind {
	case rateMove:
		return attackspeed.CharacterMoveFPS(r.move) / attackspeed.BaseFPS
	case rateAttack:
		return attackspeed.CharacterAttackFPS(r.attack, r.weaponBase, r.haveWeapon) / attackspeed.BaseFPS
	}
	return 1
}

// SetSpeedTiers gives a character its move and attack indices (0x051C) and the
// base AttackTime of the weapon it wields.
func (c *Character) SetSpeedTiers(move, attack, weaponBase uint8, haveWeapon bool) {
	if c == nil {
		return
	}
	c.rate.known, c.rate.move, c.rate.attack = true, move, attack
	c.rate.weaponBase, c.rate.haveWeapon = weaponBase, haveWeapon
}

// SetCreatureRate sets a monster's playback multiplier (frames per second / 30).
func (c *Character) SetCreatureRate(fps float32) {
	if c != nil {
		c.rate.creature = fps / attackspeed.BaseFPS
	}
}
