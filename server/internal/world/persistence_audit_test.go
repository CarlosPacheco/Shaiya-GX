//go:build integration

package world

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func auditTradePair(t *testing.T) (*Session, *Session) {
	t.Helper()
	a, _ := forgeRuleSession(t)
	id, err := db.CreateCharacter(testAccount(t), db.NewCharacter{Slot: 3, Name: "AuditReceiver", HP: 100})
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
		s.player = RegisterPlayer(*s.char, s.items, false, 0, s.out, s.logger, s)
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
	a.finalizeTrade(readyAuditTrade(a, b, offer, tradeOffer{}), b.player)
	if err := p.QueryRow(query, a.char.ID, b.char.ID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("item conservation violated after injected source DELETE failure: before=%d after=%d", before, after)
	}
}

func TestAuditTradeMoneyDurableBeforePartnerMailbox(t *testing.T) {
	a, b := auditTradePair(t)
	trade := readyAuditTrade(a, b, tradeOffer{money: 100}, tradeOffer{})
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

func readyAuditTrade(a, b *Session, mine, theirs tradeOffer) *tradeSession {
	startTrade(a.player, b.player)
	trade := tradeFor(a.char.ID)
	mine.ready, mine.confirmed, theirs.ready, theirs.confirmed = true, true, true, true
	trade.offerA, trade.offerB = mine, theirs
	return trade
}

func TestTradeRollbackAtEveryWriteStage(t *testing.T) {
	for _, stage := range []string{"source_delete", "receiver_insert", "receiver_money"} {
		t.Run(stage, func(t *testing.T) {
			a, b := auditTradePair(t)
			it := db.Item{Bag: 1, Slot: 23, Type: 1, TypeID: 1, Count: 1}
			if err := db.AddCharacterItem(a.char.ID, &it); err != nil {
				t.Fatal(err)
			}
			a.items = append(a.items, it)
			beforeA, beforeB := append([]db.Item(nil), a.items...), append([]db.Item(nil), b.items...)
			p, _ := db.Pool(db.GameData)
			var trigger string
			switch stage {
			case "source_delete":
				trigger = fmt.Sprintf("BEFORE DELETE ON PS_GameData_CharItems WHEN OLD.CharID=%d", a.char.ID)
			case "receiver_insert":
				trigger = fmt.Sprintf("BEFORE INSERT ON PS_GameData_CharItems WHEN NEW.CharID=%d", b.char.ID)
			case "receiver_money":
				trigger = fmt.Sprintf("BEFORE UPDATE OF Money ON PS_GameData_Chars WHEN OLD.CharID=%d", b.char.ID)
			}
			if _, err := p.Exec("CREATE TRIGGER trade_rollback " + trigger + " BEGIN SELECT RAISE(ABORT,'injected trade write failure'); END"); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { p.Exec("DROP TRIGGER trade_rollback") })
			offer := tradeOffer{money: 100}
			offer.items[0] = &offeredItem{Item: it, Count: 1}
			a.finalizeTrade(readyAuditTrade(a, b, offer, tradeOffer{}), b.player)
			if !reflect.DeepEqual(a.items, beforeA) || !reflect.DeepEqual(b.items, beforeB) || a.char.Money != 1000 || b.char.Money != 1000 {
				t.Fatal("failed trade changed live state")
			}
			for _, side := range []struct {
				s    *Session
				want []db.Item
			}{{a, beforeA}, {b, beforeB}} {
				items, err := db.CharacterItems(side.s.char.ID)
				if err != nil {
					t.Fatal(err)
				}
				if !sameAuditInventory(items, side.want) {
					t.Fatalf("failed trade changed stored inventory: got %+v want %+v", items, side.want)
				}
				ch, err := db.CharacterByID(side.s.char.ID)
				if err != nil || ch.Money != 1000 {
					t.Fatalf("failed trade changed stored money: %d %v", ch.Money, err)
				}
			}
		})
	}
}

func sameAuditInventory(a, b []db.Item) bool {
	if len(a) != len(b) {
		return false
	}
	for _, it := range a {
		i := findItemByUID(b, it.UID)
		if i < 0 || it.Count != b[i].Count || it.Bag != b[i].Bag || it.Slot != b[i].Slot {
			return false
		}
	}
	return true
}

