package teleportrunes

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"math"
)

//go:embed catalog.sql
var MigrationSQL string

const Table = "PS_GameDefs_TeleportRunes"

type Destination struct {
	MapID   uint16
	X, Y, Z float64
}

type Rune struct {
	Index       int16
	Light, Fury Destination
}

func ValidCoordinate(v float64) bool {
	return !math.IsNaN(v) && math.Abs(v) <= math.MaxFloat32
}

func Validate(runes []Rune) error {
	var previous int16
	for _, r := range runes {
		if r.Index <= previous {
			return fmt.Errorf("rune indices must be unique, positive and in ascending order")
		}
		previous = r.Index
		for _, d := range [...]Destination{r.Light, r.Fury} {
			if d.MapID > 999 || !ValidCoordinate(d.X) || !ValidCoordinate(d.Y) || !ValidCoordinate(d.Z) {
				return fmt.Errorf("rune %d has an invalid map or coordinate", r.Index)
			}
		}
	}
	return nil
}

func Read(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) ([]Rune, error) {
	rows, err := db.QueryContext(ctx, `SELECT "Index", LightMapID, LightX, LightY, LightZ,
		FuryMapID, FuryX, FuryY, FuryZ FROM `+Table+` ORDER BY "Index"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runes []Rune
	for rows.Next() {
		var r Rune
		if err := rows.Scan(&r.Index, &r.Light.MapID, &r.Light.X, &r.Light.Y, &r.Light.Z,
			&r.Fury.MapID, &r.Fury.X, &r.Fury.Y, &r.Fury.Z); err != nil {
			return nil, err
		}
		runes = append(runes, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return runes, Validate(runes)
}
