package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
)

func TestLegacyCleanupKeepsVerifiedPreMigrationBackup(t *testing.T) {
	pool := sqliteFixture(t)
	if _, err := pool.Exec(`
		DELETE FROM GX_SchemaMigrations WHERE Version=26;
		CREATE TABLE PS_GameData__CreatedChars(CharID INTEGER);
		INSERT INTO PS_GameData__CreatedChars VALUES(42);
		CREATE TABLE PS_Billing_Considers(Record INTEGER);
	`); err != nil {
		t.Fatal(err)
	}
	if err := Close(); err != nil {
		t.Fatal(err)
	}
	pool, err := Pool(GameData)
	if err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := pool.QueryRow(`SELECT COUNT(*) FROM sqlite_schema WHERE name IN ('PS_GameData__CreatedChars','PS_Billing_Considers')`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("retired tables present: %d, %v", remaining, err)
	}
	path := config.DatabasePath()
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".gx-before-schema-26-*", filepath.Base(path)))
	if err != nil || len(files) != 1 {
		t.Fatalf("expected one old-schema backup: %v, %v", files, err)
	}
	archive, err := sql.Open("sqlite", files[0])
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	var char, version int
	if err := archive.QueryRow("SELECT CharID FROM PS_GameData__CreatedChars").Scan(&char); err != nil || char != 42 {
		t.Fatalf("archived record=%d, %v", char, err)
	}
	if err := archive.QueryRow("SELECT MAX(Version) FROM GX_SchemaMigrations").Scan(&version); err != nil || version != 25 {
		t.Fatalf("archive schema=%d, %v", version, err)
	}
	if err := Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Pool(GameData); err != nil {
		t.Fatal(err)
	}
	files, err = filepath.Glob(filepath.Join(filepath.Dir(path), ".gx-before-schema-26-*", filepath.Base(path)))
	if err != nil || len(files) != 1 {
		t.Fatalf("restart duplicated backup: %v, %v", files, err)
	}
}
