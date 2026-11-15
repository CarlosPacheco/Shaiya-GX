package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skillrange"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func TestReachFailureSendsNativeLocalizedObstacleID(t *testing.T) {
	s, wire := gmWireSession(t, db.Character{ID: 1}, false)
	g := &mapgeometry.Geometry{}
	g.Reindex()
	s.movement.geometry = g
	for _, tri := range collision.BlockerBox(collision.Vec{4, -1, -2}, collision.Vec{5, 3, 2}, 0) {
		g.Add(tri)
	}
	s.sendSkillRangeFailure(db.CharacterSkill{SkillID: 33}, skillrange.Position{10, 0, 0})
	select {
	case p := <-wire:
		v, err := proto.DecodeSkillFailure(p)
		if err != nil || v.Skill != 33 || v.Message != 458 {
			t.Fatal(v, err)
		}
	case <-time.After(time.Second):
		t.Fatal("missing framed localized failure")
	}
}
