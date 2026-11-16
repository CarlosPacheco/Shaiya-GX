package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
)

// CommitWarehouse guards the whole account inventory under the same write lock
// as character items and withdrawal fees, before any success is published.
func CommitWarehouse(userUID uint32, change CharacterChange, before, after []Item) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error {
		if change.Before.AccountID != userUID {
			return fmt.Errorf("warehouse character ownership mismatch")
		}
		actual, err := storedItems(c, tx, userUID)
		if err != nil {
			return err
		}
		if !sameInventory(actual, before) {
			return fmt.Errorf("warehouse inventory changed")
		}
		if err = commitGameplayTx(c, tx, []CharacterChange{change}, nil); err != nil {
			return err
		}
		return writeWarehouseChange(c, tx, userUID, before, after)
	})
}
func writeWarehouseChange(c context.Context, tx *sql.Tx, userUID uint32, before, after []Item) error {
	old := map[int64]Item{}
	next := map[int64]Item{}
	positions := map[uint8]bool{}
	for _, it := range before {
		if _, ok := old[it.UID]; ok {
			return fmt.Errorf("duplicate stored item identity")
		}
		old[it.UID] = it
	}
	for _, it := range after {
		if it.UID == 0 || it.Count == 0 || it.Slot >= 240 || positions[it.Slot] {
			return fmt.Errorf("invalid warehouse inventory")
		}
		if _, ok := next[it.UID]; ok {
			return fmt.Errorf("duplicate stored item identity")
		}
		positions[it.Slot] = true
		next[it.UID] = it
	}
	for id := range old {
		if _, kept := next[id]; !kept {
			result, err := tx.ExecContext(c, `DELETE FROM PS_GameData_UserStoredItems WHERE ServerID=? AND UserUID=? AND ItemUID=?`, config.ServerID, userUID, id)
			if err != nil {
				return err
			}
			if err = requireOne(result); err != nil {
				return err
			}
		}
	}
	for _, it := range after {
		if previous, ok := old[it.UID]; ok {
			if previous.Type != it.Type || previous.TypeID != it.TypeID {
				return fmt.Errorf("stored item identity changed")
			}
			if sameItemState(previous, it) {
				continue
			}
			result, err := tx.ExecContext(c, `UPDATE PS_GameData_UserStoredItems SET Slot=?,Quality=?,Gem1=?,Gem2=?,Gem3=?,Gem4=?,Gem5=?,Gem6=?,Craftname=?,Count=? WHERE ServerID=? AND UserUID=? AND ItemUID=?`, it.Slot, it.Quality, it.Gems[0], it.Gems[1], it.Gems[2], it.Gems[3], it.Gems[4], it.Gems[5], it.CraftName, it.Count, config.ServerID, userUID, it.UID)
			if err != nil {
				return err
			}
			if err = requireOne(result); err != nil {
				return err
			}
		} else {
			_, err := tx.ExecContext(c, `INSERT INTO PS_GameData_UserStoredItems(ServerID,UserUID,Slot,ItemID,Type,TypeID,ItemUID,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,MakeTime,MakeType) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, config.ServerID, userUID, it.Slot, it.ItemID(), it.Type, it.TypeID, it.UID, it.Quality, it.Gems[0], it.Gems[1], it.Gems[2], it.Gems[3], it.Gems[4], it.Gems[5], it.CraftName, it.Count, makeTime(&it), makeType(&it))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
