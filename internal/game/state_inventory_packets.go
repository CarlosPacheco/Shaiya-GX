package game

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func (s *State) applyInventoryPacket(p []byte, ev Event) (Event, error) {
	switch ev.Op {
	case proto.MapAddItem:
		it, err := proto.DecodeGroundItem(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.GroundItems[it.ID] = it
	case proto.MapRemoveItem:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return ev.packetError(err)
		}
		delete(s.GroundItems, id)
	case proto.CharacterItems:
		items, err := proto.DecodeInventory(p)
		if err != nil {
			return ev.packetError(err)
		}
		for _, it := range items {
			s.applyItem(it)
		}
		s.InventoryReady = true
	case proto.ItemBagToBag:
		items, money, hasMoney, err := proto.DecodeWarehouseMove(p)
		if err != nil {
			return ev.packetError(err)
		}
		for _, it := range items {
			s.applyItem(it)
		}
		if hasMoney {
			s.Details.Money = money
		}
	case proto.WarehouseItemList:
		list, err := proto.DecodeWarehouse(p)
		if err != nil {
			return ev.packetError(err)
		}
		stored := make(map[uint8]proto.InventoryItem, len(list.Items))
		for _, it := range list.Items {
			stored[it.Slot] = it
		}
		s.Warehouse, s.WarehouseGold, s.WarehouseReady = stored, list.Gold, true
	case proto.AddItem:
		if len(p) == 4 {
			ev.Text = fmt.Sprintf("item rejected: %d", p[3])
			break
		}
		it, err := proto.DecodeItemAdd(p)
		if err != nil {
			return ev.packetError(err)
		}
		// Rapture's gold acknowledgement uses a synthetic slot (26/0).
		// Only 0x0213 changes the balance; this packet must not replace a bag item.
		if it.Type != 26 || it.TypeID != 0 {
			s.applyItem(it)
		}
	case proto.RemoveItem:
		r := proto.NewReader(p)
		bag, slot, typ, id, count := r.U8(), r.U8(), r.U8(), r.U8(), r.Count()
		if err := r.Err(); err != nil {
			return ev.packetError(err)
		}
		key := [2]uint8{bag, slot}
		it := s.Inventory[key]
		it.Bag, it.Slot, it.Type, it.TypeID, it.Count = bag, slot, typ, id, count
		s.applyItem(it)
	case proto.SetMoney:
		money, err := proto.DecodeU32(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.Details.Money = money
	}
	return ev, nil
}
