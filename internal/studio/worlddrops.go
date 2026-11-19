package studio

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
)

type WorldDropCatalog struct {
	DBPath string
	Rules  []worlddrops.Rule
}

func OpenWorldDrops(ctx context.Context, path string) (*WorldDropCatalog, error) {
	path, err := canonicalFile(path)
	if err != nil {
		return nil, err
	}
	db, err := openDB(ctx, path, "ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rules, err := worlddrops.Read(ctx, db)
	if err != nil {
		return nil, err
	}
	return &WorldDropCatalog{DBPath: path, Rules: rules}, nil
}

func SaveWorldDrops(ctx context.Context, root string, catalog *WorldDropCatalog, rules []worlddrops.Rule) (string, error) {
	if err := worlddrops.Validate(rules); err != nil {
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
	current, err := worlddrops.Read(ctx, tx)
	if err != nil {
		return "", err
	}
	if !slices.Equal(current, catalog.Rules) {
		return "", errors.New("world drop rules changed on disk; reopen the catalog before saving")
	}
	if err := safeTable(ctx, tx, worlddrops.Table, false); err != nil {
		return "", err
	}
	backup, err := backupDatabaseCatalog(ctx, root, catalog.DBPath, "world-drops", func(snapshot *sql.DB) error {
		old, err := worlddrops.Read(ctx, snapshot)
		if err != nil || !slices.Equal(old, current) {
			return errors.Join(errors.New("world drop backup failed verification"), err)
		}
		return nil
	})
	if err != nil {
		return backup, err
	}
	proposed := make(map[int64]worlddrops.Rule, len(rules))
	for _, rule := range rules {
		proposed[rule.Index] = rule
	}
	previous := make(map[int64]worlddrops.Rule, len(current))
	for _, rule := range current {
		previous[rule.Index] = rule
		if _, exists := proposed[rule.Index]; !exists {
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+worlddrops.Table+` WHERE "Index"=?`, rule.Index); err != nil {
				return backup, err
			}
		}
	}
	for _, rule := range rules {
		if previous[rule.Index] == rule {
			continue
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO `+worlddrops.Table+` ("Index",Description,Grade,MinLevel,MaxLevel,DropChance)
			VALUES (?,?,?,?,?,?) ON CONFLICT("Index") DO UPDATE SET Description=excluded.Description,
			Grade=excluded.Grade, MinLevel=excluded.MinLevel, MaxLevel=excluded.MaxLevel, DropChance=excluded.DropChance`,
			rule.Index, rule.Description, rule.Grade, rule.MinLevel, rule.MaxLevel, rule.DropChance)
		if err != nil {
			return backup, err
		}
	}
	written, err := worlddrops.Read(ctx, tx)
	if err != nil || !slices.Equal(written, rules) {
		return backup, errors.Join(errors.New("saved world drop rules failed verification"), err)
	}
	return backup, tx.Commit()
}
