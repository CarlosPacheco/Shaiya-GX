package game

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

// MyShopSlot is one item the player lists in their own market.
type MyShopSlot struct {
	Used      bool
	Bag, Slot uint8
	Price     uint32
}

// MyShopState is the player's own personal market: 0 none, 1 setting up,
// 2 selling.
type MyShopState struct {
	Phase       int
	Slots       [proto.MyShopSlots]MyShopSlot
	BeginSerial uint64
	BeginResult uint8
	EndedSerial uint64 // the server closed the market (move, stand, death...)
	SoldSerial  uint64
	SoldIndex   uint8
	SoldCount   uint8
	SoldGold    uint32
}

// ShopViewState is another player's market the player has open.
type ShopViewState struct {
	Seller       uint32
	Open         bool
	Items        [proto.MyShopSlots]proto.MyShopItem
	Present      [proto.MyShopSlots]bool
	EnterSerial  uint64
	EnterResult  uint8
	BuySerial    uint64
	BuyResult    uint8
	ClosedSerial uint64
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
		s.MyShop.BeginSerial++
		if result == proto.MyShopBeginOK {
			s.MyShop = MyShopState{Phase: 1, BeginSerial: s.MyShop.BeginSerial, BeginResult: result, EndedSerial: s.MyShop.EndedSerial, SoldSerial: s.MyShop.SoldSerial}
		}
	case proto.MyShopCancel:
		if s.MyShop.Phase != 0 {
			s.MyShop.EndedSerial++
		}
		serials := s.MyShop
		s.MyShop = MyShopState{BeginSerial: serials.BeginSerial, EndedSerial: serials.EndedSerial, SoldSerial: serials.SoldSerial}
	case proto.MyShopOpen:
		if s.MyShop.Phase == 1 {
			s.MyShop.Phase = 2
		}
	case proto.MyShopStop:
		if s.MyShop.Phase == 2 {
			s.MyShop.Phase = 1
		}
	case proto.MyShopAdd:
		r := proto.NewReader(p)
		index, bag, slot, price := r.U8(), r.U8(), r.U8(), r.U32()
		if err := r.Err(); err != nil {
			return err
		}
		if int(index) < proto.MyShopSlots {
			s.MyShop.Slots[index] = MyShopSlot{true, bag, slot, price}
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
		s.MyShop.SoldIndex, s.MyShop.SoldCount, s.MyShop.SoldGold = index, count, gold
		s.MyShop.SoldSerial++
	case proto.MyShopEnter:
		r := proto.NewReader(p)
		result, id := r.U8(), r.U32()
		if err := r.Err(); err != nil {
			return err
		}
		s.ShopView.EnterResult = result
		s.ShopView.EnterSerial++
		if result == 1 {
			s.ShopView = ShopViewState{Seller: id, Open: true, EnterSerial: s.ShopView.EnterSerial, EnterResult: result, BuySerial: s.ShopView.BuySerial, ClosedSerial: s.ShopView.ClosedSerial}
		}
	case proto.MyShopList:
		items, err := proto.DecodeMyShopList(p)
		if err != nil {
			return err
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
		s.ShopView.BuySerial++
	case proto.MyShopClosed:
		serial := s.ShopView.ClosedSerial + 1
		s.ShopView = ShopViewState{ClosedSerial: serial, BuySerial: s.ShopView.BuySerial, EnterSerial: s.ShopView.EnterSerial}
	case proto.MyShopLeave:
		s.ShopView = ShopViewState{ClosedSerial: s.ShopView.ClosedSerial, BuySerial: s.ShopView.BuySerial, EnterSerial: s.ShopView.EnterSerial}
	case proto.ShopBubble:
		id, title, err := proto.DecodeShopBubble(p)
		if err != nil {
			return err
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
