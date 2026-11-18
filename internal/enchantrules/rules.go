// Package enchantrules defines the SQLite catalogs shared by GX World and Studio.
package enchantrules

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
)

const MaxLevel = 49
const AttackTable = "PS_GameDefs_EnchantAttack"
const DefenseTable = "PS_GameDefs_EnchantDefense"

// LegacyCreateSQL is frozen for migration 10 and its checksum.
const LegacyCreateSQL = `CREATE TABLE IF NOT EXISTS PS_GameDefs_EnchantAttack (
	Level INTEGER PRIMARY KEY CHECK(Level BETWEEN 0 AND 49),
	SuccessChance INTEGER NOT NULL CHECK(SuccessChance BETWEEN 0 AND 1000000),
	AttackBonus INTEGER NOT NULL CHECK(AttackBonus BETWEEN 0 AND 2147483647)
) STRICT;
CREATE TABLE IF NOT EXISTS PS_GameDefs_EnchantDefense (
	Level INTEGER PRIMARY KEY CHECK(Level BETWEEN 0 AND 49),
	SuccessChance INTEGER NOT NULL CHECK(SuccessChance BETWEEN 0 AND 1000000),
	AbsorptionBonus INTEGER NOT NULL CHECK(AbsorptionBonus BETWEEN 0 AND 2147483647)
) STRICT`

const CreateSQL = `CREATE TABLE IF NOT EXISTS PS_GameDefs_EnchantAttack (
	Level INTEGER PRIMARY KEY CHECK(Level BETWEEN 1 AND 49),
	SuccessChance INTEGER NOT NULL CHECK(SuccessChance BETWEEN 0 AND 1000000),
	AttackBonus INTEGER NOT NULL CHECK(AttackBonus BETWEEN 0 AND 2147483647)
) STRICT;
CREATE TABLE IF NOT EXISTS PS_GameDefs_EnchantDefense (
	Level INTEGER PRIMARY KEY CHECK(Level BETWEEN 1 AND 49),
	SuccessChance INTEGER NOT NULL CHECK(SuccessChance BETWEEN 0 AND 1000000),
	AbsorptionBonus INTEGER NOT NULL CHECK(AbsorptionBonus BETWEEN 0 AND 2147483647)
) STRICT`

// UpgradeSQL rekeys chances by the level reached. A staged seed may already
// use the new layout; it can pass through migration 11 without shifting again.
const UpgradeSQL = `ALTER TABLE PS_GameDefs_EnchantAttack RENAME TO GX_EnchantAttackBefore11;
ALTER TABLE PS_GameDefs_EnchantDefense RENAME TO GX_EnchantDefenseBefore11;
` + CreateSQL + `;
INSERT INTO PS_GameDefs_EnchantAttack
SELECT current.Level,
       CASE WHEN EXISTS (SELECT 1 FROM GX_EnchantAttackBefore11 WHERE Level=0)
            THEN previous.SuccessChance ELSE current.SuccessChance END, current.AttackBonus
FROM GX_EnchantAttackBefore11 AS current
LEFT JOIN GX_EnchantAttackBefore11 AS previous ON previous.Level = current.Level-1
WHERE current.Level > 0;
INSERT INTO PS_GameDefs_EnchantDefense
SELECT current.Level,
       CASE WHEN EXISTS (SELECT 1 FROM GX_EnchantDefenseBefore11 WHERE Level=0)
            THEN previous.SuccessChance ELSE current.SuccessChance END, current.AbsorptionBonus
FROM GX_EnchantDefenseBefore11 AS current
LEFT JOIN GX_EnchantDefenseBefore11 AS previous ON previous.Level = current.Level-1
WHERE current.Level > 0;
DROP TABLE GX_EnchantAttackBefore11;
DROP TABLE GX_EnchantDefenseBefore11`

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

// Cap expects consecutive target levels starting at +1. Empty disables forging.
func Cap(levels []Level) (uint8, error) {
	for i, row := range levels {
		if i >= MaxLevel || int(row.Number) != i+1 {
			return 0, fmt.Errorf("expected enchant level +%d (maximum +%d), got +%d", i+1, MaxLevel, row.Number)
		}
		if row.SuccessChance > 1000000 || row.Bonus > 2147483647 {
			return 0, fmt.Errorf("invalid chance or bonus at level +%d", row.Number)
		}
	}
	return uint8(len(levels)), nil
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
