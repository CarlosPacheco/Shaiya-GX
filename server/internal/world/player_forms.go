package world

import (
	"encoding/binary"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

type formLook struct {
	kind       proto.PlayerFormKind
	mob, skill uint16
	rank       uint8
	disguise   *disguiseLook
}

type disguiseLook struct{ shape []byte }

func formOf(def *db.SkillDef) proto.PlayerFormKind {
	return proto.SkillForm(def.TypeEffect, def.TypeDetail, def.ChangeType)
}

func (s *Session) copyFormTarget(def *db.SkillDef, target uint32, completed bool) (formLook, bool) {
	look := formLook{kind: formOf(def)}
	ch := s.char
	switch look.kind {
	case proto.FormMonsterCopy:
		m, ok := MobByID(ch.MapID, target)
		md := db.MustDefs().Mobs[m.MobID]
		if !ok || md == nil || md.Obelisk || m.CurrentHP == 0 || md.Level > ch.Level ||
			mobFriendly(md, s.faction) || !s.mobSkillReach(&m, def, completed) {
			return formLook{}, false
		}
		look.mob = m.MobID
	case proto.FormDisguise:
		p := FindPlayerByID(target)
		if p == nil || p == s.player || p.MapID() != ch.MapID || !p.IsAlive() || p.Faction() == s.faction ||
			p.immune.Load() || p.mountShape() != 0 ||
			s.player == nil || !s.player.canSee(p) || duelFoes(s.player, p) || guildDuelFoes(s.player, p) ||
			!s.userSkillReach(p, def, completed) {
			return formLook{}, false
		}
		if existing := p.formLook.Load(); existing != nil && existing.kind != proto.FormEvolution {
			return formLook{}, false
		}
		p.mu.Lock()
		other := p.character
		shape := buildCharacterShapeGuild(&other, p.faction, p.items, 0, "")
		p.mu.Unlock()
		if other.MapID != ch.MapID || other.Level > ch.Level || !p.IsAlive() {
			return formLook{}, false
		}
		// Copy the native appearance and name, with the real caster's identity on the wire.
		binary.LittleEndian.PutUint32(shape[4:], ch.ID)
		look.disguise = &disguiseLook{shape: shape}
	default:
		return formLook{}, false
	}
	return look, true
}

func (s *Session) castCopiedForm(op packet.Opcode, k db.CharacterSkill, def *db.SkillDef, target uint32, stats CharacterStats, completed bool) {
	look, ok := s.copyFormTarget(def, target, completed)
	if !ok || !s.canAddEffect(def, s.char.ID) {
		s.rejectCopyForm(op, k)
		return
	}
	s.startCooldown(k, def, completed)
	sp, mp := s.skillCosts(def, &stats)
	s.spendDelta(mp, sp)
	s.persist()
	s.send(buildSkillUsedSpMp(sp, mp), "copy form skill costs")
	if _, added := s.addEffect(def, 0, s.char.ID); added {
		s.buffs[len(s.buffs)-1].Form = look
		s.tickTransform()
		s.broadcastNear(buildSkillUseResult(packet.SkillUseNoTarget, skillResultNormal, s.char.ID, s.char.ID, k.SkillID, k.Level, 0, 0, 0), "copy form applied", true)
	}
}

func (s *Session) rejectCopyForm(op packet.Opcode, k db.CharacterSkill) {
	s.send(buildSkillUseResult(op, skillResultNotTarget, s.char.ID, 0, k.SkillID, k.Level, 0, 0, 0), "copy form target refused")
}

func (s *Session) activeForm() formLook {
	for i := len(s.buffs) - 1; i >= 0; i-- {
		b := &s.buffs[i]
		if b.Form.kind != proto.FormNone && b.EndsAt.After(time.Now()) {
			return b.Form
		}
	}
	return formLook{}
}

func (s *Session) breakCombatForm() {
	if s.player == nil {
		return
	}
	look := s.player.formLook.Load()
	if look != nil && (look.kind == proto.FormMonsterCopy || look.kind == proto.FormDisguise) {
		s.removeEffects(breakableForm, "combat cancelled the copied form")
	}
}

func breakableForm(b *ActiveBuff) bool {
	return b.Form.kind == proto.FormMonsterCopy || b.Form.kind == proto.FormDisguise
}

func buildFormAppearance(id uint32, look formLook) []byte {
	switch look.kind {
	case proto.FormMonsterCopy, proto.FormDegeneration:
		return buildTransform(id, look.mob)
	}
	w := packet.New(packet.Opcode(proto.GXFormAppearance))
	w.U32(id)
	w.U8(uint8(look.kind))
	switch look.kind {
	case proto.FormEvolution:
		w.U16(look.skill)
		w.U8(look.rank)
		for _, action := range evolutionActions(db.MustDefs().Mobs[look.mob]) {
			var id uint16
			if action.skill != nil {
				id = action.skill.SkillID
			}
			w.U16(id)
			w.U8(action.slot)
		}
	case proto.FormDisguise:
		w.Bytes(look.disguise.shape[2:])
	}
	return w.MustFinish()
}

func (s *Session) publishForm(old, next formLook) {
	if old.kind != next.kind {
		switch old.kind {
		case proto.FormMonsterCopy, proto.FormDegeneration:
			s.sendFormFrame(buildTransform(s.char.ID, 0))
		case proto.FormDisguise, proto.FormEvolution:
			s.sendFormFrame(buildFormAppearance(s.char.ID, formLook{}))
		}
	}
	if next.kind != proto.FormNone {
		s.sendFormFrame(buildFormAppearance(s.char.ID, next))
	}
}

func (s *Session) sendFormFrame(frame []byte) {
	s.send(frame, "player form")
	s.sendToViewers(frame, "player form (view)")
}
