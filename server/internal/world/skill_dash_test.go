package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func TestStingerServerSweepCannotCrossWallOrTrustClientPosition(t *testing.T) {
	s, _ := forgeRuleSession(t)
	p, _ := playerPhysicsFixture(t)
	s.char.MapID, s.char.PosX, s.char.PosY, s.char.PosZ = p.char.MapID, p.char.PosX, p.char.PosY, p.char.PosZ
	s.movement = p.movement
	s.char.CurrentHP = 100
	def := &db.SkillDef{SkillID: 56}
	if s.beginStinger(def, [3]float32{12, .05, 5}) || s.dash != nil {
		t.Fatal("dash crossed wall")
	}
	if !s.beginStinger(def, [3]float32{4, .05, 5}) || s.dash == nil {
		t.Fatal("clear dash refused")
	}
	from := s.char.PosX
	s.handleCharacterMove(proto.MovePacket(1, 0, 900, 900, 900))
	if s.char.PosX != from {
		t.Fatal("client moved during server dash")
	}
	s.dash.Started = mapgeometry.PhysicsSeconds() - .35
	s.tickStinger()
	if s.char.PosX >= from || s.char.PosX < 4 {
		t.Fatal("not travelling at server speed", s.char.PosX)
	}
	s.dash.Started = mapgeometry.PhysicsSeconds() - 2
	s.tickStinger()
	if s.dash != nil || s.char.PosX < 3.95 || s.char.PosX > 4.05 {
		t.Fatal("dash did not stop at destination", s.char.PosX)
	}
	s.attackMoveUntil = time.Time{}
	if s.movement.credit != 0 {
		t.Fatal("dash granted reusable movement credit")
	}
}
