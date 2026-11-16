package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"testing"
)

func TestInventoryOperationRollbackDoesNotPublish(t *testing.T) {
	s, add := forgeRuleSession(t)
	item := add(db.Item{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 2})
	before := *s.char
	p, _ := db.Pool(db.GameData)
	if _, err := p.Exec(`CREATE TRIGGER GX_OperationFailure BEFORE UPDATE ON PS_GameData_Chars BEGIN SELECT RAISE(ABORT,'injected failure');END`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Exec(`DROP TRIGGER GX_OperationFailure`) })
	if s.inventoryOperation(func() {
		s.char.Money--
		s.items[0].Count++
		w := packet.New(packet.Ping)
		s.send(w.MustFinish(), "must not publish")
	}) {
		t.Fatal("failed operation succeeded")
	}
	if *s.char != before || s.items[0] != item {
		t.Fatal("live state changed before commit")
	}
	actual, err := db.CharacterItems(s.char.ID)
	if err != nil || len(actual) != 1 || actual[0].Count != 2 {
		t.Fatal("partial inventory write survived", actual, err)
	}
}
