package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGroupedWriteCommitsSuccessfulNeighborsAfterRollback(t *testing.T) {
	pool := sqliteFixture(t)
	if _, err := pool.Exec("CREATE TABLE GX_GroupFixture (ID INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	rejected := errors.New("fixture refused")
	makeTask := func(id int, fail bool) groupedWrite {
		return groupedWrite{Context: context.Background(), done: make(chan error, 1), run: func(c context.Context, tx *sql.Tx) error {
			if _, err := tx.ExecContext(c, "INSERT INTO GX_GroupFixture VALUES(?)", id); err != nil {
				return err
			}
			if fail {
				return rejected
			}
			return nil
		}}
	}
	tasks := []groupedWrite{makeTask(1, false), makeTask(2, true), makeTask(3, false)}
	commitWriteGroup(pool, tasks)
	for i, task := range tasks {
		err := <-task.done
		if (i == 1 && !errors.Is(err, rejected)) || (i != 1 && err != nil) {
			t.Fatalf("operation %d: %v", i, err)
		}
	}
	var ids string
	if err := pool.QueryRow("SELECT group_concat(ID) FROM (SELECT ID FROM GX_GroupFixture ORDER BY ID)").Scan(&ids); err != nil || ids != "1,3" {
		t.Fatalf("group committed %q: %v", ids, err)
	}
}

func TestReadPoolIgnoresUncommittedWritesAndCannotMutate(t *testing.T) {
	writer := sqliteFixture(t)
	if _, err := writer.Exec("CREATE TABLE GX_ReadFixture(Value INTEGER); INSERT INTO GX_ReadFixture VALUES(1)"); err != nil {
		t.Fatal(err)
	}
	reader, err := ReadPool()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := writer.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec("UPDATE GX_ReadFixture SET Value=2"); err != nil {
		t.Fatal(err)
	}
	var value int
	if err := reader.QueryRow("SELECT Value FROM GX_ReadFixture").Scan(&value); err != nil || value != 1 {
		t.Fatalf("uncommitted value=%d err=%v", value, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := reader.QueryRow("SELECT Value FROM GX_ReadFixture").Scan(&value); err != nil || value != 2 {
		t.Fatalf("committed value=%d err=%v", value, err)
	}
	if _, err := reader.Exec("UPDATE GX_ReadFixture SET Value=3"); err == nil {
		t.Fatal("read pool accepted an UPDATE")
	}
}

func TestLegacyQuickSlotIdentityUsesIndex(t *testing.T) {
	writer := sqliteFixture(t)
	var id, parent, aux int
	var plan string
	if err := writer.QueryRow("EXPLAIN QUERY PLAN SELECT MAX(RowID) FROM PS_GameData_CharQuickSlots WHERE rowid<>?", 1).Scan(&id, &parent, &aux, &plan); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan, "GX_Identity_CharQuickSlots") {
		t.Fatalf("legacy identity lookup lost its index: %s", plan)
	}
}

func TestCharacterExitCommitsTimersAndStateTogether(t *testing.T) {
	writer := sqliteFixture(t)
	ch, _ := economyCharacter(t)
	now := ch
	now.Money = 1200
	buffs := []AppliedBuff{{SkillID: 12, Level: 1, LeftSeconds: 42}}
	if _, err := writer.Exec(`CREATE TRIGGER GX_ExitFailure BEFORE INSERT ON PS_GameData_CharApplySkills BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	if err := SaveCharacterExit(now, buffs, nil, nil); err == nil {
		t.Fatal("failed buff insert acknowledged")
	}
	loaded, err := CharacterByID(ch.ID)
	if err != nil || loaded.Money != ch.Money {
		t.Fatalf("character state escaped rollback: money=%d err=%v", loaded.Money, err)
	}
	if _, err := writer.Exec("DROP TRIGGER GX_ExitFailure"); err != nil {
		t.Fatal(err)
	}
	if err := SaveCharacterExit(now, buffs, nil, nil); err != nil {
		t.Fatal(err)
	}
	loaded, err = CharacterByID(ch.ID)
	if err != nil || loaded.Money != now.Money {
		t.Fatalf("character state missing: money=%d err=%v", loaded.Money, err)
	}
	stored, err := AppliedBuffs(ch.ID)
	if err != nil || len(stored) != 1 || stored[0] != buffs[0] {
		t.Fatalf("buff snapshot missing: %+v %v", stored, err)
	}
}

func TestRuntimeCannotAliasSeed(t *testing.T) {
	seed := filepath.Join(t.TempDir(), "seed.sqlite")
	if err := os.WriteFile(seed, []byte("seed"), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(filepath.Dir(seed), "runtime.sqlite")
	if err := os.Link(seed, alias); err != nil {
		t.Skip(err)
	}
	t.Setenv("RAPTURE_DB_SEED", seed)
	t.Setenv("RAPTURE_DB_PATH", alias)
	if err := Initialize(); err == nil {
		t.Fatal("seed hard link accepted as runtime")
	}
}
