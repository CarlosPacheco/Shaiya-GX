//go:build integration

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"github.com/Spelunkern/Shaiya-GX/server/internal/telemetry"
	"github.com/Spelunkern/Shaiya-GX/server/internal/world"
)

func TestPersistentWorldLoad(t *testing.T) {
	raw := os.Getenv("GX_STORAGE_LOAD_CLIENTS")
	if raw == "" {
		t.Skip("set GX_STORAGE_LOAD_CLIENTS to run the mixed TCP storage load")
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 2 || n > 1000 {
		t.Fatal("GX_STORAGE_LOAD_CLIENTS must be 2..1000")
	}
	for name, limit := range map[string]string{
		"RAPTURE_WORLD_MAX_CONNECTIONS":         "1100",
		"RAPTURE_CONNECTIONS_PER_IP":            "1100",
		"RAPTURE_CONNECTIONS_PER_IP_PER_MINUTE": "3000",
	} {
		t.Setenv(name, limit)
	}
	setupWorld(t)
	pool, err := db.Pool(db.GameData)
	if err != nil {
		t.Fatal(err)
	}
	rounds := 8
	if raw := os.Getenv("GX_STORAGE_LOAD_ROUNDS"); raw != "" {
		rounds, err = strconv.Atoi(raw)
		if err != nil || rounds < 1 || rounds > 64 {
			t.Fatal("GX_STORAGE_LOAD_ROUNDS must be 1..64")
		}
	}
	product := &db.Product{Code: "GX_STORAGE_LOAD", BuyCost: 10, Items: []db.ProductItem{{Type: 100, TypeID: 251, Count: 1}}}
	db.MustDefs().Products[product.Code] = product
	t.Cleanup(func() { delete(db.MustDefs().Products, product.Code) })
	nav, err := mapnav.Load(filepath.Join(config.DataDir(), "geometry"), 1)
	if err != nil {
		t.Fatal(err)
	}
	g, ok := nav.PlayerGeometry()
	if !ok {
		t.Fatal("map 1 is missing physics")
	}
	// Distribute clients through the map to measure persistence while keeping
	// traffic representative of small groups in nearby world cells.
	var places []mapgeometry.Position
	for z := float32(64); z < float32(g.MapSize)-64; z += 125 {
		for x := float32(64); x < float32(g.MapSize)-64; x += 125 {
			if !world.IsMapWalkable(1, x, z) {
				continue
			}
			if p, valid := g.Entry(mapgeometry.Position{x, g.Height(x, z) + mapgeometry.FootSkin, z}); valid && !g.DeepWater(p) {
				places = append(places, p)
			}
		}
	}
	if len(places) < 25 {
		t.Fatal("not enough valid spawn locations")
	}
	type player struct {
		uid, id uint32
		wire    *client
		seen    map[packet.Opcode]int
		times   [][]time.Duration
	}
	players := make([]player, n)
	for i := range players {
		p := &players[i]
		p.uid = uint32(1100000 + i)
		user := fmt.Sprintf("gxload_%d", i)
		if _, err := pool.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,'fixture',0,0,0,0,0,'N',1000)`, p.uid, user); err != nil {
			t.Fatal(err)
		}
		if err := db.SaveCountry(int32(p.uid), 0); err != nil {
			t.Fatal(err)
		}
		at := places[i%len(places)]
		p.id, err = db.CreateCharacter(db.Account{UserUID: int32(p.uid), UserID: user}, db.NewCharacter{Slot: 0, Name: fmt.Sprintf("GXStorage%04d", i), MapID: 1, PosX: at[0], PosY: at[1], PosZ: at[2], HP: 1000, MP: 100, SP: 100})
		if err != nil {
			t.Fatal(err)
		}
		if err := db.AddCharacterItem(p.id, &db.Item{Bag: 1, Slot: 22, Type: 25, TypeID: 4, Count: 1}); err != nil {
			t.Fatal(err)
		}
		p.times = make([][]time.Duration, 4)
	}
	selectPlayer := func(p *player) {
		p.wire = connectWorld(t, p.uid)
		p.seen = make(map[packet.Opcode]int)
		p.wire.until(packet.AccountFaction, p.seen)
		p.wire.write(msg(packet.SelectCharacter, binary.LittleEndian.AppendUint32(nil, p.id)...))
		if reply := p.wire.until(packet.SelectCharacter, p.seen); reply[2] != 0 {
			t.Errorf("character load rejected uid=%d", p.uid)
			return
		}
		p.wire.until(packet.CharacterSkillBar, p.seen)
		p.wire.expanded = true
		p.wire.write(msg(packet.CharacterEnteredMap))
		p.wire.until(packet.CharacterMove, p.seen)
	}
	run := func(batch int, fn func(*player)) {
		for start := 0; start < n; start += batch {
			var wg sync.WaitGroup
			for i := start; i < min(start+batch, n); i++ {
				p := &players[i]
				wg.Go(func() { fn(p) })
			}
			wg.Wait()
			if t.Failed() {
				t.FailNow()
			}
		}
	}
	failures := telemetry.NumberMetric("gx_db_transaction_failures_total")
	overflows := telemetry.NumberMetric("gx_network_output_overflows_total")
	commits := telemetry.NumberMetric("gx_db_group_commits_total")
	operations := telemetry.NumberMetric("gx_db_group_operations_total")
	initialFailures, initialOverflows := failures.Value(), overflows.Value()
	initialCommits, initialOperations := commits.Value(), operations.Value()
	start := time.Now()
	run(32, selectPlayer)
	t.Logf("%d simultaneous World TCP sessions entered in %s across %d terrain points", n, time.Since(start), len(places))
	start = time.Now()
	run(n, func(p *player) {
		for r := range rounds {
			code := make([]byte, 21)
			copy(code, product.Code)
			begin := time.Now()
			p.wire.write(msg(packet.PointBuy, code...))
			if reply := p.wire.until(packet.PointBuy, p.seen); len(reply) < 7 || reply[2] != 0 {
				t.Errorf("uid %d purchase failed: %x", p.uid, reply)
				return
			}
			p.times[0] = append(p.times[0], time.Since(begin))
			begin = time.Now()
			dest := uint8(23 - r%2)
			p.wire.write(msg(packet.ItemBagToBag, 1, 45-dest, 1, dest))
			items, e := proto.DecodeItemMove(p.wire.until(packet.ItemBagToBag, p.seen))
			if e != nil || items[1].Type != 25 || items[1].Slot != dest {
				t.Errorf("uid %d inventory move failed: %+v %v", p.uid, items, e)
				return
			}
			p.times[1] = append(p.times[1], time.Since(begin))
			begin = time.Now()
			slots := make([]proto.QuickSlot, 0, 50)
			for i := range 50 {
				slots = append(slots, proto.QuickSlot{Bar: uint8(i / 10), Slot: uint8(i % 10), Bag: 1, Number: uint16((i + r) % 24)})
			}
			body, e := proto.SkillBarPacket(slots)
			if e != nil {
				t.Error(e)
				return
			}
			p.wire.write(body)
			p.wire.write(msg(packet.Ping))
			p.wire.until(packet.Ping, p.seen)
			p.times[2] = append(p.times[2], time.Since(begin))
		}
	})
	t.Logf("%d mixed acknowledged operations in %s", n*rounds*3, time.Since(start))
	run(n, func(p *player) {
		begin := time.Now()
		p.wire.write(msg(packet.Logout))
		p.wire.until(packet.Logout, p.seen)
		p.times[3] = append(p.times[3], time.Since(begin))
		p.wire.conn.Close()
	})
	for k, label := range []string{"purchase", "inventory", "quickbar+ping", "logout"} {
		var samples []time.Duration
		for _, p := range players {
			samples = append(samples, p.times[k]...)
		}
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		t.Logf("%s samples=%d p50=%s p95=%s p99=%s max=%s", label, len(samples), samples[len(samples)/2], samples[len(samples)*95/100], samples[len(samples)*99/100], samples[len(samples)-1])
	}
	for _, p := range players {
		var spent, entries, inventory, qty, bar, last int
		expectedSlot := 22 + rounds%2
		err := pool.QueryRow(`SELECT Point,
		(SELECT COUNT(*) FROM PS_GameData_PointLog WHERE UserUID=?1 AND ProductCode=?3),
		(SELECT COUNT(*) FROM PS_GameData_CharItems WHERE CharID=?2 AND Bag=1 AND Slot=?4 AND ItemID=25004 AND Del=0),
		(SELECT COALESCE(SUM(Count),0) FROM PS_GameData_CharItems WHERE CharID=?2 AND ItemID=100251 AND Del=0),
		(SELECT COUNT(*) FROM PS_GameData_CharQuickSlots WHERE CharID=?2),
		(SELECT COALESCE(MAX(Number),-1) FROM PS_GameData_CharQuickSlots WHERE CharID=?2 AND QuickBar=0 AND QuickSlot=0)
		FROM PS_UserData_Users_Master WHERE UserUID=?1`, p.uid, p.id, product.Code, expectedSlot).Scan(&spent, &entries, &inventory, &qty, &bar, &last)
		if err != nil || spent != 1000-rounds*10 || entries != rounds || inventory != 1 || qty != rounds || bar != 50 || last != (rounds-1)%24 {
			t.Fatalf("durable state uid=%d: points=%d logs=%d inventory=%d grants=%d bar=%d last=%d err=%v", p.uid, spent, entries, inventory, qty, bar, last, err)
		}
	}
	// Confirm a fresh encrypted selection sees the persisted bar and points.
	for _, idx := range []int{0, n - 1} {
		p := &players[idx]
		deadline := time.Now().Add(5 * time.Second)
		for {
			var active bool
			if err := pool.QueryRow("SELECT EXISTS(SELECT 1 FROM GX_ActiveSessions WHERE UserUID=?)", p.uid).Scan(&active); err != nil {
				t.Fatal(err)
			}
			if !active {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("account lease for %d did not release", p.uid)
			}
			time.Sleep(5 * time.Millisecond)
		}
		selectPlayer(p)
		p.wire.write(msg(packet.CashPoint))
		reply := p.wire.until(packet.CashPoint, p.seen)
		if len(reply) < 6 || binary.LittleEndian.Uint32(reply[2:]) != uint32(1000-rounds*10) {
			t.Fatalf("balance did not survive relog for %d: %x", p.uid, reply)
		}
		p.wire.conn.Close()
	}
	db.Observe()
	t.Logf("grouped durable operations=%d commits=%d, SQL failures=%d, output overflows=%d, WAL bytes=%d",
		operations.Value()-initialOperations, commits.Value()-initialCommits,
		failures.Value()-initialFailures, overflows.Value()-initialOverflows, telemetry.NumberMetric("gx_db_wal_bytes").Value())
	if failures.Value() != initialFailures || overflows.Value() != initialOverflows {
		t.Fatal("a database transaction or output queue failed during the load")
	}
	var integrity string
	if err := pool.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity=%q: %v", integrity, err)
	}
	if err := pool.QueryRow("SELECT COUNT(*) FROM pragma_foreign_key_check").Scan(&integrity); err != nil || integrity != "0" {
		t.Fatalf("foreign key violations=%q: %v", integrity, err)
	}
}
