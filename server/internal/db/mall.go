package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Account points, gifts and the game log share one SQLite file. Guarded balance
// updates and gift claims are transactional across the former logical databases.

const GiftBoxSlots = 240

// Points reads the account's item-mall balance.
func Points(userUID uint32) (uint32, error) {
	pool, err := ReadPool()
	if err != nil {
		return 0, err
	}
	c, cancel := ctx()
	defer cancel()
	var point sql.NullInt32
	err = pool.QueryRowContext(c, `SELECT Point FROM PS_UserData_Users_Master WHERE UserUID = ?1`, int32(userUID)).Scan(&point)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return uint32(max(point.Int32, 0)), err
}

// ErrNotEnoughPoints means the balance does not cover the cost.
var ErrNotEnoughPoints = errors.New("not enough points")

// SpendPoints guards the balance and records the original NCash PointLog in
// the same transaction. useType: 1 buy, 2 gift.
func SpendPoints(userUID, charID, cost uint32, productCode, targetName string, useType uint8) (uint32, error) {
	p, e := Pool(UserData)
	if e != nil {
		return 0, e
	}
	c, cancel := ctx()
	defer cancel()
	var remaining uint32
	e = withTx(c, p, func(tx *sql.Tx) error {
		var err error
		remaining, err = spendPointsTx(c, tx, userUID, charID, cost, productCode, targetName, useType)
		return err
	})
	return remaining, e
}

// spendPointsTx keeps charging and delivery in the same transaction.
func spendPointsTx(c context.Context, tx *sql.Tx, userUID, charID, cost uint32, productCode, targetName string, useType uint8) (uint32, error) {
	var remaining uint32
	err := tx.QueryRowContext(c, `UPDATE PS_UserData_Users_Master SET Point=Point-?2 WHERE UserUID=?1 AND Point>=?2 RETURNING Point`, int32(userUID), int64(cost)).Scan(&remaining)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotEnoughPoints
	}
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(c, `INSERT INTO PS_GameData_PointLog(UseType,UserUID,CharID,UsePoint,ProductCode,TargetName,UseDate,RemainPoint) VALUES(?1,?2,?3,?4,?5,?6,CURRENT_TIMESTAMP,?7)`, useType, int32(userUID), int32(charID), int64(cost), productCode, targetName, int64(remaining))
	return remaining, err
}

// PurchaseMallItems atomically charges, logs and inserts a validated delivery.
// The caller publishes packets and updates session memory only after commit.
func PurchaseMallItems(userUID, charID uint32, product *Product, items []Item) (uint32, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return 0, err
	}
	c, cancel := ctx()
	defer cancel()
	var remaining uint32
	err = groupWrite(c, pool, func(c context.Context, tx *sql.Tx) error {
		var err error
		remaining, err = spendPointsTx(c, tx, userUID, charID, product.BuyCost, product.Code, "", 1)
		if err != nil {
			return err
		}
		for i := range items {
			it := &items[i]
			if it.UID == 0 {
				it.UID = NewItemUID()
			}
			if it.MakeTime.IsZero() {
				it.MakeTime = time.Now()
			}
			_, err = tx.ExecContext(c, `INSERT INTO PS_GameData_CharItems
			(CharID,Bag,Slot,ItemID,Type,TypeID,ItemUID,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype)
			VALUES(?,?,?,?,?,?,?,?,0,0,0,0,0,0,'',?,?, 'B')`, int32(charID), it.Bag, it.Slot, it.ItemID(), it.Type, it.TypeID, it.UID, it.Quality, it.Count, makeTime(it))
			if err != nil {
				return err
			}
		}
		return nil
	})
	return remaining, err
}

// GiftEntry is one gift-box slot.
type GiftEntry struct {
	Slot         uint8
	Type, TypeID uint8
	Count        uint16
}

