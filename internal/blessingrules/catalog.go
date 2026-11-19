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

//go:embed 0014.sql
var MergeSQL string

const Table = "PS_GameDefs_GoddessBlessing"

type Tier struct {
	Number        uint8
	RequiredKills uint32
	Bonuses       [BonusCount]uint16
}

type Catalog []Tier

func (c Catalog) Validate() error {
	if len(c) == 0 {
		return fmt.Errorf("Goddess Blessing requires at least one tier")
	}
	var previous uint32
	for i, tier := range c {
		if int(tier.Number) != i+1 || tier.RequiredKills <= previous {
			return fmt.Errorf("blessing tier %d requires consecutive levels and increasing kill thresholds", i+1)
		}
		if tier.Bonuses[deathEXPLossReduction-1] > 100 {
			return fmt.Errorf("tier %d: death EXP loss reduction must be between 0 and 100%%", i+1)
		}
		previous = tier.RequiredKills
	}
	return nil
}

func (c Catalog) Effective(index int) (result [BonusCount]uint16) {
	for _, tier := range c[:index+1] {
		for i, bonus := range tier.Bonuses {
			result[i] = max(result[i], bonus)
		}
	}
	return
}

func Read(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) (Catalog, error) {
	var catalog Catalog
	columns := []string{"Tier", "RequiredKills"}
	for _, effect := range Effects {
		columns = append(columns, effect.Column)
	}
	rows, err := db.QueryContext(ctx, "SELECT "+strings.Join(columns, ",")+" FROM "+Table+" ORDER BY Tier")
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
		catalog = append(catalog, tier)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return Catalog{}, err
	}
	return catalog, catalog.Validate()
}
