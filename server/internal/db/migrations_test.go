package db

import (
	"context"
	"database/sql"
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

func TestExperienceMigrationPreservesProgress(t *testing.T) {
	p, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if _, err := p.Exec(`
		CREATE TABLE PS_GameDefs_ExpDefs(Level INTEGER, Grow INTEGER, Exp INTEGER);
		INSERT INTO PS_GameDefs_ExpDefs VALUES (1,1,100),(1,4,200),(2,1,300),(2,4,124200),(3,1,300),(3,4,124200);
		CREATE TABLE PS_GameData_Chars(CharID INTEGER PRIMARY KEY,Level INTEGER,Grow INTEGER,Exp INTEGER);
		INSERT INTO PS_GameData_Chars VALUES (1,2,0,200),(2,3,0,300);
	`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(context.Background(), p, []migration{{1, "unified_experience", experienceMigration}}); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]int{1: 62100, 2: 124000} {
		var exp int
		if err := p.QueryRow("SELECT Exp FROM PS_GameData_Chars WHERE CharID=?", id).Scan(&exp); err != nil || exp != want {
			t.Fatalf("character %d exp=%d, want %d (%v)", id, exp, want, err)
		}
	}
	var cap, threshold int
	if err := p.QueryRow("SELECT Level, Exp FROM PS_GameDefs_ExpDefs ORDER BY Level DESC LIMIT 1").Scan(&cap, &threshold); err != nil || cap != 3 || threshold != 124000 {
		t.Fatalf("cap=%d exp=%d (%v)", cap, threshold, err)
	}
}