func TestTradeDuplicateConfirmationAndImmediateSave(t *testing.T) {
	a, b := auditTradePair(t)
	trade := readyAuditTrade(a, b, tradeOffer{money: 100}, tradeOffer{money: 30})
	a.finalizeTrade(trade, b.player)
	// Old confirmations and a late retry cannot charge again.
	a.finalizeTrade(trade, b.player)
	b.finalizeTrade(trade, a.player)
	b.savePosition() // The original stale-owner save would lose the durable credit.
	a.savePosition()
	for _, side := range []struct {
		s    *Session
		want uint32
	}{{a, 930}, {b, 1070}} {
		ch, err := db.CharacterByID(side.s.char.ID)
		if err != nil || ch.Money != side.want || side.s.char.Money != side.want {
			t.Fatalf("replayed/stale trade balance live=%d stored=%d want=%d err=%v", side.s.char.Money, ch.Money, side.want, err)
		}
	}
	if a.player.TakeTradeRefresh() != nil || b.player.TakeTradeRefresh() != nil {
		t.Fatal("trade still depends on delayed balance refresh")
	}
}

func TestTradeBusyPartnerRetryAndDisconnect(t *testing.T) {
	a, b := auditTradePair(t)
	trade := readyAuditTrade(a, b, tradeOffer{money: 100}, tradeOffer{})
	b.stateMu.Lock()
	a.finalizeTrade(trade, b.player)
	b.stateMu.Unlock()
	if tradeFor(a.char.ID) != trade || a.char.Money != 1000 {
		t.Fatal("busy partner consumed trade or blocked")
	}
	if pending := a.player.takeTradeCommit(); pending != trade {
		t.Fatal("busy trade not queued for coordinator retry")
	}
	a.finalizeTrade(trade, b.player)
	if a.char.Money != 900 || b.char.Money != 1100 {
		t.Fatal("retry did not commit exactly once")
	}
	trade = readyAuditTrade(a, b, tradeOffer{money: 100}, tradeOffer{})
	// Canceling on disconnect invalidates a previously captured commit request.
	b.player.Unregister()
	a.finalizeTrade(trade, b.player)
	if a.char.Money != 900 || b.char.Money != 1100 {
		t.Fatal("disconnected trade still committed")
	}
}

func TestTradePartialStackAndFullReceiver(t *testing.T) {
	a, b := auditTradePair(t)
	db.MustDefs().Items[db.ItemKey{Type: 25, TypeID: 4}] = &db.ItemDef{Type: 25, TypeID: 4, Count: 255}
	source := db.Item{Bag: 1, Slot: 23, Type: 25, TypeID: 4, Count: 10}
	dest := db.Item{Bag: 1, Slot: 23, Type: 25, TypeID: 4, Count: 250}
	for _, side := range []struct {
		s  *Session
		it *db.Item
	}{{a, &source}, {b, &dest}} {
		if err := db.AddCharacterItem(side.s.char.ID, side.it); err != nil {
			t.Fatal(err)
		}
		side.s.items = append(side.s.items, *side.it)
	}
	offer := tradeOffer{}
	offer.items[0] = &offeredItem{Item: source, Count: 7}
	a.finalizeTrade(readyAuditTrade(a, b, offer, tradeOffer{}), b.player)
	if i := findItemByUID(a.items, source.UID); i < 0 || a.items[i].Count != 3 {
		t.Fatal("partial source count lost")
	}
	var total int
	for _, it := range b.items {
		if it.Type == 25 && it.TypeID == 4 {
			total += int(it.Count)
			if it.Count > 255 {
				t.Fatal("oversized stack")
			}
		}
	}
	if total != 257 {
		t.Fatalf("receiver total %d want 257", total)
	}
	for _, position := range freeInventorySlots(b.items) {
		it := db.Item{Bag: position[0], Slot: position[1], Type: 1, TypeID: 1, Count: 1}
		if err := db.AddCharacterItem(b.char.ID, &it); err != nil {
			t.Fatal(err)
		}
		b.items = append(b.items, it)
	}
	// Filling remaining stack space still leaves no room for this equipment.
	item := db.Item{Bag: 1, Slot: 22, Type: 1, TypeID: 1, Count: 1}
	if err := db.AddCharacterItem(a.char.ID, &item); err != nil {
		t.Fatal(err)
	}
	a.items = append(a.items, item)
	offer = tradeOffer{}
	offer.items[0] = &offeredItem{Item: item, Count: 1}
	a.finalizeTrade(readyAuditTrade(a, b, offer, tradeOffer{}), b.player)
	if findItemByUID(a.items, item.UID) < 0 {
		t.Fatal("full receiver consumed source")
	}
}

