package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/chatguard"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// MyShopSlot is one item the player lists in their own market.
type MyShopSlot struct {
	Used      bool
	Bag, Slot uint8
	Price     uint32
	Item      proto.InventoryItem
}

// MyShopState is the player's own personal market: 0 none, 1 setting up,
// 2 selling.
type MyShopState struct {
	Phase       int
	Slots       [proto.MyShopSlots]MyShopSlot
	BeginResult uint8
	SoldCount   uint8
	SoldGold    uint32
	SoldItem    proto.InventoryItem
}

// ShopViewState is another player's market the player has open.
type ShopViewState struct {
	Seller      uint32
	Open        bool
	Items       [proto.MyShopSlots]proto.MyShopItem
	Present     [proto.MyShopSlots]bool
	EnterResult uint8
	BuyResult   uint8
}

func (s *State) applyMyShop(p []byte) error {
	switch op := proto.Op(p); op {
	case proto.MyShopBegin:
		r := proto.NewReader(p)
		result := r.U8()
		if err := r.Err(); err != nil {
			return err
		}
		s.MyShop.BeginResult = result
		if result == proto.MyShopBeginOK {
			s.MyShop = MyShopState{Phase: proto.MyShopSetup, BeginResult: result}
		}
	case proto.MyShopCancel:
		s.MyShop = MyShopState{}
	case proto.MyShopOpen:
		if s.MyShop.Phase == proto.MyShopSetup {
			s.MyShop.Phase = proto.MyShopSelling
		}
	case proto.MyShopStop:
		if s.MyShop.Phase == proto.MyShopSelling {
			s.MyShop.Phase = proto.MyShopSetup
		}
	case proto.MyShopAdd:
		r := proto.NewReader(p)
		index, bag, slot, price := r.U8(), r.U8(), r.U8(), r.U32()
		if err := r.Err(); err != nil {
			return err
		}
		if int(index) < proto.MyShopSlots && s.MyShop.Phase == proto.MyShopSetup {
			s.MyShop.Slots[index] = MyShopSlot{Used: true, Bag: bag, Slot: slot, Price: price, Item: s.Inventory[[2]uint8{bag, slot}]}
		}
	case proto.MyShopRemove:
		r := proto.NewReader(p)
		index := r.U8()
		if err := r.Err(); err != nil {
			return err
		}
		if int(index) < proto.MyShopSlots {
			s.MyShop.Slots[index] = MyShopSlot{}
		}
	case proto.MyShopPrice:
		r := proto.NewReader(p)
		index, price := r.U8(), r.U32()
		if err := r.Err(); err != nil {
			return err
		}
		if int(index) < proto.MyShopSlots && s.MyShop.Slots[index].Used {
			s.MyShop.Slots[index].Price = price
		}
	case proto.MyShopSold:
		r := proto.NewReader(p)
		index, count, gold := r.U8(), r.U8(), r.U32()
		if err := r.Err(); err != nil {
			return err
		}
		s.MyShop.SoldCount, s.MyShop.SoldGold = count, gold
		s.MyShop.SoldItem = proto.InventoryItem{}
		if int(index) < proto.MyShopSlots {
			s.MyShop.SoldItem = s.MyShop.Slots[index].Item
		}
	case proto.MyShopEnter:
		r := proto.NewReader(p)
		result, id := r.U8(), r.U32()
		if err := r.Err(); err != nil {
			return err
		}
		s.ShopView.EnterResult = result
		if result == proto.MyShopEnterOK {
			s.ShopView = ShopViewState{Seller: id, Open: true, EnterResult: result}
		}
	case proto.MyShopList:
		items, err := proto.DecodeMyShopList(p)
		if err != nil {
			return err
		}
		if !s.ShopView.Open {
			return nil
		}
		s.ShopView.Items, s.ShopView.Present = [proto.MyShopSlots]proto.MyShopItem{}, [proto.MyShopSlots]bool{}
		for _, it := range items {
			s.ShopView.Items[it.Index], s.ShopView.Present[it.Index] = it, true
		}
	case proto.MyShopStock:
		r := proto.NewReader(p)
		index, count := r.U8(), r.U8()
		if err := r.Err(); err != nil {
			return err
		}
		if int(index) < proto.MyShopSlots && s.ShopView.Present[index] {
			if count == 0 {
				s.ShopView.Present[index] = false
			} else {
				s.ShopView.Items[index].Item.Count = uint16(count)
			}
		}
	case proto.MyShopBuy:
		r := proto.NewReader(p)
		result := r.U8()
		if err := r.Err(); err != nil {
			return err
		}
		s.ShopView.BuyResult = result
	case proto.MyShopClosed, proto.MyShopLeave:
		s.ShopView = ShopViewState{}
	case proto.ShopBubble:
		id, title, err := proto.DecodeShopBubble(p)
		if err != nil {
			return err
		}
		checked := chatguard.Filter(title, false)
		title = checked.Text
		if checked.Impersonation {
			title = "[Message hidden]"
		}
		if title == "" {
			delete(s.ShopSigns, id)
		} else if id != 0 && (id == s.SelfID || s.Entities[id] != nil) {
			if s.ShopSigns == nil {
				s.ShopSigns = make(map[uint32]string)
			}
			s.ShopSigns[id] = title
		}
		if e := s.Entities[id]; e != nil {
			e.ShopTitle = title
		}
		if id == s.SelfID {
			s.Self.ShopTitle = title
		}
	}
	return nil
}
