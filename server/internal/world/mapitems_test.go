package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGroundItemClaimIsExclusiveAndFailureReleases(t *testing.T) {
	const zone = 60096
	t.Cleanup(func() { clearMapItems(zone) })
	item := MapItem{ID: nextMapItemID(), Item: db.Item{Count: 3}, DroppedAt: time.Now()}
	addMapItem(zone, item)
	var wins atomic.Int32
	var claim *itemClaim
	var lock sync.Mutex
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			if _, lease, ok := claimMapItem(zone, item.ID); ok {
				wins.Add(1)
				lock.Lock()
				claim = lease
				lock.Unlock()
			}
		})
	}
	workers.Wait()
	if wins.Load() != 1 {
		t.Fatal("duplicate ground claim", wins.Load())
	}
	claim.Abort()
	_, claim, ok := claimMapItem(zone, item.ID)
	if !ok {
		t.Fatal("failed claim consumed loot")
	}
	updated := claim.Finish(1)
	if updated.Item.Count != 1 {
		t.Fatal(updated)
	}
	_, claim, ok = claimMapItem(zone, item.ID)
	if !ok {
		t.Fatal("remaining loot unavailable")
	}
	claim.Finish(0)
	if _, ok := findMapItem(zone, item.ID); ok {
		t.Fatal("consumed loot survived")
	}
}
func TestConcurrentGoldPickupCreditsOnlyOnce(t *testing.T) {
	const zone = 60095
	t.Cleanup(func() { clearMapItems(zone) })
	sessions := []*Session{}
	for _, name := range []string{"LootAtomicA", "LootAtomicB"} {
		account := testAccount(t)
		id, err := db.CreateCharacter(account, db.NewCharacter{Slot: 0, Name: name, HP: 100})
		if err != nil {
			t.Fatal(err)
		}
		ch, err := db.CharacterByID(id)
		if err != nil {
			t.Fatal(err)
		}
		ch.MapID = zone
		ch.Money = 1000
		if err := db.SaveCharacter(&ch); err != nil {
			t.Fatal(err)
		}
		s, _ := gmWireSession(t, ch, false)
		sessions = append(sessions, s)
		t.Cleanup(func() { p, _ := db.Pool(db.GameData); p.Exec(`DELETE FROM PS_GameData_Chars WHERE CharID=?`, id) })
	}
	item := MapItem{ID: nextMapItemID(), Money: 100, DroppedAt: time.Now()}
	addMapItem(zone, item)
	w := packet.New(packet.AddItem)
	w.U32(item.ID)
	request := w.MustFinish()[2:]
	var workers sync.WaitGroup
	for _, s := range sessions {
		workers.Go(func() { s.handlePickupItem(request) })
	}
	workers.Wait()
	var total uint32
	for _, s := range sessions {
		ch, err := db.CharacterByID(s.char.ID)
		if err != nil {
			t.Fatal(err)
		}
		total += ch.Money
	}
	if total != 2100 {
		t.Fatal("gold duplicated", total)
	}
}
