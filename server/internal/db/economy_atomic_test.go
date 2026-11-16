package db

import (
	"database/sql"
	"fmt"
	"testing"
)

func economyCharacter(t *testing.T) (Character, Item) {
	t.Helper()
	account := Account{UserUID: 990996, UserID: "atomic_economy"}
	p, err := Pool(GameData)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(990996,'atomic_economy','local',0,0,0,0,0,'N',0)`); err != nil {
		t.Fatal(err)
	}
	id, err := CreateCharacter(account, NewCharacter{Slot: 0, Name: "EconomyAtomic", HP: 100})
	if err != nil {
		t.Fatal(err)
	}
	ch, err := CharacterByID(id)
	if err != nil {
		t.Fatal(err)
	}
	ch.Money = 1000
	if err = SaveCharacter(&ch); err != nil {
		t.Fatal(err)
	}
	it := Item{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 2}
	if err = AddCharacterItem(id, &it); err != nil {
		t.Fatal(err)
	}
	return ch, it
}
func TestWarehouseFailureRollsBackBothInventoriesAndRejectsReplay(t *testing.T) {
	p := sqliteFixture(t)
	ch, item := economyCharacter(t)
	stored := item
	stored.UID = NewItemUID()
	stored.Bag = 0
	change := CharacterChange{Before: ch, After: ch, ItemsBefore: []Item{item}}
	if _, err := p.Exec(`CREATE TRIGGER GX_BankFailure BEFORE INSERT ON PS_GameData_UserStoredItems BEGIN SELECT RAISE(ABORT,'injected failure');END`); err != nil {
		t.Fatal(err)
	}
	if err := CommitWarehouse(ch.AccountID, change, nil, []Item{stored}); err == nil {
		t.Fatal("failed bank insert committed")
	}
	actual, err := CharacterItems(ch.ID)
	if err != nil || !sameInventory(actual, []Item{item}) {
		t.Fatal("source item lost", actual, err)
	}
	bank, err := StoredItems(ch.AccountID)
	if err != nil || len(bank) != 0 {
		t.Fatal("partial bank insert", bank, err)
	}
	p.Exec(`DROP TRIGGER GX_BankFailure`)
	if err := CommitWarehouse(ch.AccountID, change, nil, []Item{stored}); err != nil {
		t.Fatal(err)
	}
	if err := CommitWarehouse(ch.AccountID, change, nil, []Item{stored}); err == nil {
		t.Fatal("stale deposit replayed")
	}
}
func TestGuildBankRecipientFailureDoesNotDeleteSource(t *testing.T) {
	p := sqliteFixture(t)
	ch, item := economyCharacter(t)
	stored := item
	stored.Bag = 255
	if _, err := p.Exec(`CREATE TRIGGER GX_GuildBankFailure BEFORE INSERT ON PS_GameData_GuildStoredItems BEGIN SELECT RAISE(ABORT,'injected failure');END`); err != nil {
		t.Fatal(err)
	}
	change := CharacterChange{Before: ch, After: ch, ItemsBefore: []Item{item}}
	if err := CommitGuildBank(990996, change, nil, []Item{stored}); err == nil {
		t.Fatal("failed bank insert committed")
	}
	actual, err := CharacterItems(ch.ID)
	if err != nil || !sameInventory(actual, []Item{item}) {
		t.Fatal("source item lost", actual, err)
	}
}
func TestInventoryGrantFailureRollsBackEarlierStack(t *testing.T) {
	p := sqliteFixture(t)
	ch, item := economyCharacter(t)
	after := []Item{item, item}
	after[0].Count = 3
	after[1].UID = NewItemUID()
	after[1].Slot = 1
	if _, err := p.Exec(`CREATE TRIGGER GX_GrantFailure BEFORE INSERT ON PS_GameData_CharItems WHEN NEW.Slot=1 BEGIN SELECT RAISE(ABORT,'injected failure');END`); err != nil {
		t.Fatal(err)
	}
	if err := CommitItems(ch.ID, []Item{item}, after); err == nil {
		t.Fatal("failed grant committed")
	}
	actual, err := CharacterItems(ch.ID)
	if err != nil || !sameInventory(actual, []Item{item}) {
		t.Fatal("partial grant survived", actual, err)
	}
}
func TestPasswordManagementHashesAndRevokes(t *testing.T) {
	p := sqliteFixture(t)
	secret := "a long development passphrase with spaces"
	uid, err := CreateAccount("new_account", secret, false)
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	p.QueryRow(`SELECT Pw FROM PS_UserData_Users_Master WHERE UserUID=?`, uid).Scan(&stored)
	if stored == secret || len(stored) < 60 {
		t.Fatal("plaintext credential stored")
	}
	result, err := TryGameLogin("new_account", secret, 1, "127.0.0.1")
	if err != nil || result.Status != 0 || result.CredentialVersion != 1 {
		t.Fatal(result, err)
	}
	if err := SetAccountPassword("new_account", secret+" new"); err != nil {
		t.Fatal(err)
	}
	result, err = TryGameLogin("new_account", secret+" new", 2, "127.0.0.1")
	if err != nil || result.Status != 0 || result.CredentialVersion != 2 {
		t.Fatal(result, err)
	}
	if result, err = TryGameLogin("new_account", secret, 3, "127.0.0.1"); err != nil || result.Status != -1 {
		t.Fatal("old credential accepted", result, err)
	}
	if _, err := CreateAccount("new_account", secret, true); err == nil {
		t.Fatal("existing account overwritten")
	}
	if accepted, _ := LoginDecision(LoginResult{UserUID: uid, Status: 96}); accepted == 0 {
		t.Fatal("unknown status authenticated")
	}
}

func TestDiskFullRollsBackAndRemainsRecoverable(t *testing.T) {
	p := sqliteFixture(t)
	if _, err := p.Exec(`CREATE TABLE GX_Full(Value BLOB)`); err != nil {
		t.Fatal(err)
	}
	var pages int
	if err := p.QueryRow(`PRAGMA page_count`).Scan(&pages); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Exec(fmt.Sprintf("PRAGMA max_page_count=%d", pages)); err != nil {
		t.Fatal(err)
	}
	c, cancel := ctx()
	defer cancel()
	err := withTx(c, p, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(c, `INSERT INTO GX_Full VALUES(zeroblob(2097152))`)
		return err
	})
	if !StorageUnavailable(err) {
		t.Fatal("disk-full failure not classified", err)
	}
	var count int
	if err := p.QueryRow(`SELECT COUNT(*) FROM GX_Full`).Scan(&count); err != nil || count != 0 {
		t.Fatal("partial full-disk transaction", count, err)
	}
	if _, err := p.Exec(`PRAGMA max_page_count=2147483646`); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Exec(`INSERT INTO GX_Full VALUES(zeroblob(1024))`); err != nil {
		t.Fatal("database did not recover", err)
	}
	var integrity string
	if err := p.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatal(integrity, err)
	}
}
