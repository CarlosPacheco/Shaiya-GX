package db

import (
	"fmt"
	"testing"
)

func TestMallPurchaseRollbackAndGiftEquipmentQuantities(t *testing.T) {
	p := sqliteFixture(t)
	for uid := 994001; uid <= 994002; uid++ {
		if _, err := p.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,'local',0,0,0,0,0,'N',100)`, uid, fmt.Sprintf("mall_tx_%d", uid)); err != nil {
			t.Fatal(err)
		}
	}
	create := func(uid int32, name string) uint32 {
		id, err := CreateCharacter(Account{UserUID: uid, UserID: fmt.Sprintf("mall_tx_%d", uid)}, NewCharacter{Slot: 0, Name: name, MapID: 1, HP: 100, MP: 100, SP: 100})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	sender, receiver := create(994001, "MallTxSender"), create(994002, "MallTxReceiver")
	product := &Product{Code: "TX_TEST", BuyCost: 10, Items: []ProductItem{{Type: 1, TypeID: 1, Count: 4}}}
	// Fail the second insert: neither the first item nor the charge/log may remain.
	if _, err := p.Exec(`CREATE TRIGGER mall_fail BEFORE INSERT ON PS_GameData_CharItems WHEN NEW.CharID=` + fmt.Sprint(sender) + ` AND NEW.Slot=21 BEGIN SELECT RAISE(ABORT,'injected delivery failure'); END`); err != nil {
		t.Fatal(err)
	}
	items := []Item{{Bag: 1, Slot: 20, Type: 1, TypeID: 1, Count: 1}, {Bag: 1, Slot: 21, Type: 1, TypeID: 1, Count: 1}}
	if _, err := PurchaseMallItems(994001, sender, product, items); err == nil {
		t.Fatal("insert failure accepted")
	}
	var n int
	p.QueryRow(`SELECT COUNT(*) FROM PS_GameData_CharItems WHERE CharID=? AND Slot=20`, sender).Scan(&n)
	if balance, _ := Points(994001); balance != 100 || n != 0 {
		t.Fatal("partial delivery or charge survived rollback", balance, n)
	}
	p.QueryRow(`SELECT COUNT(*) FROM PS_GameData_PointLog WHERE ProductCode='TX_TEST'`).Scan(&n)
	if n != 0 {
		t.Fatal("failed purchase retained a point log", n)
	}
	p.Exec(`DROP TRIGGER mall_fail`)
	if result, _, balance, err := PurchaseGift("MallTxReceiver", 994001, sender, product); err != nil || result != GiftOK || balance != 90 {
		t.Fatal(result, balance, err)
	}
	gifts, err := GiftBox(994002)
	if err != nil || len(gifts) != 4 {
		t.Fatal("equipment bundle not split into individual gifts", gifts, err)
	}
	for _, g := range gifts {
		if g.Count != 1 {
			t.Fatal("stacked equipment gift", g)
		}
	}
	// Legacy unsplit equipment gifts retain the remainder after each claim.
	p.Exec(`INSERT INTO PS_GameData_UserStoredPointItems(UserUID,Slot,ItemID,ItemCount) VALUES(994002,50,1001,4)`)
	it := &Item{Bag: 1, Slot: 22, Type: 1, TypeID: 1, Count: 1}
	if ok, err := ClaimGift(994002, receiver, 50, it); err != nil || !ok {
		t.Fatal(ok, err)
	}
	p.QueryRow(`SELECT ItemCount FROM PS_GameData_UserStoredPointItems WHERE UserUID=994002 AND Slot=50`).Scan(&n)
	if n != 3 {
		t.Fatal("claim lost legacy bundle quantity", n)
	}
	// A failed charge must roll back all inserted gifts as well.
	p.Exec(`UPDATE PS_UserData_Users_Master SET Point=0 WHERE UserUID=994001`)
	if _, _, _, err := PurchaseGift("MallTxReceiver", 994001, sender, product); err != ErrNotEnoughPoints {
		t.Fatal("missing charge rejected incorrectly", err)
	}
	after, _ := GiftBox(994002)
	if len(after) != len(gifts)+1 {
		t.Fatal("unpaid gifts survived transaction", after)
	}
	p.Exec(`UPDATE PS_UserData_Users_Master SET Point=100 WHERE UserUID=994001`)
	for slot := 0; slot < GiftBoxSlots-2; slot++ {
		if _, err := p.Exec(`INSERT INTO PS_GameData_UserStoredPointItems(UserUID,Slot,ItemID,ItemCount) SELECT 994002,?,1001,1 WHERE NOT EXISTS(SELECT 1 FROM PS_GameData_UserStoredPointItems WHERE UserUID=994002 AND Slot=?)`, slot, slot); err != nil {
			t.Fatal(err)
		}
	}
	if result, _, _, err := PurchaseGift("MallTxReceiver", 994001, sender, product); err != nil || result != GiftNotEnoughSlots {
		t.Fatal("split bundle capacity", result, err)
	}
	if balance, _ := Points(994001); balance != 100 {
		t.Fatal("full gift box charged sender", balance)
	}
	after, _ = GiftBox(994002)
	if len(after) != GiftBoxSlots-2 {
		t.Fatal("partial bundle survived gift-box capacity rejection", len(after))
	}

}
