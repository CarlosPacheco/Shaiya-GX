package studio

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/itemboxes"
)

type ItemBoxCatalog struct {
	DBPath string
	Rules  []itemboxes.Box
}

func OpenItemBoxes(ctx context.Context, path string) (*ItemBoxCatalog, error) {
	path, err := canonicalFile(path)
	if err != nil {
		return nil, err
	}
	db, err := openDB(ctx, path, "ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rules, err := itemboxes.Read(ctx, db)
	if err != nil {
		return nil, err
	}
	return &ItemBoxCatalog{DBPath: path, Rules: rules}, nil
}

func SaveItemBoxes(ctx context.Context, root string, catalog *ItemBoxCatalog, rules []itemboxes.Box) (string, error) {
	if err := itemboxes.Validate(rules); err != nil {
		return "", err
	}
	lock, err := lockRoot(root)
	if err != nil {
		return "", err
	}
	defer lock.Close()
	if pending, err := Pending(root); err != nil {
		return "", err
	} else if pending {
		return "", errors.New("recover the previous save before editing")
	}

	db, err := openDB(ctx, catalog.DBPath, "rw")
	if err != nil {
		return "", err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	current, err := itemboxes.Read(ctx, tx)
	if err != nil {
		return "", err
	}
	if !itemboxes.Equal(current, catalog.Rules) {
		return "", errors.New("item boxes changed on disk; reopen the catalog before saving")
	}
	if err := safeTable(ctx, tx, itemboxes.Table, false, itemboxes.RewardsTable); err != nil {
		return "", err
	}
	if err := safeTable(ctx, tx, itemboxes.RewardsTable, false); err != nil {
		return "", err
	}
	if err := validateBoxItems(ctx, tx, rules); err != nil {
		return "", err
	}
	backup, err := backupDatabaseCatalog(ctx, root, catalog.DBPath, "item-boxes", func(snapshot *sql.DB) error {
		old, err := itemboxes.Read(ctx, snapshot)
		if err != nil || !itemboxes.Equal(old, current) {
			return errors.Join(errors.New("item box backup failed verification"), err)
		}
		return nil
	})
	if err != nil {
		return backup, err
	}
	previous := make(map[int16]itemboxes.Box, len(current))
	for _, box := range current {
		previous[box.Index] = box
	}
	for _, box := range rules {
		old, exists := previous[box.Index]
		delete(previous, box.Index)
		if exists && old.Equal(box) {
			continue
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO `+itemboxes.Table+` ("Index",Description,BoxType,ItemType) VALUES (?,?,?,?)
			ON CONFLICT("Index") DO UPDATE SET Description=excluded.Description,BoxType=excluded.BoxType,ItemType=excluded.ItemType`,
			box.Index, box.Description, box.BoxType, box.ItemType)
		if err != nil {
			return backup, err
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM "+itemboxes.RewardsTable+" WHERE BoxIndex=?", box.Index); err != nil {
			return backup, err
		}
		for i, r := range box.Rewards {
			if _, err = tx.ExecContext(ctx, "INSERT INTO "+itemboxes.RewardsTable+" (BoxIndex,Position,Item,Chance) VALUES (?,?,?,?)", box.Index, i+1, r.Item, r.Chance); err != nil {
				return backup, err
			}
		}
	}
	for id := range previous {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+itemboxes.RewardsTable+" WHERE BoxIndex=?", id); err != nil {
			return backup, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+itemboxes.Table+` WHERE "Index"=?`, id); err != nil {
			return backup, err
		}
	}
	written, err := itemboxes.Read(ctx, tx)
	if err != nil || !itemboxes.Equal(written, rules) {
		return backup, errors.Join(errors.New("saved item boxes failed verification"), err)
	}
	return backup, tx.Commit()
}

func validateBoxItems(ctx context.Context, tx *sql.Tx, boxes []itemboxes.Box) error {
	type target struct {
		itemType uint8
		item     int32
	}
	valid := make(map[target]bool)
	for _, box := range boxes {
		for _, reward := range box.Rewards {
			if reward.Chance == 0 {
				continue
			}
			key := target{box.ItemType, reward.Item}
			if valid[key] {
				continue
			}
			var candidates int
			query := "SELECT COUNT(*) FROM PS_GameDefs_Items WHERE Type<>0 AND TypeID<>0 AND ItemID=?"
			if box.ItemType == itemboxes.Grade {
				query = "SELECT COUNT(*) FROM PS_GameDefs_Items WHERE Type NOT IN (0,27,28,29) AND TypeID<>0 AND Grade=?"
			}
			if err := tx.QueryRowContext(ctx, query, reward.Item).Scan(&candidates); err != nil {
				return err
			}
			if candidates == 0 || (box.ItemType == itemboxes.Direct && candidates != 1) {
				return fmt.Errorf("box %d reward %d has no matching grade or unique ItemID", box.Index, reward.Item)
			}
			valid[key] = true
		}
	}
	return nil
}
