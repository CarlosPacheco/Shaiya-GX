//go:build integration

package world

import (
	"fmt"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestQuestItemUseOpensOnlyMatchingType4Start(t *testing.T) {
	s, _ := forgeRuleSession(t)
	s.char.Level, s.char.CurrentHP = 1, 100
	original := questDefs
	t.Cleanup(func() { questDefs = original })
	questDefs = []questDef{
		{ID: 1, StartType: 4, StartItemType: 29, StartItemID: 7, MaxLevel: 100, Jobs: [6]uint8{1, 1, 1, 1, 1, 1}, Faction: 6},
		{ID: 2, StartType: 4, StartItemType: 27, StartItemID: 7, MaxLevel: 100, Jobs: [6]uint8{1, 1, 1, 1, 1, 1}, Faction: 6},
	}
	defs := db.MustDefs().Items
	for _, typ := range []uint8{27, 29} {
		key := db.ItemKey{Type: typ, TypeID: 7}
		defs[key] = &db.ItemDef{Type: typ, TypeID: 7}
		it := db.Item{Bag: 1, Slot: typ - 20, Type: typ, TypeID: 7, Count: 1}
		if err := db.AddCharacterItem(s.char.ID, &it); err != nil {
			t.Fatal(err)
		}
		s.items = append(s.items, it)
	}
	start := proto.QuestStartPacket(7, 1)
	s.handleQuestStart(start)
	s.handleUseItem(proto.UseItemPacket(1, 7))
	s.handleQuestStart(start)
	if s.findQuest(1) != nil || findItem(s.items, 1, 7) < 0 {
		t.Fatal("wrong item opened quest, or non-consumable was removed")
	}
	s.lastQuestItem.deadline = time.Now().Add(-time.Second)
	s.handleQuestStart(proto.QuestStartPacket(7, 2))
	if s.findQuest(2) != nil {
		t.Fatal("expired use opened quest")
	}
	s.handleUseItem(proto.UseItemPacket(1, 9))
	if findItem(s.items, 1, 9) >= 0 {
		t.Fatal("type 29 was not consumed")
	}
	s.handleQuestStart(proto.QuestStartPacket(8, 1))
	if s.findQuest(1) != nil {
		t.Fatal("incorrect item context opened quest")
	}
	s.handleQuestStart(start)
	s.handleQuestStart(start)
	rows, err := db.CharacterQuests(s.char.ID)
	if err != nil || len(rows) != 1 || rows[0].QuestID != 1 || len(s.quests) != 1 {
		t.Fatalf("expected one active persisted quest after consumption, rows=%+v err=%v", rows, err)
	}
}

func TestQuestAbandonWaitsForPersistence(t *testing.T) {
	char, _ := forgeRuleSession(t)
	s, replies := gmWireSession(t, *char.char, false)
	q := &activeQuest{ID: 1, def: &questDef{ID: 1}}
	s.quests = []*activeQuest{q}
	if err := db.AddCharacterQuest(s.char.ID, db.CharQuest{QuestID: q.ID}); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Pool(db.GameData)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(fmt.Sprintf(`CREATE TRIGGER reject_quest_abandon BEFORE UPDATE OF Finish ON PS_GameData_CharQuests
		WHEN OLD.CharID=%d BEGIN SELECT RAISE(ABORT,'injected quest write failure'); END`, s.char.ID)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(`DROP TRIGGER IF EXISTS reject_quest_abandon`) })
	request := proto.QuestAbandonPacket(q.ID)
	s.handleQuestAbandon(request)
	if s.findQuest(q.ID) != q {
		t.Fatal("failed abandonment removed the active quest")
	}
	select {
	case <-replies:
		t.Fatal("failed abandonment sent a quest result")
	case <-time.After(20 * time.Millisecond):
	}
	if _, err := pool.Exec(`DROP TRIGGER reject_quest_abandon`); err != nil {
		t.Fatal(err)
	}
	s.handleQuestAbandon(request)
	select {
	case p := <-replies:
		result, err := proto.DecodeQuestEnd(p)
		if proto.Op(p) != proto.QuestEnd || result.ID != q.ID || result.Success != 0 || err != nil {
			t.Fatalf("unexpected abandonment result %+v, err=%v", result, err)
		}
	case <-time.After(time.Second):
		t.Fatal("durable abandonment did not send its result")
	}
}
