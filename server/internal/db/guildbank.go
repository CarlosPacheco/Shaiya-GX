package db

import (
	"context"
	"database/sql"
	"fmt"
)

func CommitGuildBank(guildID uint32, change CharacterChange, before, after []Item) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error {
		actual, err := guildStoredItems(c, tx, guildID)
		if err != nil {
			return err
		}
		for i := range actual {
			actual[i].Bag = 255
		}
		if !sameInventory(actual, before) {
			return fmt.Errorf("guild bank inventory changed")
		}
		if err := commitGameplayTx(c, tx, []CharacterChange{change}, nil); err != nil {
			return err
		}
		return writeGuildBankChange(c, tx, guildID, before, after)
	})
}
func writeGuildBankChange(c context.Context, tx *sql.Tx, guildID uint32, before, after []Item) error {
	old := map[int64]Item{}
	next := map[int64]Item{}
	positions := map[uint8]bool{}
	for _, it := range before {
		if _, ok := old[it.UID]; ok {
			return fmt.Errorf("duplicate guild item identity")
		}
		old[it.UID] = it
	}
	for _, it := range after {
		if it.UID == 0 || it.Count == 0 || it.Slot >= 240 || positions[it.Slot] {
			return fmt.Errorf("invalid guild bank inventory")
		}
		if _, ok := next[it.UID]; ok {
			return fmt.Errorf("duplicate guild item identity")
		}
		positions[it.Slot] = true
		next[it.UID] = it
	}
	for id := range old {
		if _, kept := next[id]; !kept {
			result, err := tx.ExecContext(c, `DELETE FROM PS_GameData_GuildStoredItems WHERE GuildID=? AND ItemUID=? AND Del=0`, guildID, id)
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
				return fmt.Errorf("guild item identity changed")
			}
			if sameItemState(previous, it) {
				continue
			}
			result, err := tx.ExecContext(c, `UPDATE PS_GameData_GuildStoredItems SET Slot=?,Quality=?,Gem1=?,Gem2=?,Gem3=?,Gem4=?,Gem5=?,Gem6=?,Craftname=?,Count=? WHERE GuildID=? AND ItemUID=? AND Del=0`, it.Slot, it.Quality, it.Gems[0], it.Gems[1], it.Gems[2], it.Gems[3], it.Gems[4], it.Gems[5], it.CraftName, it.Count, guildID, it.UID)
			if err != nil {
				return err
			}
			if err = requireOne(result); err != nil {
				return err
			}
		} else {
			if err := insertGuildStoredItem(c, tx, guildID, &it); err != nil {
				return err
			}
		}
	}
	return nil
}
