//go:build integration

package main

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

var protocolFixtureID atomic.Uint32

// Protocol cases own their fixtures instead of depending on personal seed names,
// equipment or occupied character slots. TestMain owns the disposable database.
func newProtocolCharacter(t *testing.T, class uint8) (id, uid uint32) {
	t.Helper()
	n := protocolFixtureID.Add(1)
	uid = 999000 + n
	pool, err := db.Pool(db.GameData)
	if err != nil {
		t.Fatal(err)
	}
	user := fmt.Sprintf("protocol_%d", n)
	if _, err = pool.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,'local',0,0,0,0,0,'N',1000)`, uid, user); err != nil {
		t.Fatal(err)
	}
	if err = db.SaveCountry(int32(uid), 0); err != nil {
		t.Fatal(err)
	}
	id, err = db.CreateCharacter(db.Account{UserUID: int32(uid), UserID: user}, db.NewCharacter{Slot: 0, Name: fmt.Sprintf("GXProtocol%d", n), Class: class, MapID: 1, PosX: 578, PosY: 78, PosZ: 1762, HP: 1000, MP: 1000, SP: 1000})
	if err != nil {
		t.Fatal(err)
	}
	ch, err := db.CharacterDetail(uid, id)
	if err != nil {
		t.Fatal(err)
	}
	ch.Level, ch.Money, ch.Str, ch.Dex = 40, 1000, 1000, 1000
	if err = db.SaveCharacter(&ch); err != nil {
		t.Fatal(err)
	}
	if class == 0 {
		weapon := db.Item{Bag: 0, Slot: 5, Type: 1, TypeID: 1, Count: 1, Quality: 1200}
		if err = db.AddCharacterItem(id, &weapon); err != nil {
			t.Fatal(err)
		}
	}
	return id, uid
}

func selectProtocolCharacter(t *testing.T, id, uid uint32) (*client, map[packet.Opcode]int) {
	t.Helper()
	wire := connect(t, uid)
	seen := map[packet.Opcode]int{}
	wire.until(packet.AccountFaction, seen)
	wire.write(msg(packet.SelectCharacter, byte(id), byte(id>>8), byte(id>>16), byte(id>>24)))
	if response := wire.until(packet.SelectCharacter, seen); response[2] != 0 {
		t.Fatal("protocol fixture selection failed")
	}
	wire.until(packet.CharacterSkillBar, seen)
	wire.expanded = true
	wire.write(msg(packet.CharacterEnteredMap))
	wire.until(packet.CharacterMove, seen)
	return wire, seen
}
