package studio

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
)

type testInstall struct {
	root, data, db string
	cash           string
}

func fixture(t *testing.T) testInstall {
	t.Helper()
	root := t.TempDir()
	data := filepath.Join(root, "data")
	dbPath := filepath.Join(root, "rapture.sqlite")
	db, err := openDB(t.Context(), dbPath, "rwc")
	must(t, err)
	defer db.Close()
	for _, statement := range []string{
		"PRAGMA journal_mode=WAL",
		"CREATE TABLE " + ProductTable + " (" + strings.Join(productColumns, ",") + ", PRIMARY KEY(ProductCode COLLATE NOCASE))",
		"CREATE TABLE PS_GameDefs_Items (ItemID INTEGER PRIMARY KEY, Type INTEGER, TypeID INTEGER)",
		"INSERT INTO PS_GameDefs_Items VALUES (1001,1,1)",
		"CREATE TABLE Unrelated (Name TEXT)",
		"INSERT INTO Unrelated VALUES ('preserve')",
		"INSERT INTO " + ProductTable + " (RowID,ProductCode,ProductName,BuyCost,ItemID1,ItemCount1) VALUES (31,'TEST','Name',5,1001,1)",
	} {
		_, err := db.ExecContext(t.Context(), statement)
		must(t, err)
	}
	item := binary.LittleEndian.AppendUint32(nil, 1)
	item = binary.LittleEndian.AppendUint32(item, 1)
	item = binary.LittleEndian.AppendUint32(item, 0)
	item = binary.LittleEndian.AppendUint32(item, 0)
	body := make([]byte, 81)
	body[0], body[1] = 1, 1
	writeFixture(t, filepath.Join(data, "Item", "Item.SData"), append(item, body...))
	raw, err := sdata.EncodeCash([]cash.Product{{
		Code: "TEST", Name: "Name", Cost: 5, Items: [cash.Slots]cash.Item{{ItemID: 1001, Count: 1}},
	}})
	must(t, err)
	cashPath := filepath.Join(data, CashFilename)
	writeFixture(t, cashPath, raw)
	return testInstall{root, data, dbPath, cashPath}
}

func writeFixture(t *testing.T, path string, bytes []byte) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(path), 0700))
	must(t, os.WriteFile(path, bytes, 0600))
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSaveCashCreatesConsistentWALBackups(t *testing.T) {
	f := fixture(t)
	ctx := t.Context()
	db, err := openDB(ctx, f.db, "rw")
	must(t, err)
	defer db.Close()
	// Keep a connection open so this newly committed row stays in the WAL.
	_, err = db.ExecContext(ctx, "INSERT INTO Unrelated VALUES ('in WAL')")
	must(t, err)
	p, err := Open(ctx, f.data, f.db)
	must(t, err)
	draft := slices.Clone(p.Products)
	draft[0].Cost = 42
	draft[0].Items[23] = cash.Item{ItemID: 1001, Count: 2}
	backup, err := Save(ctx, f.root, p, draft)
	must(t, err)
	after, err := Open(ctx, f.data, f.db)
	must(t, err)
	if !slices.Equal(after.Products, draft) || len(Differences(after.Products, after.Server)) != 0 {
		t.Fatalf("catalogs disagree after save: %+v", Differences(after.Products, after.Server))
	}
	old, err := Open(ctx, backup, filepath.Join(backup, "rapture.sqlite"))
	must(t, err)
	if !slices.Equal(old.Products, p.Products) || old.Server["TEST"].Cost != 5 {
		t.Fatal("backup does not contain the previous catalog")
	}
	snapshot, err := openDB(ctx, old.DBPath, "ro")
	must(t, err)
	defer snapshot.Close()
	var count int
	must(t, snapshot.QueryRowContext(ctx, "SELECT count(*) FROM Unrelated WHERE Name = 'in WAL'").Scan(&count))
	if count != 1 {
		t.Fatal("committed WAL state is missing from the SQLite backup")
	}
	var rowID int
	must(t, db.QueryRowContext(ctx, "SELECT RowID FROM "+ProductTable).Scan(&rowID))
	if rowID != 31 {
		t.Fatal("saving replaced metadata on an existing SQL row")
	}
	pending, err := Pending(f.root)
	must(t, err)
	if pending {
		t.Fatal("a completed save retained a pending journal")
	}
	_, err = Save(ctx, f.root, p, draft)
	if err == nil || !strings.Contains(err.Error(), "changed on disk") {
		t.Fatalf("stale draft was accepted: %v", err)
	}
}

