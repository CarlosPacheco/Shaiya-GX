package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

type evolutionAction struct {
	skill *db.SkillDef
	slot  uint8
}

func evolutionActions(mob *db.MobDef) (out [2]evolutionAction) {
	if mob == nil {
		return out
	}
	var n int
	for i, attack := range mob.Attacks {
		if n == len(out) {
			break
		}
		if attack.Type == 0 || attack.OK == 0 {
			continue
		}
		sk := db.MustDefs().NPCSkill(attack.Type)
		if sk == nil || sk.TypeEffect < typeEffectAttack || sk.TypeEffect > typeEffectField ||
			n == 1 && out[0].skill.SkillID == sk.SkillID {
			continue
		}
		out[n] = evolutionAction{skill: sk, slot: uint8(i)}
		n++
	}
	return out
}

func (s *Session) updateEvolutionGrant(look formLook) {
	s.evolutionGrant = 0
	s.evolutionActions = [2]evolutionAction{}
	if look.kind == proto.FormEvolution {
		s.evolutionSerial++
		if s.evolutionSerial == 0 {
			s.evolutionSerial = 1
		}
		s.evolutionGrant = s.evolutionSerial
		s.evolutionActions = evolutionActions(db.MustDefs().Mobs[look.mob])
	}
	if s.cast != nil && s.cast.evolution != 0 && s.cast.evolution != s.evolutionGrant {
		s.cast = nil
		s.syncPlayerSpeed(false)
	}
	s.sendEvolutionBar()
}

func (s *Session) sendEvolutionBar() {
	w := packet.New(packet.Opcode(proto.GXEvolutionBar))
	w.U32(s.evolutionGrant)
	for _, a := range s.evolutionActions {
		var id uint16
		var reach uint8
		var total, left uint32
		if a.skill != nil {
			id, reach = a.skill.SkillID, a.skill.AttackRange
			total = uint32(a.skill.ResetTimeSeconds) * 1000
			left = uint32(max(time.Until(s.evolutionReadyAt[id]).Milliseconds(), 0))
		}
		w.U16(id)
		w.U8(reach)
		w.U32(total)
		w.U32(left)
	}
	s.send(w.MustFinish(), "temporary monster skills")
}

func (s *Session) evolutionSkill(generation uint32, cell uint8) *db.SkillDef {
	if generation == 0 || generation != s.evolutionGrant || int(cell) >= len(s.evolutionActions) ||
		s.activeForm().kind != proto.FormEvolution {
		return nil
	}
	return s.evolutionActions[cell].skill
}

func (s *Session) handleEvolutionCast(p []byte) {
	if len(p) != 11 || s.char == nil || s.dead || s.ridingVehicle() || s.cast != nil || s.cannotAct() {
		return
	}
	r := proto.NewReader(p)
	generation, cell, target := r.U32(), r.U8(), r.U32()
	def := s.evolutionSkill(generation, cell)
	if def == nil {
		return
	}
	op := packet.Opcode(packet.SkillUse)
	if def.TypeEffect == typeEffectHeal || def.TypeEffect == typeEffectBuff || def.TargetType == targetTypeEnemiesNearCaster {
		if def.TypeEffect == typeEffectHeal {
			// A monster heals itself, regardless of the currently selected enemy.
			target = s.char.ID
		}
		op = packet.SkillUseNoTarget
	} else if p := FindPlayerByID(target); p != nil {
		op = packet.SkillUseNoTarget
	}
	s.cancelMove()
	s.cancelVehicleCast()
	s.cancelTravelCast()
	s.beginSkill(op, db.CharacterSkill{SkillID: def.SkillID, Level: def.SkillLevel, Number: cell}, def, target, generation, nil)
}

func (s *Session) definitionReady(def *db.SkillDef) bool {
	if def.MobSkill {
		return !s.evolutionReadyAt[def.SkillID].After(time.Now())
	}
	return s.skillReady(def.SkillID)
}
