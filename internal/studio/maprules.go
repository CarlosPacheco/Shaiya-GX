package studio

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"path/filepath"
	"reflect"

	"github.com/Spelunkern/Shaiya-GX/internal/maprules"
)

type MapRulesCatalog struct {
	DBPath string
	Rules  maprules.Catalog
	File   *ServerFile
}

func OpenMapRules(ctx context.Context, path string) (*MapRulesCatalog, error) {
	if filepath.Base(path) == catalogfiles.Map {
		file, value, err := openServerCatalog(path, catalogfiles.ParseMap)
		if err != nil {
			return nil, err
		}
		return &MapRulesCatalog{File: file, Rules: value}, nil
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
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rules, err := maprules.Read(ctx, tx)
	if err != nil {
		return nil, err
	}
	return &MapRulesCatalog{DBPath: path, Rules: rules}, nil
}

func SaveMapRules(ctx context.Context, root string, catalog *MapRulesCatalog, rules maprules.Catalog) (string, error) {
	if catalog.File != nil {
		next, err := catalogfiles.EncodeMap(rules, catalog.File.original)
		if err != nil {
			return "", err
		}
		return saveServerCatalog(root, catalog.File, next, catalogfiles.ParseMap, func(v maprules.Catalog) bool { return reflect.DeepEqual(v, rules) })
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
	current, err := maprules.Read(ctx, tx)
	if err != nil {
		return "", err
	}
	if !reflect.DeepEqual(current, catalog.Rules) {
		return "", errors.New("map rules changed on disk; reopen the catalog")
	}
	for _, name := range []string{maprules.Properties, maprules.Starts} {
		if err := safeTable(ctx, tx, name, false); err != nil {
			return "", err
		}
	}
	backup, err := backupDatabaseCatalog(ctx, root, catalog.DBPath, "map-rules", func(snapshot *sql.DB) error {
		before, err := maprules.Read(ctx, snapshot)
		if err != nil || !reflect.DeepEqual(before, current) {
			return errors.Join(errors.New("map rules backup failed verification"), err)
		}
		return nil
	})
	if err != nil {
		return backup, err
	}
	for _, t := range []struct {
		name          string
		fields        []maprules.Field
		before, after []maprules.Row
	}{
		{maprules.Properties, maprules.MapFields, current.Maps, rules.Maps},
		{maprules.Starts, maprules.StartFields, current.Starts, rules.Starts},
	} {
		if len(t.before) != len(t.after) {
			return backup, errors.New("keep all existing map and race rows")
		}
		for i, row := range t.after {
			if reflect.DeepEqual(t.before[i], row) {
				continue
			}
			if len(row) != len(t.fields) || row[0] != t.before[i][0] {
				return backup, errors.New("invalid map or race row")
			}
			if err := maprules.SetRow(ctx, tx, t.name, t.fields, row); err != nil {
				return backup, err
			}
		}
	}
	written, err := maprules.Read(ctx, tx)
	if err != nil || !reflect.DeepEqual(written, rules) {
		return backup, errors.Join(errors.New("saved map rules failed verification"), err)
	}
	for _, row := range written.Maps {
		for _, id := range row[11:14] {
			if id == int64(0) {
				continue
			}
			var count int
			err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM PS_GameDefs_Skills WHERE SkillID=? AND SkillLevel=1 AND TypeEffect IN (3,4)", id).Scan(&count)
			if err != nil {
				return backup, err
			}
			if count != 1 {
				return backup, fmt.Errorf("map %v: curse %v requires a level 1 applied skill", row[0], id)
			}
		}
	}
	return backup, tx.Commit()
}
