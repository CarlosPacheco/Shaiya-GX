package db

import (
	"strings"
	"testing"
)

func TestCharacterCreationInventoryRollbackAndSlotGuard(t *testing.T) {
	p := sqliteFixture(t)
	if _, err := p.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(990992,'create_atomic','local',0,0,0,0,0,'N',0)`); err != nil {
		t.Fatal(err)
	}
	account := Account{UserUID: 990992, UserID: "create_atomic"}
	character := NewCharacter{Slot: 0, Name: "CreateAtomic", HP: 100, MP: 100, SP: 100, MapID: 1}
	items := []Item{{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1}, {Bag: 1, Slot: 1, Type: 1, TypeID: 1, Count: 1}}
	if _, err := p.Exec(`CREATE TRIGGER GX_FailStarter BEFORE INSERT ON PS_GameData_CharItems WHEN NEW.Slot=1 BEGIN SELECT RAISE(ABORT,'injected starter failure'); END`); err != nil {
		t.Fatal(err)
	}
	if id, err := CreateCharacterWithItems(account, character, items); err == nil || id != 0 {
		t.Fatal("creation should roll back", id, err)
	}
	var count int
	if err := p.QueryRow(`SELECT COUNT(*) FROM PS_GameData_Chars WHERE UserUID=990992`).Scan(&count); err != nil || count != 0 {
		t.Fatal("partial character survived", count, err)
	}
	if err := p.QueryRow(`SELECT COUNT(*) FROM PS_GameData_CharItems WHERE CharID NOT IN (SELECT CharID FROM PS_GameData_Chars)`).Scan(&count); err != nil || count != 0 {
		t.Fatal("orphan starter items", count, err)
	}
	if _, err := p.Exec(`DROP TRIGGER GX_FailStarter`); err != nil {
		t.Fatal(err)
	}
	id, err := CreateCharacterWithItems(account, character, items)
	if err != nil {
		t.Fatal(err)
	}
	got, err := CharacterItems(id)
	if err != nil || len(got) != 2 {
		t.Fatal("initial inventory", got, err)
	}
	if ok, err := DeleteCharacter(uint32(account.UserUID), id); err != nil || !ok {
		t.Fatal(ok, err)
	}
	character.Name = "Replacement"
	if _, err := CreateCharacterWithItems(account, character, nil); err != nil {
		t.Fatal(err)
	}
	if ok, err := RestoreCharacter(uint32(account.UserUID), id); err != nil || ok {
		t.Fatal("restore occupied another character slot", ok, err)
	}
	character.Name = "SlotDuplicate"
	if id, err := CreateCharacterWithItems(account, character, nil); err == nil || id != 0 {
		t.Fatal("occupied slot accepted", id, err)
	}
}
func TestLoginUsesUserIDIndex(t *testing.T) {
	p := sqliteFixture(t)
	var id, parent, unused int
	var detail string
	err := p.QueryRow(`EXPLAIN QUERY PLAN SELECT UserUID FROM PS_UserData_Users_Master WHERE UserID=?`, "probe").Scan(&id, &parent, &unused, &detail)
	if err != nil || !strings.Contains(detail, "GX_Users_UserID") || !strings.Contains(detail, "SEARCH") {
		t.Fatal("login table scan", detail, err)
	}
}
