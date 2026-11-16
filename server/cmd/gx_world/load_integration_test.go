//go:build integration

package main

import (
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// This is a bounded protocol load probe, not a maximum-player benchmark.
func TestConcurrentClientsPurchaseLogoutAndRelog(t *testing.T) {
	setupWorld(t)
	pool, err := db.Pool(db.GameData)
	if err != nil {
		t.Fatal(err)
	}
	// The approved seed intentionally has a free mall. Use a private priced
	// product so this probe verifies real charges without changing shared data.
	product := &db.Product{Code: "LOAD_PROBE", BuyCost: 10, Items: []db.ProductItem{{Type: 100, TypeID: 251, Count: 1}}}
	db.MustDefs().Products[product.Code] = product
	t.Cleanup(func() { delete(db.MustDefs().Products, product.Code) })
	const count = 16
	const purchases = 5
	type player struct {
		uid, id uint32
		wire    *client
		seen    map[packet.Opcode]int
	}
	players := make([]player, count)
	selectPlayer := func(p *player) {
		t.Helper()
		p.wire = connect(t, p.uid)
		p.seen = map[packet.Opcode]int{}
		p.wire.until(packet.AccountFaction, p.seen)
		p.wire.write(msg(packet.SelectCharacter, byte(p.id), byte(p.id>>8), byte(p.id>>16), byte(p.id>>24)))
		if reply := p.wire.until(packet.SelectCharacter, p.seen); reply[2] != 0 {
			t.Fatal("load character selection rejected")
		}
		p.wire.until(packet.CharacterSkillBar, p.seen)
		p.wire.expanded = true
		p.wire.write(msg(packet.CharacterEnteredMap))
		p.wire.until(packet.CharacterMove, p.seen)
	}
	for i := range players {
		p := &players[i]
		p.uid = uint32(998000 + i)
		user := fmt.Sprintf("load_%d", p.uid)
		if _, err := pool.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,'local',0,0,0,0,0,'N',1000)`, p.uid, user); err != nil {
			t.Fatal(err)
		}
		if err := db.SaveCountry(int32(p.uid), 0); err != nil {
			t.Fatal(err)
		}
		id, err := db.CreateCharacter(db.Account{UserUID: int32(p.uid), UserID: user}, db.NewCharacter{Slot: 0, Name: fmt.Sprintf("LoadProbe%02d", i), MapID: 1, PosX: 578, PosY: 78, PosZ: 1762, HP: 100, MP: 100, SP: 100})
		if err != nil {
			t.Fatal(err)
		}
		p.id = id
		selectPlayer(p)
	}
	// Form a real party before the load, independently of legacy seed names.
	leader, member := &players[0], &players[1]
	le := func(id uint32) []byte { return binary.LittleEndian.AppendUint32(nil, id) }
	leader.wire.write(msg(packet.PartyInvite, le(member.id)...))
	member.wire.until(packet.PartyInvite, member.seen)
	member.wire.write(msg(packet.PartyAnswer, append([]byte{0}, le(leader.id)...)...))
	leader.wire.until(packet.PartyRoster, leader.seen)
	leader.wire.until(packet.PartyJoined, leader.seen)
	if roster := member.wire.until(packet.PartyRoster, member.seen); len(roster) < 4 || roster[3] != 1 {
		t.Fatalf("two-player party roster: %x", roster)
	}
	t.Log("two disposable clients formed a party over encrypted protocol before concurrent purchases")
	var wg sync.WaitGroup
	var timingMu sync.Mutex
	var latencies []time.Duration
	start := time.Now()
	for i := range players {
		p := &players[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range purchases {
				code := make([]byte, 21)
				copy(code, "LOAD_PROBE")
				begin := time.Now()
				p.wire.write(msg(packet.PointBuy, code...))
				reply := p.wire.until(packet.PointBuy, p.seen)
				if len(reply) < 7 || reply[2] != 0 {
					t.Errorf("client %d purchase rejected: %x", p.uid, reply)
					return
				}
				timingMu.Lock()
				latencies = append(latencies, time.Since(begin))
				timingMu.Unlock()
				p.wire.write(msg(packet.Ping))
				p.wire.until(packet.Ping, p.seen)
			}
		}()
	}
	wg.Wait()
	if len(latencies) != count*purchases {
		t.Fatalf("completed %d of %d purchases", len(latencies), count*purchases)
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	t.Logf("%d simultaneous encrypted clients, %d durable purchases and pings in %s; purchase p50=%s p95=%s max=%s", count, len(latencies), time.Since(start), latencies[len(latencies)/2], latencies[len(latencies)*95/100], latencies[len(latencies)-1])
	for i := range players {
		p := &players[i]
		// Half cut the socket with one additional purchase in flight; half log out.
		// An unacknowledged purchase may commit or abort, but its charge and items
		// must agree exactly after reconnection.
		if i%2 == 0 {
			code := make([]byte, 21)
			copy(code, "LOAD_PROBE")
			p.wire.write(msg(packet.PointBuy, code...))
			p.wire.conn.Close()
		} else {
			p.wire.write(msg(packet.Logout))
			p.wire.until(packet.Logout, p.seen)
			p.wire.conn.Close()
		}
	}
	for i := range players {
		p := &players[i]
		selectPlayer(p)
		want, err := db.Points(p.uid)
		if err != nil {
			t.Fatal(err)
		}
		base := uint32(1000) - purchases*product.BuyCost
		if want != base && !(i%2 == 0 && want == base-product.BuyCost) {
			t.Fatalf("unexpected relog balance uid=%d: %d", p.uid, want)
		}
		committed := (1000 - want) / product.BuyCost
		var logged int
		if err := pool.QueryRow("SELECT COUNT(*) FROM PS_GameData_PointLog WHERE UserUID=? AND ProductCode=?", p.uid, product.Code).Scan(&logged); err != nil {
			t.Fatal(err)
		}
		if uint32(logged) != committed {
			t.Fatalf("purchase log mismatch uid=%d logs=%d charged=%d", p.uid, logged, committed)
		}
		p.wire.write(msg(packet.CashPoint))
		reply := p.wire.until(packet.CashPoint, p.seen)
		if len(reply) < 6 || binary.LittleEndian.Uint32(reply[2:]) != want {
			t.Fatalf("relog wire balance uid=%d: %x", p.uid, reply)
		}
		items, err := db.CharacterItems(p.id)
		if err != nil {
			t.Fatal(err)
		}
		for _, grant := range product.Items {
			var total uint32
			for _, item := range items {
				if item.Type == grant.Type && item.TypeID == grant.TypeID {
					total += uint32(item.Count)
				}
			}
			if wantCount := uint32(grant.Count) * committed; total != wantCount {
				t.Fatalf("relog items uid=%d type=%d/%d got=%d want=%d", p.uid, grant.Type, grant.TypeID, total, wantCount)
			}
		}
		p.wire.conn.Close()
	}
	var integrity string
	if err := pool.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity=%q %v", integrity, err)
	}
	t.Log("all 16 accounts preserved exact charges and item quantities after clean logout, in-flight purchase disconnect and encrypted relog")
}
