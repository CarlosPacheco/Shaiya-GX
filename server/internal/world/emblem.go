package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// Cloak emblems (the 0x08xx group handled by CUser::PacketGem, 0x0046bd40).
// 0x0803 is CUser::ItemCloakAdd (0x00464ad0) and 0x0804 CUser::ItemCloakRemove
// (0x00464d40). The pattern is stored in the cloak's six gem bytes: front,
// back and emblem shape (index + 1, 0 = none) followed by the palette index
// of each. The NPC is not validated by the original handlers.
const emblemFee = 450000

func buildEmblemAddResult(result, bag, slot uint8, data [6]uint8, gold uint32) []byte {
	w := packet.New(packet.CloakAdd)
	w.Bytes([]byte{result, bag, slot})
	w.Bytes(data[:])
	w.U32(gold)
	return w.MustFinish()
}

// emblemTarget returns the index of a carried cloak. The original accepts only
// a populated slot whose item definition type is 24.
func (s *Session) emblemTarget(bag, slot uint8) int {
	if !isValidItemPosition(bag, slot) {
		return -1
	}
	i := findItem(s.items, bag, slot)
	if i < 0 || db.MustDefs().Item(s.items[i].Type, s.items[i].TypeID) == nil || realType(s.items[i].Type) != 24 {
		return -1
	}
	return i
}

func (s *Session) handleCloakAdd(p []byte) { s.inventoryOperation(func() { s.planCloakAdd(p) }) }
func (s *Session) planCloakAdd(p []byte) {
	if len(p) < 10 || s.char == nil {
		return
	}
	ch := s.char
	bag, slot := p[2], p[3]
	i := s.emblemTarget(bag, slot)
	if i < 0 {
		return
	}
	var data [6]uint8
	copy(data[:], p[4:10])
	if ch.Money < emblemFee {
		s.send(buildEmblemAddResult(0, bag, slot, data, ch.Money), "0x0803 cloak emblem refused")
		return
	}
	ch.Money -= emblemFee
	s.items[i].Gems = data
	s.saveInventoryItem(&s.items[i])
	s.send(buildEmblemAddResult(1, bag, slot, data, ch.Money), "0x0803 cloak emblem applied")
	if bag == 0 && slot == 7 {
		s.equipmentChanged([]uint8{7})
	}
	s.log.Printf("GX Server World cloak emblem character=%d bag=%d slot=%d data=%v", ch.ID, bag, slot, data)
}

func (s *Session) handleCloakRemove(p []byte) { s.inventoryOperation(func() { s.planCloakRemove(p) }) }
func (s *Session) planCloakRemove(p []byte) {
	if len(p) < 4 || s.char == nil {
		return
	}
	bag, slot := p[2], p[3]
	i := s.emblemTarget(bag, slot)
	if i < 0 {
		return
	}
	s.items[i].Gems = [6]uint8{}
	s.saveInventoryItem(&s.items[i])
	w := packet.New(packet.CloakRemove)
	w.Bytes([]byte{bag, slot})
	s.send(w.MustFinish(), "0x0804 cloak emblem removed")
	if bag == 0 && slot == 7 {
		s.equipmentChanged([]uint8{7})
	}
}
