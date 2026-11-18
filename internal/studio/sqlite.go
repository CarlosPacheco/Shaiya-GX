package studio

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"modernc.org/sqlite"
)

type sqlRow struct {
	RowID      sql.NullInt64
	Code, Name string
	Cost       uint32
	IDs        [cash.Slots]sql.Null[uint32]
	Counts     [cash.Slots]sql.Null[uint16]
}

func (r sqlRow) product() SQLProduct {
	p := SQLProduct{Code: strings.TrimRight(r.Code, " "), Name: r.Name, Cost: r.Cost}
	for i := range p.Items {
		p.Items[i] = cash.Item{ItemID: r.IDs[i].V, Count: r.Counts[i].V}
	}
	return p
}

var productColumns = func() []string {
	columns := []string{"RowID", "ProductCode", "ProductName", "BuyCost"}
	for i := 1; i <= cash.Slots; i++ {
		columns = append(columns, fmt.Sprintf("ItemID%d", i), fmt.Sprintf("ItemCount%d", i))
	}
	return columns
}()

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readRows(ctx context.Context, db queryer) ([]sqlRow, error) {
	rows, err := db.QueryContext(ctx, "SELECT "+strings.Join(productColumns, ",")+" FROM "+ProductTable+" ORDER BY ProductCode COLLATE BINARY")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ProductTable, err)
	}
	defer rows.Close()
	var result []sqlRow
	for rows.Next() {
		var row sqlRow
		dest := []any{&row.RowID, &row.Code, &row.Name, &row.Cost}
		for i := range row.IDs {
			dest = append(dest, &row.IDs[i], &row.Counts[i])
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func writeRows(ctx context.Context, tx *sql.Tx, old []sqlRow, products []cash.Product) error {
	if err := safeTable(ctx, tx, ProductTable, false); err != nil {
		return err
	}

	byCode := make(map[string]sqlRow, len(old))
	var lastRowID int64
	desired := make(map[string]bool, len(products))
	for _, product := range products {
		desired[product.Code] = true
	}
	for _, row := range old {
		code := row.product().Code
		byCode[code] = row
		if row.RowID.Valid {
			lastRowID = max(lastRowID, row.RowID.Int64)
		}
		if !desired[code] {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+ProductTable+" WHERE ProductCode COLLATE BINARY = ?", row.Code); err != nil {
				return err
			}
		}
	}
	for _, product := range products {
		current, exists := byCode[product.Code]
		if exists && current.Code == product.Code && current.product() == Shared(product) {
			continue
		}
		if !exists {
			if lastRowID == math.MaxInt64 {
				return errors.New("ProductList RowID range is exhausted")
			}
			lastRowID++
			current.RowID = sql.NullInt64{Int64: lastRowID, Valid: true}
		}
		if err := writeProduct(ctx, tx, current, exists, product); err != nil {
			return fmt.Errorf("product %q: %w", product.Code, err)
		}
	}
	return nil
}

func writeProduct(ctx context.Context, tx *sql.Tx, old sqlRow, exists bool, product cash.Product) error {
	if !exists {
		columns := productColumns
		values := []any{old.RowID.Int64, product.Code, product.Name, product.Cost}
		for _, item := range product.Items {
			values = append(values, item.ItemID, item.Count)
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO "+ProductTable+" ("+strings.Join(columns, ",")+") VALUES ("+strings.TrimSuffix(strings.Repeat("?,", len(columns)), ",")+")", values...)
		return err
	}

	var assignments []string
	var values []any
	add := func(column string, value any) {
		assignments = append(assignments, column+" = ?")
		values = append(values, value)
	}
	current := old.product()
	if old.Code != product.Code {
		add("ProductCode", product.Code)
	}
	if current.Name != product.Name {
		add("ProductName", product.Name)
	}
	if current.Cost != product.Cost {
		add("BuyCost", product.Cost)
	}
	for i, item := range product.Items {
		if current.Items[i].ItemID != item.ItemID {
			add(productColumns[4+2*i], item.ItemID)
		}
		if current.Items[i].Count != item.Count {
			add(productColumns[5+2*i], item.Count)
		}
	}
	values = append(values, old.Code)
	_, err := tx.ExecContext(ctx, "UPDATE "+ProductTable+" SET "+strings.Join(assignments, ",")+" WHERE ProductCode COLLATE BINARY = ?", values...)
	return err
}

func validateItems(ctx context.Context, db queryer, products []cash.Product) error {
	codes := make(map[string]bool, len(products))
	checked := make(map[uint32]bool)
	for _, p := range products {
		key := strings.ToLower(p.Code)
		if codes[key] {
			return fmt.Errorf("duplicate product code under SQLite NOCASE: %q", p.Code)
		}
		codes[key] = true
		hasItems := false
		for i, item := range p.Items {
			if item.ItemID == 0 {
				continue
			}
			hasItems = true
			if checked[item.ItemID] {
				continue
			}
			var exists bool
			err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM PS_GameDefs_Items WHERE Type = ? AND TypeID = ? AND ItemID = ?)", item.ItemID/1000, item.ItemID%1000, item.ItemID).Scan(&exists)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("product %q slot %d: item %d is missing in SQLite", p.Code, i+1, item.ItemID)
			}
			checked[item.ItemID] = true
		}
		if !hasItems {
			return fmt.Errorf("product %q needs at least one item for GX Server", p.Code)
		}
	}
	return nil
}

