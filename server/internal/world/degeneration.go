package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
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

func (s *Session) transformedMob() uint16 { return s.activeForm().mob }

func buildTransform(user uint32, mob uint16) []byte {
	w := packet.New(packet.Opcode(proto.GXTransform))
	w.U32(user)
	w.U16(mob)
	return w.MustFinish()
}

// tickTransform publishes a change and keeps the view snapshot for late observers.
func (s *Session) tickTransform() {
	p := s.player
	if p == nil || s.char == nil {
		return
	}
	var old formLook
	if published := p.formLook.Load(); published != nil {
		old = *published
	}
	next := s.activeForm()
	if old == next {
		return
	}
	if next.kind == proto.FormNone {
		p.formLook.Store(nil)
	} else {
		p.formLook.Store(&next)
	}
	s.publishForm(old, next)
	if old.kind == proto.FormEvolution || next.kind == proto.FormEvolution {
		s.updateEvolutionGrant(next)
	}
}
