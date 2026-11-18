// Package enchantrules defines the SQLite catalogs shared by GX World and Studio.
package enchantrules

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
)

const MaxLevel = 49
const AttackTable = "PS_GameDefs_EnchantAttack"
const DefenseTable = "PS_GameDefs_EnchantDefense"
const CreateSQL = `CREATE TABLE IF NOT EXISTS PS_GameDefs_EnchantAttack (
	Level INTEGER PRIMARY KEY CHECK(Level BETWEEN 0 AND 49),
	SuccessChance INTEGER NOT NULL CHECK(SuccessChance BETWEEN 0 AND 1000000),
	AttackBonus INTEGER NOT NULL CHECK(AttackBonus BETWEEN 0 AND 2147483647)
) STRICT;
CREATE TABLE IF NOT EXISTS PS_GameDefs_EnchantDefense (
	Level INTEGER PRIMARY KEY CHECK(Level BETWEEN 0 AND 49),
	SuccessChance INTEGER NOT NULL CHECK(SuccessChance BETWEEN 0 AND 1000000),
	AbsorptionBonus INTEGER NOT NULL CHECK(AbsorptionBonus BETWEEN 0 AND 2147483647)
) STRICT`

type Level struct {
	Number        uint8
	SuccessChance uint32
	Bonus         uint32
}

type Catalog struct {
	Attack, Defense []Level
}

func (c Catalog) Equal(other Catalog) bool {
	return slices.Equal(c.Attack, other.Attack) && slices.Equal(c.Defense, other.Defense)
}

var ErrEmpty = errors.New("enchant tables have no definitions")

// Cap expects consecutive rows ordered from +0.
func Cap(levels []Level) (uint8, error) {
	if len(levels) == 0 {
		return 0, errors.New("level +0 is required")
	}
	for i, row := range levels {
		if i > MaxLevel || int(row.Number) != i {
			return 0, fmt.Errorf("expected enchant level +%d (maximum +%d), got +%d", i, MaxLevel, row.Number)
		}
		if row.SuccessChance > 1000000 || row.Bonus > 2147483647 {
			return 0, fmt.Errorf("invalid chance or bonus at level +%d", row.Number)
		}
	}
	return uint8(len(levels) - 1), nil
}

func (c Catalog) Caps() (attack, defense uint8, err error) {
	if attack, err = Cap(c.Attack); err != nil {
		return 0, 0, fmt.Errorf("%s: %w", AttackTable, err)
	}
	if defense, err = Cap(c.Defense); err != nil {
		return 0, 0, fmt.Errorf("%s: %w", DefenseTable, err)
	}
	return attack, defense, nil
}

type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func Read(ctx context.Context, db Queryer) (Catalog, error) {
	var catalog Catalog
	// A single statement reads both tables from the same SQLite snapshot.
	rows, err := db.QueryContext(ctx, `SELECT 0 AS Family, Level, SuccessChance, AttackBonus FROM `+AttackTable+
		` UNION ALL SELECT 1, Level, SuccessChance, AbsorptionBonus FROM `+DefenseTable+` ORDER BY Family, Level`)
	if err != nil {
		return catalog, err
	}
	defer rows.Close()
	for rows.Next() {
		var family uint8
		var row Level
		if err := rows.Scan(&family, &row.Number, &row.SuccessChance, &row.Bonus); err != nil {
			return catalog, err
		}
		if family == 0 {
			catalog.Attack = append(catalog.Attack, row)
		} else {
			catalog.Defense = append(catalog.Defense, row)
		}
	}
	if err := rows.Err(); err != nil {
		return catalog, err
	}
	if len(catalog.Attack) == 0 && len(catalog.Defense) == 0 {
		return catalog, ErrEmpty
	}
	_, _, err = catalog.Caps()
	return catalog, err
}

func Replace(ctx context.Context, tx *sql.Tx, catalog Catalog) error {
	if _, _, err := catalog.Caps(); err != nil {
		return err
	}
	for _, rules := range []struct {
		table, bonus string
		levels       []Level
	}{
		{AttackTable, "AttackBonus", catalog.Attack},
		{DefenseTable, "AbsorptionBonus", catalog.Defense},
	} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+rules.table); err != nil {
			return err
		}
		for _, row := range rules.levels {
			if _, err := tx.ExecContext(ctx, "INSERT INTO "+rules.table+" (Level, SuccessChance, "+rules.bonus+") VALUES (?, ?, ?)",
				row.Number, row.SuccessChance, row.Bonus); err != nil {
				return err
			}
		}
	}
	return nil
}
