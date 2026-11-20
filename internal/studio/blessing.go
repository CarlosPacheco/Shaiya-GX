package studio

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/blessingrules"
)

type BlessingCatalog struct {
	DBPath string
	Rules  blessingrules.Catalog
	File   *ServerFile
}

func OpenBlessing(ctx context.Context, path string) (*BlessingCatalog, error) {
	if filepath.Base(path) == catalogfiles.KillStatus {
		file, value, err := openServerCatalog(path, catalogfiles.ParseKillStatus)
		if err != nil {
			return nil, err
		}
		return &BlessingCatalog{File: file, Rules: value}, nil
	}
	path, err := canonicalFile(path)
	if err != nil {
		return nil, err
	}
	db, err := openDB(ctx, path, "ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rules, err := blessingrules.Read(ctx, db)
	if err != nil {
		return nil, err
	}
	return &BlessingCatalog{DBPath: path, Rules: rules}, nil
}

func SaveBlessing(ctx context.Context, root string, catalog *BlessingCatalog, rules blessingrules.Catalog) (string, error) {
	if catalog.File != nil {
		next := catalogfiles.EncodeKillStatus(rules)
		return saveServerCatalog(root, catalog.File, next, catalogfiles.ParseKillStatus, func(v blessingrules.Catalog) bool { return slices.Equal(v, rules) })
	}
	if err := rules.Validate(); err != nil {
		return "", err
	}
	if len(rules) != len(catalog.Rules) {
		return "", errors.New("keep the existing blessing tiers")
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
	current, err := blessingrules.Read(ctx, tx)
	if err != nil {
		return "", err
	}
	if !slices.Equal(current, catalog.Rules) {
		return "", errors.New("blessing tiers changed on disk; reopen the catalog before saving")
	}
	if err := safeTable(ctx, tx, blessingrules.Table, false); err != nil {
		return "", err
	}
	backup, err := backupDatabaseCatalog(ctx, root, catalog.DBPath, "blessing", func(snapshot *sql.DB) error {
		old, err := blessingrules.Read(ctx, snapshot)
		if err != nil || !slices.Equal(old, current) {
			return errors.Join(errors.New("blessing backup failed verification"), err)
		}
		return nil
	})
	if err != nil {
		return backup, err
	}
	columns := []string{"RequiredKills=?"}
	for _, effect := range blessingrules.Effects {
		columns = append(columns, effect.Column+"=?")
	}
	for i, tier := range rules {
		if tier == current[i] {
			continue
		}
		args := []any{tier.RequiredKills}
		for _, value := range tier.Bonuses {
			args = append(args, value)
		}
		args = append(args, tier.Number)
		if _, err := tx.ExecContext(ctx, "UPDATE "+blessingrules.Table+" SET "+strings.Join(columns, ",")+" WHERE Tier=?", args...); err != nil {
			return backup, err
		}
	}
	written, err := blessingrules.Read(ctx, tx)
	if err != nil || !slices.Equal(written, rules) {
		return backup, errors.Join(errors.New("saved blessing tiers failed verification"), err)
	}
	return backup, tx.Commit()
}
