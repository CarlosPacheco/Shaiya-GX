package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// CUser::PacketMaininterface 0x226 scans the 13 equipment slots for sex-only
// items (Special 64/65) and answers 0x0228 instead of changing sex.
func TestGenderRestrictedEquipmentBlocksSexChange(t *testing.T) {
	s := &Session{}
	if s.genderRestrictedEquipment() {
		t.Fatal("empty equipment blocked the change")
	}
	for _, id := range []uint8{171, 172} { // Valentine rings: male-only, female-only
		s.items = []db.Item{{Bag: 0, Slot: 9, Type: 22, TypeID: id}}
		if !s.genderRestrictedEquipment() {
			t.Fatalf("worn sex-only item %d did not block", id)
		}
	}
	s.items = []db.Item{{Bag: 1, Slot: 1, Type: 22, TypeID: 171}} // carried, not worn
	if s.genderRestrictedEquipment() {
		t.Fatal("a carried ring blocked the change")
	}
}
