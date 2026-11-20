package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func shopSession(mapID uint16) *Session {
	p := &ConnectedPlayer{CharacterID: 7, CharacterName: "Seller"}
	s := &Session{player: p, char: &db.Character{ID: 7, MapID: mapID, CurrentHP: 100}}
	s.items = []db.Item{{UID: 1, Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 10}, {UID: 2, Bag: 1, Slot: 1, Type: 27, TypeID: 1, Count: 1}}
	return s
}

func addFrame(idx, bag, slot uint8, price uint32) []byte {
	w := packet.New(opMyShopAdd)
	w.U8(idx)
	w.U8(bag)
	w.U8(slot)
	w.U32(price)
	return w.MustFinish()[2:] // dispatch hands handlers the opcode and body, no length
}

func TestMyShopOnlyInAllowedMaps(t *testing.T) {
	s := shopSession(1)
	s.handleMyShop(opMyShopBegin, packet.New(opMyShopBegin).MustFinish()[2:])
	if s.shop.status != myShopStatusNone {
		t.Fatal("a market was started in a map that does not allow it")
	}
	for _, m := range []uint16{35, 36, 42} {
		s = shopSession(m)
		s.handleMyShop(opMyShopBegin, packet.New(opMyShopBegin).MustFinish()[2:])
		if s.shop.status != myShopStatusSetup {
			t.Fatalf("map %d refused", m)
		}
	}
}

func TestMyShopSetupAndOpen(t *testing.T) {
	s := shopSession(35)
	s.handleMyShop(opMyShopBegin, nil)
	// A quest item and a zero price are refused; a normal item is listed.
	s.handleMyShop(opMyShopAdd, addFrame(0, 1, 1, 50))
	s.handleMyShop(opMyShopAdd, addFrame(0, 1, 0, 0))
	if s.shop.slots[0].used {
		t.Fatal("an invalid listing was accepted")
	}
	s.handleMyShop(opMyShopAdd, addFrame(2, 1, 0, 50))
	s.handleMyShop(opMyShopAdd, addFrame(3, 1, 0, 70)) // the same item twice
	if !s.shop.slots[2].used || s.shop.slots[3].used {
		t.Fatalf("slots %+v", s.shop.slots[:4])
	}
	price := packet.New(opMyShopPrice)
	price.U8(2)
	price.U32(90)
	s.handleMyShop(opMyShopPrice, price.MustFinish()[2:])
	if s.shop.slots[2].price != 90 {
		t.Fatal("price not changed")
	}
	// Open needs a message.
	empty := packet.New(opMyShopOpen)
	empty.U8(0)
	s.handleMyShop(opMyShopOpen, empty.MustFinish()[2:])
	if s.shop.status != myShopStatusSetup {
		t.Fatal("opened without a message")
	}
	w := packet.New(opMyShopOpen)
	w.U8(5)
	w.Bytes([]byte("Hello"))
	s.handleMyShop(opMyShopOpen, w.MustFinish()[2:])
	if s.shop.status != myShopStatusSelling || s.player.currentShopTitle() != "Hello" || !s.player.IsSitting() {
		t.Fatalf("status %d title %q sitting %v", s.shop.status, s.player.currentShopTitle(), s.player.IsSitting())
	}
	// Items cannot be edited while selling, and the list carries what is for sale.
	s.handleMyShop(opMyShopRemove, func() []byte { w := packet.New(opMyShopRemove); w.U8(2); return w.MustFinish()[2:] }())
	if !s.shop.slots[2].used {
		t.Fatal("edited while selling")
	}
	list := s.myShopList()
	if list[4] != 1 || len(list) != 4+1+37 {
		t.Fatalf("list %d bytes, count %d", len(list), list[4])
	}
	s.endMyShop()
	if s.shop.status != myShopStatusNone || s.player.currentShopTitle() != "" {
		t.Fatal("the market did not end")
	}
}
