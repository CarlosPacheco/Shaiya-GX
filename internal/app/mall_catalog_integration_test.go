//go:build integration

package app

import (
	"os"
	"sort"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

// All products, including hidden legacy codes, must deliver their exact native
// contents through encrypted buy/gift/claim packets using disposable accounts.
func TestEntireMallCatalogAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_ALL_MALL") != "1" {
		t.Skip("requires two disposable GM accounts and synchronized catalog")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	s1, w1, send1, _ := testworld.Enter(t)
	s2, w2, send2, _ := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	a, b := inputApp(), inputApp()
	a.state, a.world, a.arc = s1, w1, arc
	b.state, b.world, b.arc = s2, w2, arc
	defer a.Close()
	defer b.Close()
	if err = a.loadMallCatalog(); err != nil {
		t.Fatal(err)
	}
	wait := func(check func() bool) {
		t.Helper()
		until := time.Now().Add(10 * time.Second)
		for !check() {
			a.pumpWorld()
			b.pumpWorld()
			if a.state == nil || b.state == nil || time.Now().After(until) {
				t.Fatal("catalog probe disconnected or timed out")
			}
			time.Sleep(time.Millisecond)
		}
	}
	wait(func() bool { return s1.Self.IsGM && s2.Self.IsGM && s2.Mall.StoredReady })
	carried := func(items map[[2]uint8]proto.InventoryItem) map[uint32]uint32 {
		totals := map[uint32]uint32{}
		for k, it := range items {
			if k[0] > 0 {
				totals[uint32(it.Type)*1000+uint32(it.TypeID)] += uint32(it.Count)
			}
		}
		return totals
	}
	checkTotals := func(got, want map[uint32]uint32) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatal("wrong delivered identities", got, want)
		}
		for i, n := range want {
			if got[i] != n {
				t.Fatal("wrong delivered quantity", i, got[i], n)
			}
		}
	}
	clear := func(app *App, send func([]byte)) {
		t.Helper()
		send(proto.ChatPacket(proto.ChannelNormal, false, "/iclear"))
		wait(func() bool { return len(carried(app.state.Inventory)) == 0 })
	}
	for index, p := range a.mallCatalog {
		clear(a, send1)
		clear(b, send2)
		want := map[uint32]uint32{}
		for _, it := range p.Items {
			if it.ItemID > 0 {
				want[it.ItemID] += uint32(it.Count)
			}
		}
		serial := s1.Mall.PurchaseSerial
		send1(proto.MallBuyPacket(p.Code))
		wait(func() bool { return s1.Mall.PurchaseSerial > serial })
		if s1.Mall.LastPurchase.Result != 0 {
			t.Fatal("product cannot be bought", p.Code, s1.Mall.LastPurchase)
		}
		checkTotals(carried(s1.Inventory), want)
		for _, it := range s1.Mall.LastPurchase.Items {
			def := a.itemsItem(it.Type, it.TypeID)
			if def == nil || it.Count > max(def.Count, 1) {
				t.Fatal("client/server stack mismatch", p.Code, it)
			}
		}
		giftSerial := s1.Mall.GiftSerial
		send1(proto.MallGiftPacket(p.Code, s2.Slots[0].Name))
		wait(func() bool { return s1.Mall.GiftSerial > giftSerial && len(s2.Mall.Stored) > 0 })
		if s1.Mall.LastGift.Result != 0 {
			t.Fatal("product cannot be gifted", p.Code, s1.Mall.LastGift)
		}
		slots := []int{}
		giftTotals := map[uint32]uint32{}
		for slot, it := range s2.Mall.Stored {
			slots = append(slots, int(slot))
			giftTotals[uint32(it.Type)*1000+uint32(it.TypeID)] += uint32(it.Count)
		}
		checkTotals(giftTotals, want)
		sort.Ints(slots)
		for _, slot := range slots {
			claimSerial := s2.Mall.ClaimSerial
			send2(proto.MallClaimPacket(uint8(slot)))
			wait(func() bool { return s2.Mall.ClaimSerial > claimSerial && s2.Mall.PendingClaim == nil })
		}
		if len(s2.Mall.Stored) != 0 {
			t.Fatal("gift not fully claimed", p.Code, s2.Mall.Stored)
		}
		checkTotals(carried(s2.Inventory), want)
		if (index+1)%50 == 0 {
			t.Logf("%d/%d products: buy, gift, claims and exact quantities passed", index+1, len(a.mallCatalog))
		}
	}
	t.Logf("ALL %d native products purchased, gifted and claimed over encrypted connections", len(a.mallCatalog))
}
