package blessingrules

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
)

//go:embed 0013.sql
var MigrationSQL string

var Tables = [2]string{"PS_GameDefs_BlessingLight", "PS_GameDefs_BlessingFury"}

type Tier struct {
	Number        uint8
	RequiredKills uint32
	Bonuses       [BonusCount]uint16
}

type Catalog [2][]Tier

func (c Catalog) Validate() error {
	if len(c[0]) == 0 || len(c[0]) != len(c[1]) {
		return fmt.Errorf("Light and Fury require the same nonzero number of blessing tiers")
	}
	for side, tiers := range c {
		var previous uint32
		for i, tier := range tiers {
			if int(tier.Number) != i+1 || tier.RequiredKills <= previous {
				return fmt.Errorf("%s tier %d requires consecutive levels and increasing kill thresholds", Tables[side], i+1)
			}
			if tier.Bonuses[deathEXPLossReduction-1] > 100 {
				return fmt.Errorf("tier %d: death EXP loss reduction must be between 0 and 100%%", i+1)
			}
			if tier.Bonuses != c[0][i].Bonuses {
				return fmt.Errorf("blessing bonuses differ between Light and Fury at tier %d", i+1)
			}
			previous = tier.RequiredKills
		}
	}
	return nil
}

func (c Catalog) Effective(side, index int) (result [BonusCount]uint16) {
	for _, tier := range c[side][:index+1] {
		for i, bonus := range tier.Bonuses {
			result[i] = max(result[i], bonus)
		}
	}
	return
}

// ReadDB takes one SQLite snapshot for both faction tables.
func ReadDB(ctx context.Context, db *sql.DB) (Catalog, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Catalog{}, err
	}
	defer tx.Rollback()
	return Read(ctx, tx)
}

func Read(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) (Catalog, error) {
	var catalog Catalog
	for side, table := range Tables {
		columns := []string{"Tier", "RequiredKills"}
		for _, effect := range Effects {
			columns = append(columns, effect.Column)
		}
		rows, err := db.QueryContext(ctx, "SELECT "+strings.Join(columns, ",")+" FROM "+table+" ORDER BY Tier")
		if err != nil {
			return Catalog{}, err
		}
		for rows.Next() {
			var tier Tier
			dest := []any{&tier.Number, &tier.RequiredKills}
			for i := range tier.Bonuses {
				dest = append(dest, &tier.Bonuses[i])
			}
			if err := rows.Scan(dest...); err != nil {
				rows.Close()
				return Catalog{}, err
			}
			catalog[side] = append(catalog[side], tier)
		}
		if err := errors.Join(rows.Err(), rows.Close()); err != nil {
			return Catalog{}, err
		}
	}
	return catalog, catalog.Validate()
}
