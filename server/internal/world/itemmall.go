package world

import (
	"errors"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const (
	pointBuyOK              = 0
	pointBuyNoProduct       = 1
	pointBuyNotEnoughPoints = 2
	pointBuyNotEnoughSlots  = 3

	giftProductNotFound = 6
	giftNotEnoughPoints = 10
)

func time32() uint32 { return uint32(time.Now().Unix()) }

type grantedItem struct {
	Bag, Slot, Type, TypeID uint8
	Count                   uint16
}

func buildPointBuyResult(result uint8, points uint32, code string, cost uint32, items []grantedItem) []byte {
	w := packet.New(packet.PointBuy)
	w.U8(result)
	w.U32(points)
	w.FixedString(code, 21)
	w.U32(time32())
	w.U32(cost)
	w.U8(uint8(len(items)))
	for _, g := range items {
		w.Bytes([]byte{g.Bag, g.Slot, g.Type, g.TypeID})
		w.Count(g.Count)
	}
	return w.MustFinish()
}

func buildGiftSendResult(result uint8, target string, points uint32, code string, cost uint32) []byte {
	w := packet.New(packet.PointGiftSend)
	w.U8(result)
	w.FixedString(target, 21)
	w.U32(points)
	w.FixedString(code, 21)
	w.U32(time32())
	w.U32(cost)
	return w.MustFinish()
}

// buildGiftBoxList (0x2606) lists UserStoredPointItems (slots 0-239).
func buildGiftBoxList(userUID uint32) ([]byte, error) {
	gifts, err := db.GiftBox(userUID)
	if err != nil {
		return nil, err
	}
	w := packet.New(packet.PointStoredItemList)
	w.U8(uint8(min(len(gifts), db.GiftBoxSlots)))
	for _, g := range gifts {
		w.Bytes([]byte{g.Slot, g.Type, g.TypeID})
		w.Count(g.Count)
	}
	return w.MustFinish(), nil
}

func (s *Session) sendGiftBox() {
	if list, err := buildGiftBoxList(s.accountID); err == nil {
		s.send(list, "0x2606 gift box list")
	} else {
		s.log.Printf("GX Server World failed to read gift box: %v", err)
	}
}

// handlePointBuy (0x2602): product code.
func (s *Session) handlePointBuy(p []byte) {
	if s.char == nil {
		return
	}
	code := ""
	if len(p) > 2 {
		code = packet.FixedText(p[2:])
	}
	prod := db.MustDefs().Products[code]
	if prod == nil {
		s.log.Printf("GX Server World point buy: unknown ProductList product %q", code)
		s.send(buildPointBuyResult(pointBuyNoProduct, s.points, code, 0, nil), "0x2602 point buy result")
		return
	}
	if s.points < prod.BuyCost {
		s.send(buildPointBuyResult(pointBuyNotEnoughPoints, s.points, code, prod.BuyCost, nil), "0x2602 point buy result")
		return
	}
	free := freeInventorySlots(s.items)
	var delivery []db.Item
	defs := db.MustDefs()
	for _, pi := range prod.Items {
		def := defs.Item(pi.Type, pi.TypeID)
		if def == nil || pi.Count == 0 {
			s.send(buildPointBuyResult(pointBuyNoProduct, s.points, code, 0, nil), "invalid mall definition")
			return
		}
		for remaining := pi.Count; remaining > 0; {
			if len(delivery) >= len(free) {
				s.send(buildPointBuyResult(pointBuyNotEnoughSlots, s.points, code, prod.BuyCost, nil), "0x2602 point buy result")
				return
			}
			n := min(remaining, def.MaxStack())
			slot := free[len(delivery)]
			delivery = append(delivery, db.Item{Bag: slot[0], Slot: slot[1], Type: pi.Type, TypeID: pi.TypeID, Quality: def.Quality, Count: n, MakeType: "B"})
			remaining -= n
		}
	}
	if len(delivery) == 0 {
		s.send(buildPointBuyResult(pointBuyNoProduct, s.points, code, 0, nil), "empty mall definition")
		return
	}
	left, err := db.PurchaseMallItems(s.accountID, s.char.ID, prod, delivery)
	if err != nil {
		s.log.Printf("GX Server World point buy transaction failed: %v", err)
		result := uint8(pointBuyNoProduct)
		if errors.Is(err, db.ErrNotEnoughPoints) {
			result = pointBuyNotEnoughPoints
			if balance, e := db.Points(s.accountID); e == nil {
				s.points = balance
			}
		}
		s.send(buildPointBuyResult(result, s.points, code, prod.BuyCost, nil), "0x2602 point buy result")
		return
	}
	s.points = left
	var granted []grantedItem
	for _, it := range delivery {
		s.items = append(s.items, it)
		s.send(buildAddItem(&it), "0x0205 point buy add item")
		granted = append(granted, grantedItem{it.Bag, it.Slot, it.Type, it.TypeID, it.Count})
	}

	s.log.Printf("GX Server World point buy product=%s cost=%d items=%d points_left=%d", code, prod.BuyCost, len(granted), s.points)
	s.send(buildPointBuyResult(pointBuyOK, s.points, code, prod.BuyCost, granted), "0x2602 point buy result")
}

// handleGiftSend (0x2603): product code[21], target name[21].
func (s *Session) handleGiftSend(p []byte) {
	if len(p) < 44 || s.char == nil {
		return
	}
	code, target := packet.FixedText(p[2:23]), packet.FixedText(p[23:44])
	reply := func(result uint8, cost uint32) {
		s.send(buildGiftSendResult(result, target, s.points, code, cost), "0x2603 gift send result")
	}
	prod := db.MustDefs().Products[code]
	if prod == nil {
		reply(giftProductNotFound, 0)
		return
	}
	if s.points < prod.BuyCost {
		reply(giftNotEnoughPoints, prod.BuyCost)
		return
	}
	result, targetUID, left, err := db.PurchaseGift(target, s.accountID, s.char.ID, prod)
	if err != nil {
		s.log.Printf("GX Server World gift transaction failed: %v", err)
		if errors.Is(err, db.ErrNotEnoughPoints) {
			result = giftNotEnoughPoints
			if balance, e := db.Points(s.accountID); e == nil {
				s.points = balance
			}
		}
	}
	if result != db.GiftOK {
		reply(result, prod.BuyCost)
		return
	}

	s.points = left
	s.log.Printf("GX Server World gift sent product=%s to=%q account=%d", code, target, targetUID)
	reply(db.GiftOK, prod.BuyCost)
	// An online recipient sees the gift without relogging.
	if rp := FindPlayerByName(target); rp != nil {
		if list, err := buildGiftBoxList(targetUID); err == nil {
			rp.Send(list, "0x2606 gift box push")
		}
	}
}

// handleGiftClaim (0x2607): gift-box slot.
func (s *Session) handleGiftClaim(p []byte) {
	if len(p) < 3 || s.char == nil || p[2] >= db.GiftBoxSlots {
		return
	}
	slot := p[2]
	gifts, err := db.GiftBox(s.accountID)
	if err != nil {
		return
	}
	var entry *db.GiftEntry
	for i := range gifts {
		if gifts[i].Slot == slot {
			entry = &gifts[i]
		}
	}
	if entry == nil {
		return
	}
	def := db.MustDefs().Item(entry.Type, entry.TypeID)
	free := freeInventorySlots(s.items)
	if def == nil || len(free) == 0 {
		return
	}
	it := db.Item{Bag: free[0][0], Slot: free[0][1], Type: entry.Type, TypeID: entry.TypeID, Quality: def.Quality,
		Count: min(entry.Count, def.MaxStack()), MakeType: "B"}
	ok, err := db.ClaimGift(s.accountID, s.char.ID, slot, &it)
	if err != nil || !ok {
		s.log.Printf("GX Server World gift claim failed slot=%d: %v", slot, err)
		return
	}
	s.items = append(s.items, it)
	s.log.Printf("GX Server World gift claimed slot=%d item=%d/%d", slot, it.Type, it.TypeID)
	s.send(tradePacket(packet.PointStoredItemGet, slot, it.Bag, it.Slot), "0x2607 gift claim result")
}
