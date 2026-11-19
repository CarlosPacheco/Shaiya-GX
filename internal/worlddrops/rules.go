package worlddrops

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
)

const (
	Table     = "PS_GameDefs_WorldDrops"
	Scale     = 100_000_000
	RollRange = 100 * Scale
)

//go:embed migration.sql
var MigrationSQL string

type Rule struct {
	Index                     int64
	Description               string
	Grade, MinLevel, MaxLevel uint16
	DropChance                uint64
}

func FormatChance(units uint64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%d.%08d", units/Scale, units%Scale), "0"), ".")
}

func ParseChance(text string) (uint64, error) {
	text = strings.TrimSpace(text)
	if text == "" || text == "." {
		return 0, fmt.Errorf("enter a drop percentage")
	}
	whole, fraction, _ := strings.Cut(text, ".")
	if whole == "" {
		whole = "0"
	}
	integer, err := strconv.ParseUint(whole, 10, 8)
	if err != nil || len(fraction) > 8 {
		return 0, fmt.Errorf("enter a percentage from 0 to 100 with up to 8 decimal places")
	}
	decimal, err := strconv.ParseUint(fraction+strings.Repeat("0", 8-len(fraction)), 10, 32)
	if err != nil || integer*Scale+decimal > RollRange {
		return 0, fmt.Errorf("enter a percentage from 0 to 100 with up to 8 decimal places")
	}
	return integer*Scale + decimal, nil
}

func Validate(rules []Rule) error {
	var previous int64
	for _, r := range rules {
		if r.Index <= previous || r.Grade < 1 || r.Grade > 1024 || r.MinLevel < 1 || r.MaxLevel < r.MinLevel || r.DropChance > RollRange {
			return fmt.Errorf("invalid world drop rule %d: check index, grade, level range and chance", r.Index)
		}
		previous = r.Index
	}
	return nil
}

func Read(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) ([]Rule, error) {
	rows, err := db.QueryContext(ctx, `SELECT "Index", Description, Grade, MinLevel, MaxLevel, DropChance FROM `+Table+` ORDER BY "Index"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.Index, &r.Description, &r.Grade, &r.MinLevel, &r.MaxLevel, &r.DropChance); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rules, Validate(rules)
}
