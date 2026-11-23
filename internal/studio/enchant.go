package studio

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
)

type EnchantCatalog struct {
	DBPath string
	Rules  enchantrules.Catalog
	File   *ServerFile
}

func OpenEnchant(ctx context.Context, path string) (*EnchantCatalog, error) {
	if filepath.Base(path) == catalogfiles.Enchant {
		file, value, err := openServerCatalog(path, catalogfiles.ParseEnchant)
		if err != nil {
			return nil, err
		}
		return &EnchantCatalog{File: file, Rules: value}, nil
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
	levels, err := enchantrules.Read(ctx, db)
	if err != nil {
		return nil, err
	}
	return &EnchantCatalog{DBPath: path, Rules: levels}, nil
}

func SaveEnchant(ctx context.Context, root string, catalog *EnchantCatalog, levels enchantrules.Catalog) (string, error) {
	if catalog.File != nil {
		next, err := catalogfiles.UpdateEnchant(levels, catalog.File.original)
		if err != nil {
			return "", err
		}
		return saveServerCatalog(root, catalog.File, next, catalogfiles.ParseEnchant, func(v enchantrules.Catalog) bool {
			return slices.Equal(v.Attack, levels.Attack) && slices.Equal(v.Defense, levels.Defense)
		})
	}
	if _, _, err := levels.Caps(); err != nil {
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
	current, err := enchantrules.Read(ctx, tx)
	if err != nil {
		return "", err
	}
	if !current.Equal(catalog.Rules) {
		return "", errors.New("enchant definitions changed on disk; reopen the catalog before saving")
	}
	for _, table := range []string{enchantrules.AttackTable, enchantrules.DefenseTable} {
		if err := safeTable(ctx, tx, table, false); err != nil {
			return "", err
		}
	}
	backup, err := backupDatabaseCatalog(ctx, root, catalog.DBPath, "enchant", func(snapshot *sql.DB) error {
		levels, err := enchantrules.Read(ctx, snapshot)
		if err != nil || !levels.Equal(catalog.Rules) {
			return errors.Join(errors.New("enchant backup failed verification"), err)
		}
		return nil
	})
	if err != nil {
		return backup, err
	}
	if err := enchantrules.Replace(ctx, tx, levels); err != nil {
		return backup, err
	}
	written, err := enchantrules.Read(ctx, tx)
	if err != nil || !written.Equal(levels) {
		return backup, errors.Join(errors.New("saved rules failed verification"), err)
	}
	return backup, tx.Commit()
}

func backupDatabaseCatalog(ctx context.Context, root, database, label string, verify func(*sql.DB) error) (string, error) {
	parent := filepath.Join(root, "backups")
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", err
	}
	if err := syncDirectory(root); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(parent, time.Now().UTC().Format("20060102T150405Z")+"-"+label+"-")
	if err != nil {
		return "", err
	}
	if err := syncDirectory(parent); err != nil {
		return dir, err
	}
	source, err := openDB(ctx, database, "ro")
	if err != nil {
		return dir, err
	}
	defer source.Close()
	path := filepath.Join(dir, "rapture.sqlite")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return dir, err
	}
	if err := file.Close(); err != nil {
		return dir, err
	}
	if err := backupDB(ctx, source, path); err != nil {
		return dir, err
	}
	snapshot, err := openDB(ctx, path, "ro")
	if err != nil {
		return dir, err
	}
	defer snapshot.Close()
	if err := integrity(ctx, snapshot); err != nil {
		return dir, err
	}
	if err := verify(snapshot); err != nil {
		return dir, err
	}
	if err := syncFile(path); err != nil {
		return dir, err
	}
	return dir, syncDirectory(dir)
}
