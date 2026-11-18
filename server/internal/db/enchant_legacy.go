package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
)

// ReadLegacyEnchant is used only to initialize a pre-migration installation.
func ReadLegacyEnchant(path string) (enchantrules.Catalog, error) {
	ini, err := config.ReadGameINI(path)
	if err != nil {
		return enchantrules.Catalog{}, err
	}
	cap := 20
	var weaponRate, defenseRate [enchantrules.MaxLevel + 1]uint32
	if raw, ok := ini["ENCHANT"]["MAXLEVEL"]; ok {
		n, err := strconv.ParseUint(raw, 10, 8)
		if err != nil || n > enchantrules.MaxLevel {
			return enchantrules.Catalog{}, fmt.Errorf("MaxLevel must be 0..%d", enchantrules.MaxLevel)
		}
		cap = int(n)
	}
	var weapon, armor [enchantrules.MaxLevel + 1]uint32
	for _, table := range []struct {
		section, prefix string
		dst             *[enchantrules.MaxLevel + 1]uint32
		limit           uint64
	}{
		{"LAPISIANENCHANTPERCENTRATE", "WEAPON", &weaponRate, 1000000},
		{"LAPISIANENCHANTPERCENTRATE", "DEFENSE", &defenseRate, 1000000},
		{"LAPISIANENCHANTADDVALUE", "WEAPON", &weapon, 2147483647},
		{"LAPISIANENCHANTADDVALUE", "DEFENSE", &armor, 2147483647},
	} {
		for step := 0; step <= enchantrules.MaxLevel; step++ {
			key := fmt.Sprintf("%sSTEP%02d", table.prefix, step)
			raw, ok := ini[table.section][key]
			if !ok {
				if step <= cap {
					return enchantrules.Catalog{}, fmt.Errorf("[%s] missing %s", table.section, key)
				}
				continue
			}
			n, err := strconv.ParseUint(raw, 10, 32)
			if err != nil || n > table.limit {
				return enchantrules.Catalog{}, fmt.Errorf("[%s] %s must be 0..%d", table.section, key, table.limit)
			}
			table.dst[step] = uint32(n)
		}
	}
	var rows enchantrules.Catalog
	for level := 1; level <= cap; level++ {
		rows.Attack = append(rows.Attack, enchantrules.Level{Number: uint8(level), SuccessChance: weaponRate[level-1], Bonus: weapon[level]})
		rows.Defense = append(rows.Defense, enchantrules.Level{Number: uint8(level), SuccessChance: defenseRate[level-1], Bonus: armor[level]})
	}
	return rows, nil
}

// initializeEnchant11 runs inside migration 11 and never imports an INI again
// after that migration is recorded, including when both caps are later zero.
func initializeEnchant11(ctx context.Context, tx *sql.Tx) error {
	rules, err := enchantrules.Read(ctx, tx)
	if err != nil || len(rules.Attack) != 0 || len(rules.Defense) != 0 {
		return err
	}
	rules, err = ReadLegacyEnchant(filepath.Join(config.DataDir(), "game", "ItemEnchant.ini"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("import legacy enchant rules: %w", err)
	}
	return enchantrules.Replace(ctx, tx, rules)
}