func TestQuestRollbackIncludesRequiredItemsRewardExpAndCompletion(t *testing.T) {
	for _, stage := range []string{"required_delete", "reward_insert", "character_update"} {
		t.Run(stage, func(t *testing.T) {
			a, _ := auditTradePair(t)
			required := db.Item{Bag: 1, Slot: 23, Type: 1, TypeID: 1, Count: 1}
			if err := db.AddCharacterItem(a.char.ID, &required); err != nil {
				t.Fatal(err)
			}
			a.items = append(a.items, required)
			db.MustDefs().Exp = map[db.ExpKey]uint32{{Level: 1, Grow: 1}: 10, {Level: 2, Grow: 1}: 1000}
			q := &activeQuest{ID: 1, def: &questDef{ID: 1, EndType: 1, ResultType: 1, FarmItems: [3]questItem{{Type: 1, TypeID: 1, Count: 1}}, Results: [3]questResult{{Money: 100, Exp: 20, ItemType: 16, ItemTypeID: 1}}}}
			a.quests = []*activeQuest{q}
			if err := db.AddCharacterQuest(a.char.ID, db.CharQuest{QuestID: q.ID}); err != nil {
				t.Fatal(err)
			}
			before := *a.char
			items := append([]db.Item(nil), a.items...)
			p, _ := db.Pool(db.GameData)
			var trigger string
			switch stage {
			case "required_delete":
				trigger = fmt.Sprintf("BEFORE DELETE ON PS_GameData_CharItems WHEN OLD.CharID=%d", a.char.ID)
			case "reward_insert":
				trigger = fmt.Sprintf("BEFORE INSERT ON PS_GameData_CharItems WHEN NEW.CharID=%d", a.char.ID)
			case "character_update":
				trigger = fmt.Sprintf("BEFORE UPDATE OF Money ON PS_GameData_Chars WHEN OLD.CharID=%d", a.char.ID)
			}
			if _, err := p.Exec("CREATE TRIGGER quest_rollback " + trigger + " BEGIN SELECT RAISE(ABORT,'injected quest write failure'); END"); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				p.Exec("DROP TRIGGER quest_rollback")
				p.Exec("DELETE FROM PS_GameData_CharQuests WHERE CharID=?", a.char.ID)
			})
			a.awardQuest(q, 0, 0)
			if *a.char != before || !reflect.DeepEqual(a.items, items) || a.findQuest(q.ID) != q || len(a.finishedQuests) != 0 {
				t.Fatal("failed quest published live reward/level/completion")
			}
			saved, err := db.CharacterByID(a.char.ID)
			if err != nil {
				t.Fatal(err)
			}
			if saved.Money != before.Money || saved.Level != before.Level || saved.Exp != before.Exp {
				t.Fatal("failed quest persisted gold/level/exp")
			}
			stored, err := db.CharacterItems(a.char.ID)
			if err != nil || !sameAuditInventory(stored, items) {
				t.Fatal("failed quest persisted item changes", err)
			}
			quests, err := db.CharacterQuests(a.char.ID)
			if err != nil || len(quests) != 1 {
				t.Fatal("failed quest no longer active", quests, err)
			}
			if _, err := p.Exec("DROP TRIGGER quest_rollback"); err != nil {
				t.Fatal(err)
			}
			a.awardQuest(q, 0, 0)
			if a.char.Money != 1100 || a.char.Level != 2 || findItemByUID(a.items, required.UID) >= 0 || a.findQuest(q.ID) != nil {
				t.Fatalf("retry not committed atomically: %+v items=%+v", a.char, a.items)
			}
			gold, exp := a.char.Money, a.char.Exp
			a.awardQuest(q, 0, 0)
			if a.char.Money != gold || a.char.Exp != exp {
				t.Fatal("completed quest replayed")
			}
		})
	}
}

