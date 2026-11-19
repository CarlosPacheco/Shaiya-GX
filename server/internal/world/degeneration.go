package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

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

// transformedMob is the monster the active Degeneration makes this user look
// like (the skill's ChangeLevel when its ChangeType is 101), 0 when none.
func (s *Session) transformedMob() uint16 {
	for _, b := range s.buffs {
		if b.Effect == typeEffectDebuff && b.Detail == detailDegeneration {
			if def := db.MustDefs().EffectSkill(b.SkillID, b.SkillLevel); def != nil && def.ChangeType == 101 {
				return def.ChangeLevel
			}
		}
	}
	return 0
}

func buildTransform(user uint32, mob uint16) []byte {
	w := packet.New(packet.Opcode(proto.GXTransform))
	w.U32(user)
	w.U16(mob)
	return w.MustFinish()
}

// tickTransform shows or ends the creature look: whoever sees this user and the
// user itself are told when it changes (a user entering a view gets it with its
// other state).
func (s *Session) tickTransform() {
	p := s.player
	if p == nil || s.char == nil {
		return
	}
	mob := uint32(s.transformedMob())
	if p.transformMob.Swap(mob) == mob {
		return
	}
	frame := buildTransform(s.char.ID, uint16(mob))
	s.send(frame, "GX transform")
	s.sendToViewers(frame, "GX transform (view)")
}
