//go:build integration && persistenceaudit

package world

import (
	"fmt"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func auditTradePair(t *testing.T) (*Session, *Session) {
	t.Helper()
	a, _ := forgeRuleSession(t)
	id, err := db.CreateCharacter(db.Account{UserUID: 1, UserID: "admin1"}, db.NewCharacter{Slot: 3, Name: "AuditReceiver", HP: 100})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := db.Pool(db.GameData)
	t.Cleanup(func() {
		p.Exec("DELETE FROM PS_GameData_CharItems WHERE CharID=?", id)
		p.Exec("DELETE FROM PS_GameData_Chars WHERE CharID=?", id)
	})
	b, _ := gmWireSession(t, db.Character{ID: id, Level: 1, Money: 1000, MapID: 60091, CurrentHP: 100}, false)
	a.char.Money = 1000
	for _, s := range []*Session{a, b} {
		if err := db.SaveCharacter(s.char); err != nil {
			t.Fatal(err)
		}
		s.items, err = db.CharacterItems(s.char.ID)
		if err != nil {
			t.Fatal(err)
		}
		s.player = RegisterPlayer(*s.char, s.items, false, 0, s.out, s.logger)
		t.Cleanup(s.player.Unregister)
	}
	return a, b
}

func TestAuditTradeWriteFailureConservesItems(t *testing.T) {
	a, b := auditTradePair(t)
	it := db.Item{Bag: 1, Slot: 23, Type: 1, TypeID: 1, Count: 1}
	if err := db.AddCharacterItem(a.char.ID, &it); err != nil {
		t.Fatal(err)
	}
	a.items = append(a.items, it)
	p, _ := db.Pool(db.GameData)
	trigger := fmt.Sprintf("CREATE TRIGGER audit_fail BEFORE DELETE ON PS_GameData_CharItems WHEN OLD.CharID=%d AND OLD.ItemUID=%d BEGIN SELECT RAISE(ABORT,'injected write failure'); END", a.char.ID, it.UID)
	if _, err := p.Exec(trigger); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Exec("DROP TRIGGER audit_fail") })
	var before, after int
	query := `SELECT COALESCE(SUM(Count),0) FROM PS_GameData_CharItems WHERE CharID IN (?,?) AND Type=1 AND TypeID=1 AND Del=0`
	if err := p.QueryRow(query, a.char.ID, b.char.ID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	offer := tradeOffer{}
	offer.items[0] = &offeredItem{Item: it, Count: 1}
	transferOffer(a.char.ID, b.char.ID, &offer, &a.items, &b.items)
	if err := p.QueryRow(query, a.char.ID, b.char.ID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("item conservation violated after injected source DELETE failure: before=%d after=%d", before, after)
	}
}

func TestAuditTradeMoneyDurableBeforePartnerMailbox(t *testing.T) {
	a, b := auditTradePair(t)
	trade := &tradeSession{a: a.char.ID, b: b.char.ID, offerA: tradeOffer{money: 100}}
	a.finalizeTrade(trade, b.player)
	p, _ := db.Pool(db.GameData)
	var ma, mb int
	if err := p.QueryRow("SELECT Money FROM PS_GameData_Chars WHERE CharID=?", a.char.ID).Scan(&ma); err != nil {
		t.Fatal(err)
	}
	if err := p.QueryRow("SELECT Money FROM PS_GameData_Chars WHERE CharID=?", b.char.ID).Scan(&mb); err != nil {
		t.Fatal(err)
	}
	if ma != 900 || mb != 1100 {
		t.Fatalf("successful exchange not durable before partner mailbox (process exit here loses gold): sender=%d receiver=%d total=%d", ma, mb, ma+mb)
	}
}

func TestAuditQuestFinishFailureCannotRepeatReward(t *testing.T) {
	a, _ := auditTradePair(t)
	q := &activeQuest{ID: 1, def: &questDef{ID: 1, Results: [3]questResult{{Money: 100}}}}
	a.quests = []*activeQuest{q}
	if err := db.AddCharacterQuest(a.char.ID, db.CharQuest{QuestID: q.ID}); err != nil {
		t.Fatal(err)
	}
	p, _ := db.Pool(db.GameData)
	if _, err := p.Exec(fmt.Sprintf("CREATE TRIGGER audit_finish_fail BEFORE UPDATE ON PS_GameData_CharQuests WHEN OLD.CharID=%d BEGIN SELECT RAISE(ABORT,'injected finish failure'); END", a.char.ID)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		p.Exec("DROP TRIGGER audit_finish_fail")
		p.Exec("DELETE FROM PS_GameData_CharQuests WHERE CharID=?", a.char.ID)
	})
	a.awardQuest(q, 0, 0)
	var money, finish int
	if err := p.QueryRow("SELECT Money FROM PS_GameData_Chars WHERE CharID=?", a.char.ID).Scan(&money); err != nil {
		t.Fatal(err)
	}
	if err := p.QueryRow("SELECT COALESCE(Finish,0) FROM PS_GameData_CharQuests WHERE CharID=? AND QuestID=?", a.char.ID, q.ID).Scan(&finish); err != nil {
		t.Fatal(err)
	}
	if !((money == 1000 && finish == 0) || (money == 1100 && finish == 1)) {
		t.Fatalf("reward persisted but quest still active after finish failure: money=%d finished=%d; reward/completion invariant violated", money, finish)
	}
}
