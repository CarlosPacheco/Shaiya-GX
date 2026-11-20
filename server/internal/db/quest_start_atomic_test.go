package db

import (
	"testing"
	"time"
)

func TestQuestStartFailureRetainsOriginalInventory(t *testing.T) {
	pool := sqliteFixture(t)
	ch, item := economyCharacter(t)
	if _, err := pool.Exec(`CREATE TRIGGER GX_DenyQuestStart BEFORE INSERT ON PS_GameData_CharQuests BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	itemAfter := item
	itemAfter.Count--
	if err := CommitQuestStart(ch.ID, CharQuest{QuestID: 1}, []Item{item}, []Item{itemAfter}, time.Minute); err == nil {
		t.Fatal("failed insertion committed")
	}
	actual, err := CharacterItems(ch.ID)
	if err != nil || !sameInventory(actual, []Item{item}) {
		t.Fatalf("inventory after rollback: %v %v", actual, err)
	}
	var terms int
	if err := pool.QueryRow("SELECT COUNT(*) FROM GX_QuestStartTerms WHERE CharID=?", ch.ID).Scan(&terms); err != nil || terms != 0 {
		t.Fatalf("failed start consumed a start interval, count=%d err=%v", terms, err)
	}
}
