package db

import (
	"context"
	"testing"
)

func TestMigrationsAreAtomicAndDetectChanges(t *testing.T) {
	p := sqliteFixture(t)
	broken := append(append([]migration(nil), migrations...), migration{Version: len(migrations) + 1, Name: "broken", SQL: `CREATE TABLE GX_Partial(Value INTEGER); INSERT INTO GX_Missing VALUES(1)`})
	if err := migrate(context.Background(), p, broken); err == nil {
		t.Fatal("broken migration succeeded")
	}
	var count int
	if err := p.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='GX_Partial'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("partial migration survived", count, err)
	}
	if err := p.QueryRow(`SELECT COUNT(*) FROM GX_SchemaMigrations`).Scan(&count); err != nil || count != len(migrations) {
		t.Fatal("migration ledger changed", count, err)
	}
	changed := append([]migration(nil), migrations...)
	changed[0].SQL += " "
	if err := migrate(context.Background(), p, changed); err == nil {
		t.Fatal("changed migration accepted")
	}
	if err := migrate(context.Background(), p, migrations); err != nil {
		t.Fatal(err)
	}
}