func TestTradeRejectsGoldOverflowWithoutLosingMoney(t *testing.T) {
	a, b := auditTradePair(t)
	b.char.Money = db.MoneyCap
	if err := db.SaveCharacter(b.char); err != nil {
		t.Fatal(err)
	}
	a.finalizeTrade(readyAuditTrade(a, b, tradeOffer{money: 1}, tradeOffer{}), b.player)
	if a.char.Money != 1000 || b.char.Money != db.MoneyCap {
		t.Fatal("gold overflow destroyed currency")
	}
}

func TestQuestFullBagAndDatabaseReplayGuard(t *testing.T) {
	a, _ := auditTradePair(t)
	for _, position := range freeInventorySlots(a.items) {
		it := db.Item{Bag: position[0], Slot: position[1], Type: 1, TypeID: 1, Count: 1}
		if err := db.AddCharacterItem(a.char.ID, &it); err != nil {
			t.Fatal(err)
		}
		a.items = append(a.items, it)
	}
	q := &activeQuest{ID: 1, def: &questDef{ID: 1, ResultType: 1, Results: [3]questResult{{Money: 100, ItemType: 16, ItemTypeID: 1}}}}
	a.quests = []*activeQuest{q}
	if err := db.AddCharacterQuest(a.char.ID, db.CharQuest{QuestID: 1}); err != nil {
		t.Fatal(err)
	}
	pool, _ := db.Pool(db.GameData)
	t.Cleanup(func() { pool.Exec("DELETE FROM PS_GameData_CharQuests WHERE CharID=?", a.char.ID) })
	a.awardQuest(q, 0, 0)
	if a.char.Money != 1000 || a.findQuest(1) != q {
		t.Fatal("full bag consumed quest or awarded gold without item")
	}
	// Free one slot; the same choice may now be retried successfully.
	i := findItem(a.items, 1, 0)
	if i < 0 {
		t.Fatal("fixture has no occupied bag slot")
	}
	if err := db.DeleteCharacterItems(a.char.ID, a.items[i].UID); err != nil {
		t.Fatal(err)
	}
	a.items = append(a.items[:i], a.items[i+1:]...)
	a.awardQuest(q, 0, 0)
	if a.char.Money != 1100 || a.findQuest(1) != nil {
		t.Fatal("full-bag retry failed")
	}
	// Even a stale restored runtime object cannot bypass durable Finish=1.
	a.quests = []*activeQuest{q}
	a.awardQuest(q, 0, 0)
	if a.char.Money != 1100 {
		t.Fatal("stale quest object replayed durable reward")
	}
}

func TestRepeatableQuestCommitsEachAcceptedInstanceOnce(t *testing.T) {
	a, _ := auditTradePair(t)
	pool, _ := db.Pool(db.GameData)
	t.Cleanup(func() { pool.Exec("DELETE FROM PS_GameData_CharQuests WHERE CharID=?", a.char.ID) })
	for instance := 0; instance < 2; instance++ {
		q := &activeQuest{ID: 1, def: &questDef{ID: 1, ResultType: 1, TickReceive: 1, Results: [3]questResult{{Money: 100}}}}
		a.quests = []*activeQuest{q}
		if err := db.AddCharacterQuest(a.char.ID, db.CharQuest{QuestID: 1}); err != nil {
			t.Fatal(err)
		}
		a.awardQuest(q, 0, 0)
		if a.char.Money != uint32(1100+100*instance) || len(a.finishedQuests) != 0 {
			t.Fatal("repeatable reward not committed")
		}
		active, err := db.CharacterQuests(a.char.ID)
		if err != nil || len(active) != 0 {
			t.Fatal("repeatable quest not removed atomically", err)
		}
		a.quests = []*activeQuest{q}
		a.awardQuest(q, 0, 0)
		if a.char.Money != uint32(1100+100*instance) {
			t.Fatal("repeatable stale hand-in replayed")
		}
	}
}
