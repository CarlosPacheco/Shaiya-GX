package db

import (
	"database/sql"
	"testing"
)

func TestLogoutCannotChangeNewSessionOrCreditTwice(t *testing.T) {
	p := sqliteFixture(t)
	_, err := p.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(990991,'session_guard','local',0,0,0,0,0,'N',0)`)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{1, 2} {
		if _, err := TryGameLogin("session_guard", "local", id, "127.0.0.1"); err != nil {
			t.Fatal(err)
		}
	}
	if err := TryGameLogout(990991, 1); err != nil {
		t.Fatal(err)
	}
	var online int
	if err := p.QueryRow(`SELECT LoginStatus FROM PS_UserData_UserLoginStatus WHERE UserUID=990991`).Scan(&online); err != nil || online != 1 {
		t.Fatal("stale logout changed new login", online, err)
	}
	if _, err := p.Exec(`UPDATE PS_UserData_UserLoginStatus SET LoginTime=datetime('now','-10 minutes') WHERE UserUID=990991`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := TryGameLogout(990991, 2); err != nil {
			t.Fatal(err)
		}
	}
	var points int
	if err := p.QueryRow(`SELECT Point FROM PS_UserData_Users_Master WHERE UserUID=990991`).Scan(&points); err != nil || points != 5 {
		t.Fatal("logout credit", points, err)
	}
	var stored string
	if err := p.QueryRow(`SELECT Pw FROM PS_UserData_Users_Master WHERE UserUID=990991`).Scan(&stored); err != nil || stored == "local" {
		t.Fatal("password not migrated", err)
	}
	if r, err := TryGameLogin("session_guard", "wrong", 3, "127.0.0.1"); err != nil || r.Status != -1 {
		t.Fatal("wrong password", r, err)
	}
}

func TestTransactionPanicReleasesWriter(t *testing.T) {
	p := sqliteFixture(t)
	c, cancel := ctx()
	defer cancel()
	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected panic")
			}
		}()
		withTx(c, p, func(tx *sql.Tx) error { panic("injected failure") })
	}()
	if _, err := p.ExecContext(c, `CREATE TABLE GX_PanicRecovery(Value INTEGER)`); err != nil {
		t.Fatal("transaction retained writer", err)
	}
}
