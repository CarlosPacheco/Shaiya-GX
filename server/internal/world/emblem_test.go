package world

import (
	"encoding/binary"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func emblemSession(t *testing.T, gold uint32) *Session {
	t.Helper()
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{
		{Type: 24, TypeID: 1}: {Type: 24, TypeID: 1, Slot: 7},
		{Type: 1, TypeID: 1}:  {Type: 1, TypeID: 1, Slot: 1},
	}})
	s, _ := gmWireSession(t, db.Character{ID: 999999, Level: 20, Money: gold}, false)
	s.operation = &inventoryOperation{}
	s.items = []db.Item{
		{UID: 1, Bag: 0, Slot: 7, Type: 24, TypeID: 1, Count: 1},
		{UID: 2, Bag: 1, Slot: 0, Type: 24, TypeID: 1, Count: 1, Gems: [6]uint8{1, 2, 3, 4, 5, 6}},
		{UID: 3, Bag: 1, Slot: 1, Type: 1, TypeID: 1, Count: 1},
	}
	return s
}

func lastOperationPacket(s *Session, op proto.Opcode) []byte {
	for i := len(s.operation.packets) - 1; i >= 0; i-- {
		if f := s.operation.packets[i].frame; len(f) >= 4 && binary.LittleEndian.Uint16(f[2:]) == uint16(op) {
			return f
		}
	}
	return nil
}

func TestCloakEmblemChargesAndStoresPattern(t *testing.T) {
	s := emblemSession(t, 500000)
	data := [6]uint8{3, 4, 5, 6, 7, 8}
	s.planCloakAdd(proto.EmblemApplyPacket(0, 7, data))
	if s.char.Money != 50000 || s.items[0].Gems != data {
		t.Fatalf("money %d gems %v", s.char.Money, s.items[0].Gems)
	}
	frame := lastOperationPacket(s, proto.CloakAdd)
	if len(frame) != 2+15 {
		t.Fatalf("reply is %d bytes, want the original 15 plus the length word", len(frame))
	}
	got, err := proto.DecodeEmblemResult(frame[2:])
	if err != nil || got.Result != 1 || got.Bag != 0 || got.Slot != 7 || got.Pattern != data || got.Money != 50000 {
		t.Fatalf("reply %+v %v", got, err)
	}
	if lastOperationPacket(s, 0x0507) == nil {
		t.Fatal("equipped cloak did not publish its equipment")
	}
}

func TestCloakEmblemRefusedWithoutGold(t *testing.T) {
	s := emblemSession(t, 449999)
	data := [6]uint8{1, 1, 1, 35, 35, 35}
	s.planCloakAdd(proto.EmblemApplyPacket(1, 0, data))
	if s.char.Money != 449999 || s.items[1].Gems != [6]uint8{1, 2, 3, 4, 5, 6} {
		t.Fatal("a refused request changed the cloak or the gold")
	}
	got, err := proto.DecodeEmblemResult(lastOperationPacket(s, proto.CloakAdd)[2:])
	if err != nil || got.Result != 0 || got.Pattern != data || got.Money != 449999 {
		t.Fatalf("reply %+v %v", got, err)
	}
	if lastOperationPacket(s, 0x0507) != nil {
		t.Fatal("a carried cloak must not publish equipment")
	}
}

func TestCloakEmblemRejectsOtherItems(t *testing.T) {
	s := emblemSession(t, 1000000)
	for _, at := range [][2]uint8{{1, 1}, {2, 2}, {9, 0}, {0, 30}} {
		s.planCloakAdd(proto.EmblemApplyPacket(at[0], at[1], [6]uint8{1}))
	}
	if s.char.Money != 1000000 || len(s.operation.packets) != 0 {
		t.Fatal("a non-cloak request was answered")
	}
}

func TestCloakEmblemRemoveIsFree(t *testing.T) {
	s := emblemSession(t, 7)
	s.planCloakRemove(proto.EmblemRemovePacket(1, 0))
	if s.items[1].Gems != [6]uint8{} || s.char.Money != 7 {
		t.Fatal("emblem not cleared for free")
	}
	got, err := proto.DecodeEmblemRemoved(lastOperationPacket(s, proto.CloakRemove)[2:])
	if err != nil || got != (proto.EmblemRemoved{Bag: 1, Slot: 0}) {
		t.Fatalf("reply %+v %v", got, err)
	}
}
