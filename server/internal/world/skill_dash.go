package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"math"
	"time"
)

// Stinger is a bounded server-authored sweep, never a client speed allowance.
// The selected target is captured once, as in the original result handler.
func (s *Session) beginStinger(def *db.SkillDef, destination [3]float32) bool {
	if def.SkillID != 56 {
		return true
	}
	if s.initPlayerMovement(time.Now()) != nil || s.dead || s.char.CurrentHP == 0 || s.movement.jumping || s.movement.ladder.Active || s.movement.geometry.DeepWater(s.movement.pos) {
		return false
	}
	from := s.movement.pos
	to, ok := s.movement.geometry.Spawn(mapgeometry.Position(destination))
	if !ok {
		s.sendSkillMessage(def.SkillID, 458)
		return false
	}
	delta := mapgeometry.Position{to[0] - from[0], to[1] - from[1], to[2] - from[2]}
	end, _ := s.movement.geometry.MoveFeet(from, delta)
	if math.Hypot(float64(end[0]-to[0]), float64(end[2]-to[2])) > .12 || math.Abs(float64(end[1]-to[1])) > .12 {
		s.sendSkillMessage(def.SkillID, 458)
		return false
	}
	d := proto.SkillDash{Source: s.char.ID, From: [3]float32(from), To: [3]float32(end), Started: mapgeometry.PhysicsSeconds(), Contact: skilltiming.StingerContact(s.char.Race, s.char.Gender)}
	if d.Distance() > proto.MaxSkillDashDistance {
		return false
	}
	s.dash = &d
	s.autoAttack = false
	s.broadcastNear(buildSkillDash(d), "Stinger trajectory", true)
	return true
}

func (s *Session) tickStinger() bool {
	d := s.dash
	if d == nil {
		return false
	}
	if s.char == nil || s.dead || s.char.CurrentHP == 0 || s.ridingVehicle() || s.movement.geometry == nil {
		s.dash = nil
		return false
	}
	elapsed := mapgeometry.PhysicsSeconds() - d.Started
	want := mapgeometry.Position(d.Position(elapsed))
	old := s.movement.pos
	delta := mapgeometry.Position{want[0] - old[0], want[1] - old[1], want[2] - old[2]}
	next, hit := s.movement.geometry.MoveFeet(old, delta)
	s.movement.pos = next
	if s.recoverPlayerHeight() {
		return true
	}
	s.movement.at = time.Now()
	s.movement.budgetAt = s.movement.at
	s.movement.credit = 0
	s.commitMovement()
	s.positionDirty = true
	if next != old {
		if s.player != nil && s.player.moveViewCell(next[0], next[2]) {
			s.refreshView()
			s.sendToViewers(buildSkillDash(*d), "Stinger new view cell")
		}
		s.duelCheckMove()
		s.sendToViewers(buildCharacterMove(s.char, 0, s.movement.direction), "Stinger movement")
	}
	if s.cannotAct() || hit && math.Hypot(float64(next[0]-want[0]), float64(next[2]-want[2])) > .05 || elapsed >= d.Duration() {
		s.dash = nil
		stopped := proto.SkillDash{Source: s.char.ID, From: [3]float32(next), To: [3]float32(next), Started: mapgeometry.PhysicsSeconds(), Contact: 10, Stopped: true}
		s.broadcastNear(buildSkillDash(stopped), "Stinger stopped", true)
		s.movement.packetAt = time.Now()
		s.correctMovement("Stinger complete")
		s.sendToViewers(buildCharacterMove(s.char, 0, s.movement.direction), "Stinger complete")
	}
	return true
}

func buildSkillDash(d proto.SkillDash) []byte {
	raw := proto.SkillDashPacket(d)
	w := packet.New(packet.Opcode(proto.GXSkillDash))
	w.Bytes(raw[2:])
	return w.MustFinish()
}
