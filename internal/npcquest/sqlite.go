package npcquest

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
)

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func identifiers(t Table) []string {
	out := make([]string, len(t.Fields))
	for i, f := range t.Fields {
		out[i] = `"` + f.Name + `"`
	}
	return out
}

func Read(ctx context.Context, db queryer) (Catalog, error) {
	var c Catalog
	for i, t := range Tables {
		rows, err := readTable(ctx, db, t)
		if err != nil {
			return Catalog{}, err
		}
		c[i] = rows
	}
	return c, nil
}

func readTable(ctx context.Context, db queryer, t Table) ([]Row, error) {
	cols := identifiers(t)
	rows, err := db.QueryContext(ctx, "SELECT "+strings.Join(cols, ",")+" FROM "+t.Name+" ORDER BY "+strings.Join(cols[:t.Keys], ","))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		row := make(Row, len(cols))
		dest := make([]any, len(cols))
		for i := range row {
			dest[i] = &row[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func Update(ctx context.Context, tx *sql.Tx, before, after Catalog) error {
	for i, t := range Tables {
		sameKeys := slices.EqualFunc(before[i], after[i], func(a, b Row) bool { return slices.Equal(a[:t.Keys], b[:t.Keys]) })
		if !sameKeys {
			if i != Products && i != NPCLinks && i != ItemLinks {
				return fmt.Errorf("cannot change the keys of %s", t.Name)
			}
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+t.Name); err != nil {
				return err
			}
			if err := insertRows(ctx, tx, t, after[i]); err != nil {
				return err
			}
			continue
		}
		cols := identifiers(t)
		pred := make([]string, t.Keys)
		for j := range pred {
			pred[j] = cols[j] + "=?"
		}
		for j, row := range after[i] {
			var sets []string
			var args []any
			for k, col := range cols {
				if row[k] == before[i][j][k] {
					continue
				}
				if k < t.Keys {
					return fmt.Errorf("cannot change %s.%s", t.Name, col)
				}
				sets = append(sets, col+"=?")
				args = append(args, row[k])
			}
			if len(sets) == 0 {
				continue
			}
			args = append(args, row[:t.Keys]...)
			if _, err := tx.ExecContext(ctx, "UPDATE "+t.Name+" SET "+strings.Join(sets, ",")+" WHERE "+strings.Join(pred, " AND "), args...); err != nil {
				return err
			}
		}
	}
	return nil
}

func Insert(ctx context.Context, tx *sql.Tx, c Catalog) error {
	for i, t := range Tables {
		if err := insertRows(ctx, tx, t, c[i]); err != nil {
			return err
		}
	}
	return nil
}

func insertRows(ctx context.Context, tx *sql.Tx, t Table, rows []Row) error {
	cols := identifiers(t)
	q := "INSERT INTO " + t.Name + " (" + strings.Join(cols, ",") + ") VALUES (?" + strings.Repeat(",?", len(cols)-1) + ")"
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, row := range rows {
		if _, err := stmt.ExecContext(ctx, row...); err != nil {
			return fmt.Errorf("%s: %w", t.Name, err)
		}
	}
	return nil
}
