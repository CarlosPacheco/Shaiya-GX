package app

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func relicPacket(op proto.Opcode, body ...any) []byte {
	p := binary.LittleEndian.AppendUint16(nil, uint16(op))
	for _, v := range body {
		switch x := v.(type) {
		case uint8:
			p = append(p, x)
		case uint16:
			p = binary.LittleEndian.AppendUint16(p, x)
		case uint32:
			p = binary.LittleEndian.AppendUint32(p, x)
		case float32:
			p = binary.LittleEndian.AppendUint32(p, math.Float32bits(x))
		}
	}
	return p
}

func TestRelicNoticeMessage(t *testing.T) {
	for _, c := range []struct {
		old, new, faction uint8
		id                int
		ok                bool
	}{
		{0, 1, 0, 463, true}, {0, 1, 1, 0, false}, {0, 2, 1, 463, true}, {0, 2, 0, 0, false},
		{1, 2, 0, 462, true}, {1, 2, 1, 463, true}, {2, 1, 0, 463, true}, {2, 1, 1, 462, true},
		{1, 1, 0, 0, false},
	} {
		id, ok := relicNoticeMessage(c.old, c.new, c.faction)
		if id != c.id || ok != c.ok {
			t.Errorf("relicNoticeMessage(%d,%d,%d) = %d,%v", c.old, c.new, c.faction, id, ok)
		}
	}
}

func TestRelicStateAndMarkers(t *testing.T) {
	s := game.New()
	if _, err := s.Apply(relicPacket(proto.RelicList, uint8(2), uint32(1), uint8(0), float32(512), float32(1024),
		uint32(2), uint8(1), float32(0), float32(0))); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(relicPacket(proto.RelicOwner, uint32(1), uint8(2))); err != nil {
		t.Fatal(err)
	}
	if len(s.Relics) != 2 || s.Relics[0].Owner != 2 || s.Relics[1].Owner != 1 {
		t.Fatalf("relics = %+v", s.Relics)
	}
	markers := relicMapMarkers(s, 2048, nil)
	if len(markers) != 2 || markers[0].Kind != ui.MapRelicFury || markers[1].Kind != ui.MapRelicLight {
		t.Fatalf("markers = %+v", markers)
	}
	if _, err := s.Apply(relicPacket(proto.RelicNotice, uint16(52), uint8(0), uint8(1))); err != nil {
		t.Fatal(err)
	}
	if s.LastRelicNotice != (proto.RelicDestroyed{MobID: 52, OldOwner: 0, NewOwner: 1}) {
		t.Fatalf("notice = %+v", s.LastRelicNotice)
	}
}

func TestRelicPortalMarkersAndOriginalMapGates(t *testing.T) {
	s := game.New()
	s.Relics = []proto.Relic{{ID: 0, Owner: 100, X: 10, Z: 10}, {ID: 1, Owner: 101, X: 20, Z: 20}, {ID: 2, Owner: 102, X: 30, Z: 30}, {ID: 3, Owner: 103, X: 40, Z: 40}, {ID: 4, Owner: 1, X: 50, Z: 50}}
	for _, faction := range []uint8{0, 1} {
		s.Faction.Faction = faction
		markers := relicMapMarkers(s, 100, nil)
		if len(markers) != 4 {
			t.Fatal(markers)
		}
		for _, m := range markers[:3] {
			if m.Kind != ui.MapRelicPortal {
				t.Fatal("open portal icon missing")
			}
		}
	}
	for _, mapID := range []uint16{68, 69, 105, 108} {
		s.MapID = mapID
		if len(relicMapMarkers(s, 100, nil)) != 0 {
			t.Fatalf("native map %d gate ignored", mapID)
		}
	}
	s.MapID = 100
	if m := relicMapMarkers(s, 100, nil); len(m) != 1 || m[0].Kind != ui.MapRelicLight {
		t.Fatal("instance portal gate")
	}

	for _, mapID := range []uint16{0, 18, 30, 70} {
		if relicNoticeStyle(mapID) != 8 {
			t.Fatal("missing native central notice")
		}
	}
	if relicNoticeStyle(81) != 7 {
		t.Fatal("unverified other-build notice routing leaked into original reference")
	}
}