// GiftBox runs usp_Read_User_StoredPointItems.
func GiftBox(userUID uint32) ([]GiftEntry, error) {
	pool, err := ReadPool()
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT Slot,ItemID,ItemCount FROM PS_GameData_UserStoredPointItems WHERE UserUID=@useruid`, sql.Named("useruid", int32(userUID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_User_StoredPointItems: %w", err)
	}
	defer rows.Close()
	var gifts []GiftEntry
	for rows.Next() {
		var slot uint8
		var count uint16
		var itemID int32
		if err := rows.Scan(&slot, &itemID, &count); err != nil {
			return nil, err
		}
		if slot < GiftBoxSlots && itemID > 0 && count > 0 {
			gifts = append(gifts, GiftEntry{Slot: slot, Type: uint8(min(itemID/1000, 255)),
				TypeID: uint8(min(itemID%1000, 255)), Count: count})
		}
	}
	return gifts, rows.Err()
}

// GiftResult codes match the original usp_Save_User_GiftPointItems_NCash
// and the client's gift result byte.
const (
	GiftOK             = 0
	GiftNoSuchChar     = 1
	GiftSameAccount    = 2
	GiftBoxFull        = 3
	GiftNotEnoughSlots = 4
	GiftInsertError    = 5
)

// StoreGift puts every item of product into the gift box of the account
// owning targetName, in free slots, atomically.
func StoreGift(targetName string, senderUID uint32, product *Product) (uint8, uint32, error) {
	result, target, _, err := storeGift(targetName, senderUID, 0, product, false)
	return result, target, err
}

// PurchaseGift charges the sender and stores every gift in one transaction.
func PurchaseGift(targetName string, senderUID, charID uint32, product *Product) (uint8, uint32, uint32, error) {
	return storeGift(targetName, senderUID, charID, product, true)
}

func storeGift(targetName string, senderUID, charID uint32, product *Product, charge bool) (result uint8, targetUID, points uint32, err error) {
	pool, err := Pool(GameData)
	if err != nil {
		return GiftInsertError, 0, 0, err
	}
	c, cancel := ctx()
	defer cancel()
	defs := MustDefs()
	err = withTx(c, pool, func(tx *sql.Tx) error {
		var uid sql.NullInt32
		err := tx.QueryRowContext(c, `SELECT UserUID FROM PS_GameData_Chars WHERE CharName=? AND Del=0 LIMIT 1`, targetName).Scan(&uid)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if !uid.Valid || uid.Int32 == 0 {
			result = GiftNoSuchChar
			return errRollback
		}
		targetUID = uint32(uid.Int32)
		if targetUID == senderUID {
			result = GiftSameAccount
			return errRollback
		}
		used := map[int]bool{}
		rows, err := tx.QueryContext(c, `SELECT Slot FROM PS_GameData_UserStoredPointItems WHERE UserUID=?`, targetUID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var slot int
			if err = rows.Scan(&slot); err != nil {
				rows.Close()
				return err
			}
			used[slot] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(used) >= GiftBoxSlots {
			result = GiftBoxFull
			return errRollback
		}
		next := 0
		for _, item := range product.Items {
			def := defs.Item(item.Type, item.TypeID)
			if def == nil || item.Count == 0 {
				return fmt.Errorf("invalid gift item %d/%d", item.Type, item.TypeID)
			}
			// Equipment bundles occupy one gift slot per individual piece.
			for remaining := item.Count; remaining > 0; {
				for next < GiftBoxSlots && used[next] {
					next++
				}
				if next >= GiftBoxSlots {
					result = GiftNotEnoughSlots
					return errRollback
				}
				n := min(remaining, def.MaxStack())
				_, err := tx.ExecContext(c, `INSERT INTO PS_GameData_UserStoredPointItems(UserUID,Slot,ItemID,ItemCount,BuyDate) VALUES(?,?,?,?,CURRENT_TIMESTAMP)`, targetUID, next, int32(item.Type)*1000+int32(item.TypeID), n)
				if err != nil {
					return err
				}
				used[next] = true
				remaining -= n
			}
		}
		if len(product.Items) == 0 {
			return fmt.Errorf("empty gift product")
		}
		if charge {
			points, err = spendPointsTx(c, tx, senderUID, charID, product.BuyCost, product.Code, targetName, 2)
			return err
		}
		return nil
	})
	if errors.Is(err, errRollback) {
		return result, targetUID, 0, nil
	}
	if err != nil {
		return GiftInsertError, targetUID, 0, err
	}
	return GiftOK, targetUID, points, nil
}

// ClaimGift moves gift-box slot into the character's inventory as it, in
// one transaction (the effect of usp_Save_User_StoredPointItems). It
// returns false if the slot was already claimed.
func ClaimGift(userUID, charID uint32, giftSlot uint8, it *Item) (bool, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return false, err
	}
	c, cancel := ctx()
	defer cancel()
	if it.UID == 0 {
		it.UID = NewItemUID()
	}
	if it.MakeTime.IsZero() {
		it.MakeTime = time.Now()
	}
	claimed := false
	err = withTx(c, pool, func(tx *sql.Tx) error {
		if it.Count == 0 {
			return errRollback
		}
		var available uint16
		var identity int32
		err := tx.QueryRowContext(c, `SELECT ItemID,ItemCount FROM PS_GameData_UserStoredPointItems WHERE UserUID=? AND Slot=?`, userUID, giftSlot).Scan(&identity, &available)
		if errors.Is(err, sql.ErrNoRows) {
			return errRollback
		}
		if err != nil {
			return err
		}
		if identity != it.ItemID() || available < it.Count {
			return errRollback
		}
		if available == it.Count {
			_, err = tx.ExecContext(c, `DELETE FROM PS_GameData_UserStoredPointItems WHERE UserUID=? AND Slot=?`, userUID, giftSlot)
		} else {
			_, err = tx.ExecContext(c, `UPDATE PS_GameData_UserStoredPointItems SET ItemCount=ItemCount-? WHERE UserUID=? AND Slot=?`, it.Count, userUID, giftSlot)
		}
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(c, `INSERT INTO PS_GameData_CharItems (CharID, Bag, Slot, ItemID, Type, TypeID, ItemUID,
			Quality, Gem1, Gem2, Gem3, Gem4, Gem5, Gem6, Craftname, Count, Maketime, Maketype)
			VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, 0, 0, 0, 0, 0, 0, '', ?9, ?10, 'B')`,
			int32(charID), it.Bag, it.Slot, it.ItemID(), it.Type, it.TypeID, it.UID,
			int16(min(it.Quality, 4999)), it.Count, makeTime(it)); err != nil {
			return err
		}
		claimed = true
		return nil
	})
	if errors.Is(err, errRollback) {
		return false, nil
	}
	return claimed, err
}

var errRollback = errors.New("rollback")
