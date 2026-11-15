package db

import "testing"

func TestKeepsPersistPackedOwnerAndCalendarWithoutOverwritingOtherKeeps(t *testing.T) {
	p := sqliteFixture(t)
	fixture := []Keep{{ID: 0x40001, Country: 0x21, ResetTime: 0xf8abcdef}, {ID: 0x10000002, Country: 3}}
	if err := SaveKeeps(fixture); err != nil {
		t.Fatal(err)
	}
	update := Keep{ID: fixture[0].ID, Country: 0x12, ResetTime: 0}
	if err := SaveKeeps([]Keep{update}); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadKeeps()
	if err != nil {
		t.Fatal(err)
	}
	if rows[update.ID] != update || rows[fixture[1].ID] != fixture[1] {
		t.Fatal(rows)
	}
	var count int
	if err = p.QueryRow(`SELECT count(*) FROM PS_GameData_Keeps WHERE KeepID=?`, update.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate keep count=%d error=%v", count, err)
	}
}

func TestCharacterMapZeroSurvivesPersistenceAndSelection(t *testing.T) {
	p := sqliteFixture(t)
	account := Account{UserUID: 1, UserID: "admin1"}
	id, err := CreateCharacter(account, NewCharacter{Slot: 3, Name: "MapZeroProbe", MapID: 1, HP: 100, MP: 100, SP: 100})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Exec(`UPDATE PS_GameData_Chars SET Map=0,PosX=155,PosY=39,PosZ=190 WHERE CharID=?`, id); err != nil {
		t.Fatal(err)
	}
	ch, err := CharacterByID(id)
	if err != nil || ch.MapID != 0 {
		t.Fatalf("world selection map=%d error=%v", ch.MapID, err)
	}
	rows, err := CharactersForAccount(1)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rows {
		if row.ID == id {
			found = true
			if row.MapID != 0 {
				t.Fatal("selection list changed map zero")
			}
		}
	}
	if !found {
		t.Fatal("fixture not in selection list")
	}
	if err := SaveCharacter(&ch); err != nil {
		t.Fatal(err)
	}
	again, err := CharacterByID(id)
	if err != nil || again.MapID != 0 {
		t.Fatal("map zero did not survive save/reload", err)
	}
}
