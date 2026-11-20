package db

import "testing"

func TestQuestStartFailureRetainsOriginalInventory(t *testing.T) {
	pool := sqliteFixture(t)
	ch, item := economyCharacter(t)
	if _, err := pool.Exec(`CREATE TRIGGER GX_DenyQuestStart BEFORE INSERT ON PS_GameData_CharQuests BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	itemAfter := item
	itemAfter.Count--
	if err := CommitQuestStart(ch.ID, CharQuest{QuestID: 1}, []Item{item}, []Item{itemAfter}); err == nil {
		t.Fatal("failed insertion committed")
	}
	actual, err := CharacterItems(ch.ID)
	if err != nil || !sameInventory(actual, []Item{item}) {
		t.Fatalf("inventory after rollback: %v %v", actual, err)
	}
}