func backupDB(ctx context.Context, db *sql.DB, path string) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(driverConn any) error {
		provider, ok := driverConn.(interface {
			NewBackup(string) (*sqlite.Backup, error)
		})
		if !ok {
			return errors.New("SQLite backup API unavailable")
		}
		backup, err := provider.NewBackup(dbURI(path, "rwc"))
		if err != nil {
			return err
		}
		for {
			if err = ctx.Err(); err != nil {
				break
			}
			more, stepErr := backup.Step(256)
			if stepErr != nil || !more {
				err = stepErr
				break
			}
		}
		return errors.Join(err, backup.Finish())
	})
}

func integrity(ctx context.Context, db queryer) error {
	var result string
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check(1)").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("SQLite integrity check: %s", result)
	}
	return nil
}

func safeTable(ctx context.Context, tx *sql.Tx, table string, definitions bool) error {
	triggers, err := tx.QueryContext(ctx, "SELECT sql FROM sqlite_schema WHERE type = 'trigger' AND tbl_name = ? COLLATE NOCASE", table)
	if err != nil {
		return err
	}
	// The GX seed has these identity triggers. They fire on INSERT only;
	// definition saves use UPDATE and preserve the existing RowID.
	identity := fmt.Sprintf(`CREATE TRIGGER "%[1]s_identity" AFTER INSERT ON "%[1]s" WHEN NEW."RowID"=0 BEGIN UPDATE "%[1]s" SET "RowID"=(SELECT COALESCE(MAX("RowID"),0)+1 FROM "%[1]s" WHERE _rowid_<>NEW._rowid_) WHERE _rowid_=NEW._rowid_; END`, table)
	for triggers.Next() {
		var ddl string
		if err := triggers.Scan(&ddl); err != nil {
			triggers.Close()
			return err
		}
		if !definitions || ddl != identity {
			triggers.Close()
			return fmt.Errorf("custom triggers on %s require review before Studio can write", table)
		}
	}
	if err := errors.Join(triggers.Err(), triggers.Close()); err != nil {
		return err
	}
	// Defer deletion checks only through SQLite's normal foreign-key enforcement.
	// GX has no child tables for ProductList. Refuse custom cascading schemas.
	rows, err := tx.QueryContext(ctx, "SELECT m.name FROM sqlite_schema m, pragma_foreign_key_list(m.name) f WHERE m.type = 'table' AND f.[table] = ? COLLATE NOCASE LIMIT 1", table)
	if err != nil {
		return err
	}
	hasChild := rows.Next()
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return err
	}
	if hasChild {
		return fmt.Errorf("custom foreign keys to %s require review before Studio can write", table)
	}

	return nil
}
