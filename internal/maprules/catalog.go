package maprules

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const Properties = "PS_GameDefs_MapProperties"
const Starts = "PS_GameDefs_CharacterStartPositions"

//go:embed migration.sql
var MigrationSQL string

type Row []any
type Catalog struct {
	Maps, Starts []Row
}
type Field struct {
	Name     string
	Kind     byte // i integer, f coordinate, s native code
	Optional bool
}

func propertyFields() []Field {
	var fields []Field
	add := func(names string, kind byte, nullable bool) {
		for _, name := range strings.Fields(names) {
			fields = append(fields, Field{name, kind, nullable})
		}
	}
	add("MapID", 'i', false)
	add("MapType WarType CreateType", 's', false)
	add("MinLevel MaxLevel CreateMinUser EnterMaxUser ExpireMinutes WeatherType WeatherFrequency MapCurse1 MapCurse2 MapCurse3 MapModifier1 MapModifier2 MapModifier3", 'i', false)
	for _, side := range []string{"Light", "Fury"} {
		prefix := side + "Rebirth"
		add(prefix+"MapID", 'i', true)
		for _, axis := range []string{"X", "Y", "Z"} {
			add(prefix+axis, 'f', true)
		}
	}
	for i := 1; i <= 7; i++ {
		prefix := fmt.Sprintf("Schedule%d", i)
		add(prefix+"Day "+prefix+"StartHour "+prefix+"DurationHours", 'i', true)
	}
	return fields
}

var MapFields = propertyFields()
var StartFields = []Field{{"Race", 'i', false}, {"MapID", 'i', false}, {"X", 'f', false}, {"Y", 'f', false}, {"Z", 'f', false}}

func ParseValue(f Field, text string) (any, error) {
	text = strings.TrimSpace(text)
	if f.Optional && text == "" {
		return nil, nil
	}
	switch f.Kind {
	case 's':
		return strings.ToUpper(text), nil
	case 'f':
		v, err := strconv.ParseFloat(text, 32)
		if err == nil && (math.IsNaN(v) || math.IsInf(v, 0)) {
			return nil, fmt.Errorf("coordinate must be finite")
		}
		return v, err
	default:
		return strconv.ParseInt(text, 10, 32)
	}
}
func columns(fields []Field) string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return strings.Join(names, ",")
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func Read(ctx context.Context, q queryer) (Catalog, error) {
	var c Catalog
	var err error
	c.Maps, err = readRows(ctx, q, Properties, MapFields)
	if err != nil {
		return Catalog{}, err
	}
	c.Starts, err = readRows(ctx, q, Starts, StartFields)
	if err != nil {
		return Catalog{}, err
	}
	if len(c.Starts) != 4 {
		return Catalog{}, fmt.Errorf("expected exactly four character start positions")
	}
	for _, row := range c.Starts {
		if err := validateBirth(c, row); err != nil {
			return Catalog{}, err
		}
	}
	return c, nil
}
func validateBirth(c Catalog, start Row) error {
	for _, row := range c.Maps {
		if row[0] != start[1] {
			continue
		}
		if row[4] != int64(0) || row[5] != int64(0) || row[3] != "" {
			return fmt.Errorf("birth map %v must allow every level and must be a shared map", start[1])
		}
		return nil
	}
	return fmt.Errorf("birth map %v has no map properties", start[1])
}
func readRows(ctx context.Context, q queryer, table string, fields []Field) ([]Row, error) {
	rows, err := q.QueryContext(ctx, "SELECT "+columns(fields)+" FROM "+table+" ORDER BY "+fields[0].Name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		r := make(Row, len(fields))
		targets := make([]any, len(r))
		for i := range r {
			targets[i] = &r[i]
		}
		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}
		for i, v := range r {
			if f, ok := v.(float64); ok && (math.IsInf(f, 0) || math.IsNaN(f) || math.IsInf(float64(float32(f)), 0)) {
				return nil, fmt.Errorf("%s.%s contains an invalid coordinate", table, fields[i].Name)
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func SetRow(ctx context.Context, tx *sql.Tx, table string, fields []Field, row Row) error {
	assign := make([]string, len(fields)-1)
	for i, f := range fields[1:] {
		assign[i] = f.Name + "=?"
	}
	args := append(append([]any{}, row[1:]...), row[0])
	result, err := tx.ExecContext(ctx, "UPDATE "+table+" SET "+strings.Join(assign, ",")+" WHERE "+fields[0].Name+"=?", args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count != 1 {
		return fmt.Errorf("map or race row %v no longer exists", row[0])
	}
	return err
}
