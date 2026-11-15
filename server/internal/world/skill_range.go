package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
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

func (s *Session) sendSkillRangeFailure(k db.CharacterSkill) {
	s.send(buildSkillUseResult(packet.SkillUseNoTarget, 5, s.char.ID, 0, k.SkillID, k.Level, 0, 0, 0), "skill out of range or obstructed")
}
