package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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

func TestEnchantMigrationImportsLegacyOnlyOnce(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RAPTURE_DATA_DIR", dir)
	game := filepath.Join(dir, "game")
	if err := os.Mkdir(game, 0700); err != nil {
		t.Fatal(err)
	}
	ini, err := os.ReadFile(filepath.Join("..", "world", "testdata", "enchant-legacy.ini"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(game, "ItemEnchant.ini"), ini, 0600); err != nil {
		t.Fatal(err)
	}
	pool, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	steps := append([]migration(nil), migrations[:11]...)
	for i := range steps[:9] {
		steps[i].SQL = "SELECT 1"
	}
	if err := migrate(t.Context(), pool, steps); err != nil {
		t.Fatal(err)
	}
	var imported int
	if err := pool.QueryRow("SELECT COUNT(*) FROM PS_GameDefs_EnchantAttack").Scan(&imported); err != nil || imported != 20 {
		t.Fatalf("legacy levels imported: %d (%v)", imported, err)
	}
	if _, err := pool.Exec("DELETE FROM PS_GameDefs_EnchantAttack; DELETE FROM PS_GameDefs_EnchantDefense"); err != nil {
		t.Fatal(err)
	}
	if err := migrate(t.Context(), pool, steps); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow("SELECT COUNT(*) FROM PS_GameDefs_EnchantAttack").Scan(&imported); err != nil || imported != 0 {
		t.Fatalf("legacy levels returned: %d (%v)", imported, err)
	}
}

func TestMigrationChecksumsAcrossLineEndings(t *testing.T) {
	pool, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	steps := []migration{{1, "example", "CREATE TABLE Example(Value INTEGER);\r\nINSERT INTO Example VALUES(1);"}}
	if err := migrate(t.Context(), pool, steps); err != nil {
		t.Fatal(err)
	}
	legacy := fmt.Sprintf("%x", sha256.Sum256([]byte(steps[0].SQL)))
	steps[0].SQL = "CREATE TABLE Example(Value INTEGER);\nINSERT INTO Example VALUES(1);"
	var stored string
	if err := pool.QueryRow("SELECT Checksum FROM GX_SchemaMigrations").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("%x", sha256.Sum256([]byte(steps[0].SQL))); stored != want {
		t.Fatalf("stored checksum %s, want canonical %s", stored, want)
	}
	if _, err := pool.Exec("UPDATE GX_SchemaMigrations SET Checksum=?", legacy); err != nil {
		t.Fatal(err)
	}
	if err := migrate(t.Context(), pool, steps); err != nil {
		t.Fatal(err)
	}
	steps[0].SQL += " "
	if err := migrate(t.Context(), pool, steps); err == nil {
		t.Fatal("accepted a changed statement")
	}
}

func TestRemoveGrowthModesKeepsAllocatedPointsAndClaims(t *testing.T) {
	p, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if _, err := p.Exec(`
		CREATE TABLE PS_GameData_Chars(CharID INTEGER PRIMARY KEY,Grow INTEGER,Level INTEGER,StatPoint INTEGER,SkillPoint INTEGER,Str INTEGER,KillLevel INTEGER,DeadLevel INTEGER,Del INTEGER);
		INSERT INTO PS_GameData_Chars VALUES
			(1,0,10,3,2,50,0,0,0),(2,1,10,3,2,50,4,6,0),
			(3,2,10,3,2,50,4,6,0),(4,3,10,3,2,50,4,6,1);
		CREATE TABLE PS_GameData_UserMaxGrow(Country INTEGER,MaxGrow INTEGER);
		INSERT INTO PS_GameData_UserMaxGrow VALUES (1,3);
		CREATE TABLE PS_GameDefs_Skills(Grow INTEGER);
		INSERT INTO PS_GameDefs_Skills VALUES (3);
	`); err != nil {
		t.Fatal(err)
	}
	steps := []migration{{1, "remove_growth_modes", removeGrowthModesMigration}}
	for range 2 {
		if err := migrate(context.Background(), p, steps); err != nil {
			t.Fatal(err)
		}
	}
	for i, want := range [][2]int{{39, 20}, {39, 20}, {21, 11}, {3, 2}} {
		var grow, stats, skills, str, kills, deaths, deleted int
		err := p.QueryRow(`SELECT Grow,StatPoint,SkillPoint,Str,KillLevel,DeadLevel,Del FROM PS_GameData_Chars WHERE CharID=?`, i+1).
			Scan(&grow, &stats, &skills, &str, &kills, &deaths, &deleted)
		if err != nil || grow != 0 || stats != want[0] || skills != want[1] || str != 50 {
			t.Fatalf("legacy mode %d: grow=%d stat=%d skill=%d str=%d err=%v", i, grow, stats, skills, str, err)
		}
		if i > 0 && (kills != 4 || deaths != 6) || (deleted == 1) != (i == 3) {
			t.Fatalf("legacy mode %d changed reward stages or deletion status", i)
		}
	}
	var country, maxGrow int
	if err := p.QueryRow(`SELECT Country,MaxGrow FROM PS_GameData_UserMaxGrow`).Scan(&country, &maxGrow); err != nil || country != 1 || maxGrow != 0 {
		t.Fatalf("account faction changed: faction=%d legacy=%d err=%v", country, maxGrow, err)
	}
}
