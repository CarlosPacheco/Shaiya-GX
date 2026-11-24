package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestShopSaleRespectsGoldCap(t *testing.T) {
	for _, tc := range []struct {
		name string
		gold uint32
		want uint8
	}{
		{"at cap", db.MoneyCap, 1},
		{"would exceed cap", db.MoneyCap - 19, 1},
		{"reaches cap exactly", db.MoneyCap - 20, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture, add := forgeRuleSession(t)
			fixture.char.Money = tc.gold
			if err := db.SaveCharacter(fixture.char); err != nil {
				t.Fatal(err)
			}
			item := add(db.Item{Bag: 1, Slot: 0, Type: 25, TypeID: 43, Count: 3})
			db.MustDefs().Item(item.Type, item.TypeID).Sell = 10
			s, wire := gmWireSession(t, *fixture.char, false)
			s.items = fixture.items
			s.handleShopSell(proto.ShopSellPacket(item.Bag, item.Slot, 2))
			r, err := proto.DecodeShopResult(waitWorldPacket(t, wire, proto.ShopSell))
			gold, count := tc.gold, item.Count
			if tc.want == 0 {
				gold, count = db.MoneyCap, 1
			}
			if err != nil || r.Result != tc.want || r.Bag != item.Bag || r.Count != count || r.Money != gold {
				t.Fatalf("shop response = %+v, error = %v", r, err)
			}
			stored, err := db.CharacterItems(s.char.ID)
			if err != nil || len(stored) != 1 || stored[0].Count != count || s.char.Money != gold || len(s.items) != 1 || s.items[0].Count != count {
				t.Fatalf("sale state = gold %d, inventory %v, stored %v, error %v", s.char.Money, s.items, stored, err)
			}
		})
	}
}
