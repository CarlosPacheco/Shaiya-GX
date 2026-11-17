package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestMapLoadAcknowledgementBoundaries(t *testing.T) {
	s := New()
	s.SelfID = 10
	s.MapID = 1
	s.Phase = PhaseLoading
	placement := func(id uint32, mapID uint16) Event {
		t.Helper()
		w := proto.NewWriter(proto.CharacterMapTeleport)
		w.U32(id)
		w.U16(mapID)
		w.F32(1)
		w.F32(2)
		w.F32(3)
		ev, err := s.Apply(w.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		return ev
	}
	if placement(10, 18).MapLoadAck {
		t.Fatal("initial placement acknowledged before inventory/encryption boundary")
	}
	bar := proto.NewWriter(proto.CharacterSkillBar)
	bar.U8(0)
	ev, err := s.Apply(bar.Bytes())
	if err != nil || !ev.MapLoadAck {
		t.Fatal("initial skill-bar boundary not acknowledged", err)
	}
	s.Phase = PhaseInWorld
	for _, mapID := range []uint16{45, 1, 18, 45, 1} {
		if !placement(10, mapID).MapLoadAck {
			t.Fatalf("map %d transfer not acknowledged", mapID)
		}
		if placement(10, mapID).MapLoadAck {
			t.Fatal("same-map teleport/entry echo acknowledged again")
		}
	}
	if placement(99, 45).MapLoadAck || s.MapID != 1 {
		t.Fatal("foreign teleport acknowledged")
	}
	ev, err = s.Apply(bar.Bytes())
	if err != nil || ev.MapLoadAck {
		t.Fatal("in-world skillbar update acknowledged", err)
	}
	ev, err = s.Apply([]byte{0x0b, 0x02})
	if err == nil || ev.MapLoadAck {
		t.Fatal("malformed placement acknowledged")
	}
}
