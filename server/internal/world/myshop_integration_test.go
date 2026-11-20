//go:build integration

package world

import (
	"fmt"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestMyShopPurchaseRollsBackAndRetries(t *testing.T) {
	seller, buyer := auditTradePair(t)
	for _, s := range []*Session{seller, buyer} {
		s.char.MapID = 35
		s.player.SetMap(s.char.MapID)
		s.syncPlayer()
		if err := db.SaveCharacter(s.char); err != nil {
			t.Fatal(err)
		}
	}
	item := db.Item{Bag: 1, Slot: 23, Type: 25, TypeID: 43, Count: 4}
	if err := db.AddCharacterItem(seller.char.ID, &item); err != nil {
		t.Fatal(err)
	}
	seller.items = append(seller.items, item)
	seller.handleMyShop(opMyShopBegin, proto.NewWriter(proto.MyShopBegin).Bytes())
	seller.handleMyShop(opMyShopAdd, proto.MyShopAddPacket(0, item.Bag, item.Slot, 70))
	seller.openMyShop(proto.MyShopOpenPacket("Trade goods"))
	if seller.shop.status != myShopStatusSelling {
		t.Fatal("fixture failed to open the market")
	}
	for range myShopBuyersMax {
		seller.shop.buyers = append(seller.shop.buyers, &ConnectedPlayer{}) // old browsers left
	}
	buyer.enterMyShop(seller.char.ID)
	if buyer.player.currentViewedShop() != seller.player || len(seller.shop.buyers) != 1 {
		t.Fatal("inactive buyers exhausted shop capacity")
	}
	pool, _ := db.Pool(db.GameData)
	trigger := fmt.Sprintf("CREATE TRIGGER market_write_failure BEFORE UPDATE OF Money ON PS_GameData_Chars WHEN OLD.CharID=%d BEGIN SELECT RAISE(ABORT,'market write failed'); END", buyer.char.ID)
	if _, err := pool.Exec(trigger); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec("DROP TRIGGER IF EXISTS market_write_failure") })
	check := func(sellerMoney, buyerMoney uint32, sellerStock, buyerStock uint16) {
		t.Helper()
		for _, row := range []struct {
			s     *Session
			money uint32
			count uint16
		}{{seller, sellerMoney, sellerStock}, {buyer, buyerMoney, buyerStock}} {
			c, err := db.CharacterByID(row.s.char.ID)
			if err != nil || c.Money != row.money || row.s.char.Money != row.money {
				t.Fatalf("money persisted=%d memory=%d expected=%d error=%v", c.Money, row.s.char.Money, row.money, err)
			}
			items, err := db.CharacterItems(row.s.char.ID)
			if err != nil {
				t.Fatal(err)
			}
			count := func(items []db.Item) uint16 {
				var units uint16
				for _, it := range items {
					if it.Type == item.Type && it.TypeID == item.TypeID {
						units += it.Count
					}
				}
				return units
			}
			if count(items) != row.count || count(row.s.items) != row.count {
				t.Fatalf("stock persisted=%d memory=%d expected=%d", count(items), count(row.s.items), row.count)
			}
		}
	}
	buyer.buyFromMyShop(0, 2)
	check(1000, 1000, 4, 0)
	if seller.shop.slots[0].item.Count != 4 {
		t.Fatal("failed sale changed public stock")
	}
	if _, err := pool.Exec("DROP TRIGGER market_write_failure"); err != nil {
		t.Fatal(err)
	}
	buyer.buyFromMyShop(0, 2)
	check(1140, 860, 2, 2)
	if seller.shop.slots[0].item.Count != 2 {
		t.Fatal("successful partial sale failed to update public stock")
	}
	buyer.char.PosX += proto.MyShopReach + 1
	buyer.buyFromMyShop(0, 1)
	check(1140, 860, 2, 2)
}
