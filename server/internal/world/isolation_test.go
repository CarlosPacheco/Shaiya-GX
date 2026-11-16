package world

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// Every integration test in this package must use an isolated runtime, even
// when the developer has server/.local/runtime.env and a personal database.
func TestMain(m *testing.M) {
	root, err := filepath.Abs("../..")
	if err != nil {
		panic(err)
	}
	temporary, err := os.MkdirTemp("", "gx-world-integration-")
	if err != nil {
		panic(err)
	}
	os.Setenv("RAPTURE_DB_PATH", filepath.Join(temporary, "runtime.sqlite"))
	os.Setenv("RAPTURE_DB_SEED", filepath.Join(root, "data", "rapture.sqlite"))
	os.Setenv("RAPTURE_DATA_DIR", filepath.Join(root, "data"))
	code := m.Run()
	os.RemoveAll(temporary)
	os.Exit(code)
}

var fixtureAccountID atomic.Int32

// Each fixture owns its slots instead of depending on the published test seed.
func testAccount(t *testing.T) db.Account {
	t.Helper()
	id := int32(1900000000) + fixtureAccountID.Add(1)
	name := fmt.Sprintf("t_%d", id)
	pool, err := db.Pool(db.UserData)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,'local',0,0,0,0,0,'N',0)`, id, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(`DELETE FROM PS_UserData_Users_Master WHERE UserUID=?`, id) })
	return db.Account{UserUID: id, UserID: name}
}
