package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestEmblemRepliesUpdateTheCloak(t *testing.T) {
	s := New()
	key := [2]uint8{0, 7}
	s.Inventory[key] = proto.InventoryItem{Bag: 0, Slot: 7, Type: 24, TypeID: 1, Count: 1}
	w := proto.NewWriter(proto.CloakAdd)
	w.Raw([]byte{1, 0, 7, 3, 4, 5, 6, 7, 8})
	w.U32(55000)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Inventory[key].Gems != [6]uint8{3, 4, 5, 6, 7, 8} || s.Details.Money != 55000 || s.EmblemSerial != 1 {
		t.Fatalf("%+v %d", s.Inventory[key], s.Details.Money)
	}
	refused := proto.NewWriter(proto.CloakAdd)
	refused.Raw([]byte{0, 0, 7, 9, 9, 9, 9, 9, 9})
	refused.U32(100)
	if _, err := s.Apply(refused.Bytes()); err != nil || s.Inventory[key].Gems != [6]uint8{3, 4, 5, 6, 7, 8} || s.LastEmblem.Result != 0 {
		t.Fatal("a refusal changed the cloak")
	}
	if _, err := s.Apply(proto.NewWriter(proto.CloakRemove).Bytes()); err == nil {
		t.Fatal("short removal accepted")
	}
	r := proto.NewWriter(proto.CloakRemove)
	r.Raw([]byte{0, 7})
	if _, err := s.Apply(r.Bytes()); err != nil || s.Inventory[key].Gems != [6]uint8{} || s.EmblemRemoveSerial != 1 {
		t.Fatal("removal not applied")
	}
}
