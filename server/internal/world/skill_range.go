package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skillrange"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func (s *Session) skillLineClear(a, b skillrange.Position) bool {
	// Live entry requires geometry. Unselected synthetic sessions have none.
	g := s.movement.geometry
	return g == nil || g.LineOfSight(mapgeometry.Position(a), mapgeometry.Position(b))
}
func (s *Session) skillTargetReach(p skillrange.Position, base, size float32, completed bool) bool {
	a := skillrange.Position{s.char.PosX, s.char.PosY, s.char.PosZ}
	return skillrange.Within(a, p, skillrange.Reach(base, size, completed)) && s.skillLineClear(a, p)
}
func (s *Session) mobSkillReach(m *MapMob, def *db.SkillDef, completed bool) bool {
	size := float32(0)
	if d := db.MustDefs().Mobs[m.MobID]; d != nil {
		size = float32(d.Size)
	}
	return s.skillTargetReach(skillrange.Position{m.PosX, m.PosY, m.PosZ}, float32(def.AttackRange), size, completed)
}
func (s *Session) userSkillReach(p *ConnectedPlayer, def *db.SkillDef, completed bool) bool {
	if p == s.player {
		return true
	}
	c := p.Snapshot()
	return s.skillTargetReach(skillrange.Position{c.PosX, c.PosY, c.PosZ}, float32(def.AttackRange), 0, completed)
}
func (s *Session) skillAreaContains(center, target skillrange.Position, radius float32) bool {
	return skillrange.Within(center, target, radius) && s.skillLineClear(center, target)
}

func (s *Session) sendSkillRangeFailure(k db.CharacterSkill, target ...skillrange.Position) {
	message := uint16(312)
	if len(target) > 0 && !s.skillLineClear(skillrange.Position{s.char.PosX, s.char.PosY, s.char.PosZ}, target[0]) {
		message = 458
	}
	w := packet.New(packet.Opcode(proto.GXSkillFailure))
	w.U16(k.SkillID)
	w.U16(message)
	s.send(w.MustFinish(), "localized skill reach failure")
}
func (s *Session) sendUserSkillRangeFailure(k db.CharacterSkill, p *ConnectedPlayer) {
	if p == nil {
		s.sendSkillRangeFailure(k)
		return
	}
	c := p.Snapshot()
	s.sendSkillRangeFailure(k, skillrange.Position{c.PosX, c.PosY, c.PosZ})
}
func (s *Session) sendMobSkillRangeFailure(k db.CharacterSkill, m *MapMob) {
	s.sendSkillRangeFailure(k, skillrange.Position{m.PosX, m.PosY, m.PosZ})
}
