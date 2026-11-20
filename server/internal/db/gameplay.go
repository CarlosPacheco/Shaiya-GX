package db

import (
	"context"
	"database/sql"
	"fmt"
)

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// CharacterChange is a plan made under the owning sessions' state locks.
// Nothing is published in memory or to clients until CommitGameplay succeeds.
type CharacterChange struct {
	Before, After           Character
	ItemsBefore, ItemsAfter []Item
}

type QuestCompletion struct {
	CharacterID uint32
	QuestID     uint16
	Repeatable  bool
}

// CommitGameplay makes both trade participants, or one quest's entire reward,
// durable together. Expected inventory and money protect against stale plans.
func CommitGameplay(changes []CharacterChange, quest *QuestCompletion) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error { return commitGameplayTx(c, tx, changes, quest) })
}

func commitGameplayTx(c context.Context, tx *sql.Tx, changes []CharacterChange, quest *QuestCompletion) error {

	seen := map[uint32]bool{}
	for _, change := range changes {
		id := change.Before.ID
		if id == 0 || change.After.ID != id || seen[id] {
			return fmt.Errorf("invalid gameplay character plan")
		}
		seen[id] = true
		var money int64
		if err := tx.QueryRowContext(c, "SELECT Money FROM PS_GameData_Chars WHERE CharID=? AND Del=0", id).Scan(&money); err != nil {
			return err
		}
		if uint32(money) != change.Before.Money {
			return fmt.Errorf("gameplay character money changed")
		}
		actual, err := characterItems(c, tx, id)
		if err != nil {
			return err
		}
		if !sameInventory(actual, change.ItemsBefore) {
			return fmt.Errorf("gameplay inventory changed")
		}
	}
	if quest != nil {
		if !seen[quest.CharacterID] {
			return fmt.Errorf("quest character not in gameplay plan")
		}
		query := "UPDATE PS_GameData_CharQuests SET Success=1,Finish=1 WHERE CharID=? AND QuestID=? AND COALESCE(Finish,0)=0 AND Del=0"
		if quest.Repeatable {
			query = "DELETE FROM PS_GameData_CharQuests WHERE CharID=? AND QuestID=? AND COALESCE(Finish,0)=0 AND Del=0"
		}
		result, err := tx.ExecContext(c, query, quest.CharacterID, int16(quest.QuestID))
		if err != nil {
			return err
		}
		if err := requireOne(result); err != nil {
			return fmt.Errorf("quest already completed or absent: %w", err)
		}
	}
	for _, change := range changes {
		if err := writeInventoryChange(c, tx, change); err != nil {
			return err
		}
		result, err := writeCharacter(c, tx, &change.After)
		if err != nil {
			return err
		}
		if err := requireOne(result); err != nil {
			return err
		}
	}
	return nil
}

func requireOne(result sql.Result) error {
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("expected one changed row, got %d", n)
	}
	return nil
}

func sameItemState(a, b Item) bool {
	return a.UID == b.UID && a.Bag == b.Bag && a.Slot == b.Slot && a.Type == b.Type && a.TypeID == b.TypeID && a.Count == b.Count && a.Quality == b.Quality && a.Gems == b.Gems && a.CraftName == b.CraftName
}

func sameInventory(a, b []Item) bool {
	if len(a) != len(b) {
		return false
	}
	byUID := map[int64]Item{}
	for _, it := range a {
		if _, exists := byUID[it.UID]; exists {
			return false
		}
		byUID[it.UID] = it
	}
	for _, it := range b {
		old, ok := byUID[it.UID]
		if !ok || !sameItemState(old, it) {
			return false
		}
		delete(byUID, it.UID)
	}
	return len(byUID) == 0
}

func writeInventoryChange(c context.Context, tx *sql.Tx, change CharacterChange) error {
	before, after := map[int64]Item{}, map[int64]Item{}
	positions := map[[2]uint8]bool{}
	for _, it := range change.ItemsBefore {
		before[it.UID] = it
	}
	for _, it := range change.ItemsAfter {
		position := [2]uint8{it.Bag, it.Slot}
		if it.UID == 0 || it.Count == 0 || positions[position] {
			return fmt.Errorf("invalid gameplay inventory plan")
		}
		if _, exists := after[it.UID]; exists {
			return fmt.Errorf("duplicate gameplay item uid")
		}
		positions[position] = true
		after[it.UID] = it
	}
	for uid := range before {
		if _, kept := after[uid]; !kept {
			result, err := tx.ExecContext(c, "DELETE FROM PS_GameData_CharItems WHERE CharID=? AND ItemUID=? AND Del=0", change.After.ID, uid)
			if err != nil {
				return err
			}
			if err := requireOne(result); err != nil {
				return err
			}
		}
	}
	// Keep unchanged rows, timestamps and all original metadata intact.
	for _, it := range change.ItemsAfter {
		old, exists := before[it.UID]
		if exists {
			if old.Type != it.Type || old.TypeID != it.TypeID {
				return fmt.Errorf("gameplay item identity changed")
			}
			if sameItemState(old, it) {
				continue
			}
			result, err := tx.ExecContext(c, `UPDATE PS_GameData_CharItems SET Bag=?,Slot=?,Quality=?,Gem1=?,Gem2=?,Gem3=?,Gem4=?,Gem5=?,Gem6=?,Craftname=?,Count=? WHERE CharID=? AND ItemUID=? AND Del=0`, it.Bag, it.Slot, it.Quality, it.Gems[0], it.Gems[1], it.Gems[2], it.Gems[3], it.Gems[4], it.Gems[5], it.CraftName, it.Count, change.After.ID, it.UID)
			if err != nil {
				return err
			}
			if err := requireOne(result); err != nil {
				return err
			}
		} else {
			_, err := tx.ExecContext(c, `INSERT INTO PS_GameData_CharItems(CharID,Bag,Slot,ItemID,Type,TypeID,ItemUID,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,MakeTime,MakeType) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, change.After.ID, it.Bag, it.Slot, it.ItemID(), it.Type, it.TypeID, it.UID, it.Quality, it.Gems[0], it.Gems[1], it.Gems[2], it.Gems[3], it.Gems[4], it.Gems[5], it.CraftName, it.Count, makeTime(&it), makeType(&it))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CommitItems applies an inventory-only plan without overwriting character
// money/stats. Used by item grants whose character state is owned elsewhere.
func CommitItems(charID uint32, before, after []Item) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return groupWrite(c, pool, func(c context.Context, tx *sql.Tx) error {
		actual, err := characterItems(c, tx, charID)
		if err != nil {
			return err
		}
		if !sameInventory(actual, before) {
			return fmt.Errorf("grant inventory changed")
		}
		return writeInventoryChange(c, tx, CharacterChange{Before: Character{ID: charID}, After: Character{ID: charID}, ItemsBefore: before, ItemsAfter: after})
	})
}

// CommitStatusItem saves the consumed item and remaining effects together.
func CommitStatusItem(change CharacterChange, buffs []AppliedBuff) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error {
		if err := commitGameplayTx(c, tx, []CharacterChange{change}, nil); err != nil {
			return err
		}
		return saveBuffsTx(c, tx, change.After.ID, buffs)
	})
}