func TestRecoverInterruptBetweenFileAndCommit(t *testing.T) {
	for _, tc := range []struct {
		title                      string
		dbCost, fileCost, wantCost uint32
		conflict                   bool
	}{
		{title: "rollback after file replacement", dbCost: 5, fileCost: 42, wantCost: 5},
		{title: "finish committed SQL", dbCost: 42, fileCost: 5, wantCost: 42},
		{title: "keep external edit for review", dbCost: 43, fileCost: 42, wantCost: 42, conflict: true},
	} {
		t.Run(tc.title, func(t *testing.T) {
			f := fixture(t)
			ctx := t.Context()
			p, err := Open(ctx, f.data, f.db)
			must(t, err)
			next := slices.Clone(p.Products)
			next[0].Cost = 42
			encoded, err := sdata.EncodeCash(next)
			must(t, err)
			db, err := openDB(ctx, f.db, "rw")
			must(t, err)
			defer db.Close()
			newHash, err := committedCost(ctx, db, 42)
			must(t, err)
			_, err = committedCost(ctx, db, tc.dbCost)
			must(t, err)

			backup := filepath.Join(f.root, "backups", "interrupted")
			writeFixture(t, filepath.Join(backup, CashFilename), p.cashRaw)
			writeFixture(t, filepath.Join(backup, "next-"+CashFilename), encoded)
			if tc.fileCost == 42 {
				writeFixture(t, f.cash, encoded)
			}
			record := journal{Backup: filepath.Base(backup), CashPath: f.cash, DBPath: f.db,
				CashBefore: p.fileHash, CashAfter: hash(encoded), SQLBefore: p.dbHash, SQLAfter: newHash}
			raw, err := json.Marshal(record)
			must(t, err)
			writeFixture(t, filepath.Join(f.root, pendingName), raw)
			_, err = Recover(ctx, f.root)
			if tc.conflict {
				if err == nil || !strings.Contains(err.Error(), "unexpected state") {
					t.Fatalf("expected recovery to refuse external changes, got %v", err)
				}
			} else {
				must(t, err)
			}
			pending, err := Pending(f.root)
			must(t, err)
			if pending != tc.conflict {
				t.Fatalf("pending = %t, wanted %t", pending, tc.conflict)
			}
			got, err := Open(ctx, f.data, f.db)
			must(t, err)
			if got.Products[0].Cost != tc.wantCost || got.Server["TEST"].Cost != tc.dbCost {
				t.Fatalf("wrong recovered costs: file %d, SQLite %d", got.Products[0].Cost, got.Server["TEST"].Cost)
			}
		})
	}
}

func committedCost(ctx context.Context, db *sql.DB, cost uint32) (string, error) {
	if _, err := db.ExecContext(ctx, "UPDATE "+ProductTable+" SET BuyCost = ?", cost); err != nil {
		return "", err
	}
	return stateHash(ctx, db, Cash)
}

func TestBackupFailureLeavesOriginals(t *testing.T) {
	f := fixture(t)
	p, err := Open(t.Context(), f.data, f.db)
	must(t, err)
	writeFixture(t, filepath.Join(f.root, "backups"), []byte("blocked directory"))
	draft := slices.Clone(p.Products)
	draft[0].Cost++
	_, err = Save(t.Context(), f.root, p, draft)
	if err == nil {
		t.Fatal("saved without both backups")
	}
	reopened, err := Open(t.Context(), f.data, f.db)
	must(t, err)
	if reopened.fileHash != p.fileHash || reopened.dbHash != p.dbHash {
		t.Fatal("backup failure modified an original")
	}
	pending, err := Pending(f.root)
	if pending || err != nil {
		t.Fatal(pending, err)
	}
}

func TestCashPreservesUntouchedLegacyWireText(t *testing.T) {
	f := fixture(t)
	raw, err := os.ReadFile(f.cash)
	must(t, err)
	start := 4 + cashBodySize
	end := start + 4 + len("Name")
	legacy := []byte{0xe9, 0, 0x77}
	var original []byte
	original = append(original, raw[:start]...)
	original = binary.LittleEndian.AppendUint32(original, uint32(len(legacy)))
	original = append(original, legacy...)
	original = append(original, raw[end:]...)
	products, err := sdata.ParseCash(original)
	must(t, err)
	products[0].Cost = 120
	encoded, err := encodeCash(original, products)
	must(t, err)
	if !bytes.Equal(encoded[start:], original[start:]) {
		t.Fatal(errors.New("an untouched wire string was re-encoded"))
	}
}
