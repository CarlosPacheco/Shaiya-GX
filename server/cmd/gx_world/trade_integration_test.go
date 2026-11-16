//go:build integration

package main

import (
	"encoding/binary"
	"sync"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func TestTradeDurableBeforeAckAndDisconnectAgainstSQLite(t *testing.T) {
	setupWorld(t)
	idA, uidA := newProtocolCharacter(t, 0)
	idB, uidB := newProtocolCharacter(t, 0)
	item := db.Item{Bag: 1, Slot: 23, Type: 25, TypeID: 4, Count: 10}
	if err := db.AddCharacterItem(idA, &item); err != nil {
		t.Fatal(err)
	}
	a, seenA := selectProtocolCharacter(t, idA, uidA)
	b, seenB := selectProtocolCharacter(t, idB, uidB)
	le := func(v uint32) []byte { return binary.LittleEndian.AppendUint32(nil, v) }
	request := func() {
		a.write(msg(packet.ExchangeRequest, le(idB)...))
		b.until(packet.ExchangeRequest, seenB)
		b.write(msg(packet.ExchangeAnswer, 0))
		a.until(packet.ExchangeStart, seenA)
		b.until(packet.ExchangeStart, seenB)
	}
	request()
	add := packet.New(packet.ExchangeAdd)
	add.U8(1)
	add.U8(23)
	add.Count(7)
	add.U8(0)
	a.write(add.MustFinish()[2:])
	a.until(packet.ExchangeAdd, seenA)
	b.until(packet.ExchangeAddDest, seenB)
	a.write(msg(packet.ExchangeAddMoney, le(100)...))
	a.until(packet.ExchangeAddMoney, seenA)
	b.until(packet.ExchangeAddMoney, seenB)
	a.write(msg(packet.ExchangeReady, 0))
	b.write(msg(packet.ExchangeReady, 0))
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a.write(msg(packet.ExchangeConfirm, 1)) }()
	go func() { defer wg.Done(); b.write(msg(packet.ExchangeConfirm, 1)) }()
	wg.Wait()
	if result := a.until(packet.ExchangeEnd, seenA); result[2] != 0 {
		t.Fatalf("exchange failed: %x", result)
	}
	// The receiver need not read a refresh/tick for either row to be durable.
	check := func() {
		t.Helper()
		for _, side := range []struct {
			id    uint32
			money uint32
			count uint16
		}{{idA, 900, 3}, {idB, 1100, 7}} {
			ch, err := db.CharacterByID(side.id)
			if err != nil || ch.Money != side.money {
				t.Fatalf("stored balance id=%d got=%d want=%d err=%v", side.id, ch.Money, side.money, err)
			}
			items, err := db.CharacterItems(side.id)
			if err != nil {
				t.Fatal(err)
			}
			var count uint16
			for _, it := range items {
				if it.Type == 25 && it.TypeID == 4 {
					count += it.Count
				}
			}
			if count != side.count {
				t.Fatalf("stored item count id=%d got=%d want=%d", side.id, count, side.count)
			}
		}
	}
	check()
	b.until(packet.ExchangeEnd, seenB)
	a.write(msg(packet.ExchangeConfirm, 1))
	b.write(msg(packet.ExchangeConfirm, 1))
	a.write(msg(packet.Ping))
	a.until(packet.Ping, seenA)
	b.write(msg(packet.Ping))
	b.until(packet.Ping, seenB)
	check()
	// Confirmed money must survive immediate recipient logout and source close.
	b.write(msg(packet.Logout))
	b.until(packet.Logout, seenB)
	b.conn.Close()
	a.conn.Close()
	time.Sleep(100 * time.Millisecond)
	a, seenA = selectProtocolCharacter(t, idA, uidA)
	b, seenB = selectProtocolCharacter(t, idB, uidB)
	check()
	request()
	a.write(msg(packet.ExchangeAddMoney, le(100)...))
	a.until(packet.ExchangeAddMoney, seenA)
	b.until(packet.ExchangeAddMoney, seenB)
	b.conn.Close()
	// Sender's session processes the disconnect cancellation; no debit occurs.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		p := a.read(time.Second)
		if p != nil && op(p) == packet.ExchangeReady {
			check()
			t.Log("atomic trade, simultaneous/duplicate confirmations, immediate logout/relog and pending-trade disconnect verified")
			return
		}
	}
	t.Fatal("disconnect did not cancel pending exchange")
}
