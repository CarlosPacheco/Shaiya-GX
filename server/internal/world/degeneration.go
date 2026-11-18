package world

const (
	detailDegeneration = 90 // debuff: a creature that cannot attack or cast and crawls
	// degenerationSlow takes the move tier to its lowest whatever buffs there are.
	degenerationSlow = 4
)

// degenerated is Degeneration (TypeEffect 3, TypeDetail 90) on this user: the
// ChangeLevel of the skill names the monster it looks like (visual only).
func (s *Session) degenerated() bool {
	for _, b := range s.buffs {
		if b.Effect == typeEffectDebuff && b.Detail == detailDegeneration {
			return true
		}
	}
	return false
}

// cannotAct: stunned, asleep or transformed users cannot attack or use spells.
// They can still use consumables and change equipment, which do not ask this.
func (s *Session) cannotAct() bool { return s.actionBlocked() || s.degenerated() }
