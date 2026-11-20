package studio

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"github.com/Spelunkern/Shaiya-GX/internal/teleportrunes"
)

type TeleportRuneCatalog struct {
	DBPath string
	Rules  []teleportrunes.Rune
	File   *ServerFile
}

func OpenTeleportRunes(ctx context.Context, path string) (*TeleportRuneCatalog, error) {
	if strings.EqualFold(filepath.Base(path), catalogfiles.TeleportRunes) {
		file, value, err := openServerCatalog(catalogfiles.TeleportRunesPath(filepath.Dir(path)), catalogfiles.ParseTeleportRunes)
		if err != nil {
			return nil, err
		}
		return &TeleportRuneCatalog{File: file, Rules: value}, nil
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
	rules, err := teleportrunes.Read(ctx, db)
	if err != nil {
		return nil, err
	}
	return &TeleportRuneCatalog{DBPath: path, Rules: rules}, nil
}

func SaveTeleportRunes(ctx context.Context, root string, catalog *TeleportRuneCatalog, rules []teleportrunes.Rune) (string, error) {
	if catalog.File != nil {
		next := catalogfiles.EncodeTeleportRunes(rules)
		return saveServerCatalog(root, catalog.File, next, catalogfiles.ParseTeleportRunes, func(v []teleportrunes.Rune) bool { return slices.Equal(v, rules) })
	}
	if err := teleportrunes.Validate(rules); err != nil {
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
	current, err := teleportrunes.Read(ctx, tx)
	if err != nil {
		return "", err
	}
	if !slices.Equal(current, catalog.Rules) {
		return "", errors.New("teleport runes changed on disk; reopen the catalog before saving")
	}
	if err := safeTable(ctx, tx, teleportrunes.Table, false); err != nil {
		return "", err
	}
	backup, err := backupDatabaseCatalog(ctx, root, catalog.DBPath, "teleport-runes", func(snapshot *sql.DB) error {
		old, err := teleportrunes.Read(ctx, snapshot)
		if err != nil || !slices.Equal(old, current) {
			return errors.Join(errors.New("teleport rune backup failed verification"), err)
		}
		return nil
	})
	if err != nil {
		return backup, err
	}
	proposed := make(map[int16]teleportrunes.Rune, len(rules))
	for _, rule := range rules {
		proposed[rule.Index] = rule
	}
	previous := make(map[int16]teleportrunes.Rune, len(current))
	for _, rule := range current {
		previous[rule.Index] = rule
		if _, exists := proposed[rule.Index]; !exists {
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+teleportrunes.Table+` WHERE "Index"=?`, rule.Index); err != nil {
				return backup, err
			}
		}
	}
	for _, rule := range rules {
		if previous[rule.Index] == rule {
			continue
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO `+teleportrunes.Table+` ("Index", LightMapID, LightX, LightY, LightZ, FuryMapID, FuryX, FuryY, FuryZ)
			VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT("Index") DO UPDATE SET
			LightMapID=excluded.LightMapID, LightX=excluded.LightX, LightY=excluded.LightY, LightZ=excluded.LightZ,
			FuryMapID=excluded.FuryMapID, FuryX=excluded.FuryX, FuryY=excluded.FuryY, FuryZ=excluded.FuryZ`,
			rule.Index, rule.Light.MapID, rule.Light.X, rule.Light.Y, rule.Light.Z,
			rule.Fury.MapID, rule.Fury.X, rule.Fury.Y, rule.Fury.Z)
		if err != nil {
			return backup, err
		}
	}
	written, err := teleportrunes.Read(ctx, tx)
	if err != nil || !slices.Equal(written, rules) {
		return backup, errors.Join(errors.New("saved teleport runes failed verification"), err)
	}
	return backup, tx.Commit()
}
