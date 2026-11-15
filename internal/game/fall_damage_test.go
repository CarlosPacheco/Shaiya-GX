package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestFallNotificationDoesNotPredictHealthAndRejectsTruncation(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.Pools.HP = 100
	w := proto.NewWriter(proto.GXFallDamage)
	w.U32(1)
	w.U32(70000)
	p := w.Bytes()
	if _, err := s.Apply(p[:len(p)-1]); err == nil {
		t.Fatal("truncated event accepted")
	}
	if s.LastFallDamage != (proto.FallDamage{}) {
		t.Fatal("failed parse mutated state")
	}
	if _, err := s.Apply(p); err != nil {
		t.Fatal(err)
	}
	if s.LastFallDamage.Amount != 70000 || s.Pools.HP != 100 {
		t.Fatal("fall notice changed authoritative pools or truncated amount")
	}
}

func TestEnvironmentalDeathKeepsLossFreeRespawnPresentation(t *testing.T) {
	s := New()
	s.SelfID = 1
	w := proto.NewWriter(proto.CharacterDeath)
	w.U32(1)
	w.U8(2)
	w.U32(0)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if !s.Self.Dead || !s.DeathLossFree {
		t.Fatal("environment death displayed PvP or XP penalties")
	}
}
