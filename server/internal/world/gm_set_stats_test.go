package world

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestGMSetSixBaseStatsPersistsAndRefreshesClient(t *testing.T) {
	id, err := db.CreateCharacter(testAccount(t), db.NewCharacter{Slot: 4, Name: "GMStatsProbe", HP: 100, MP: 100, SP: 100, MapID: 60070})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.Pool(db.GameData)
	t.Cleanup(func() { pool.Exec(`DELETE FROM PS_GameData_Chars WHERE CharID=?`, id) })
	ch, err := db.CharacterByID(id)
	if err != nil {
		t.Fatal(err)
	}
	ch.StatPoint, ch.KillLevel, ch.DeadLevel = 17, 20, 12
	ch.Kills, ch.Deaths, ch.Money = 123, 456, 1000
	s, packets := gmWireSession(t, ch, true)
	state := game.New()
	state.SelfID = id
	state.Slots[0] = proto.CharacterSlot{ID: id}
	applyUntilReply := func() {
		t.Helper()
		for {
			select {
			case p := <-packets:
				if _, err := state.Apply(p); err != nil {
					t.Fatal(err)
				}
				if proto.Op(p) == proto.GXGMState {
					return
				}
			case <-time.After(3 * time.Second):
				t.Fatal("missing stat confirmation")
			}
		}
	}
	attributes := func(c db.Character) [6]uint16 { return [6]uint16{c.Str, c.Dex, c.Rec, c.Int, c.Wis, c.Luc} }
	fields := []string{"str", "dex", "rec", "int", "wis", "luc"}
	for index, field := range fields {
		for _, tc := range []struct {
			raw  string
			want uint16
		}{{"1234", 1234}, {"32000", 32000}, {"9999999999999999999999999999999", 32000}, {"0", 0}} {
			before := attributes(*s.char)
			command := fmt.Sprintf("/SET %s %s", field, tc.raw)
			if !s.handleGMCommand(command) {
				t.Fatal("command escaped to chat")
			}
			applyUntilReply()
			saved, err := db.CharacterByID(id)
			if err != nil {
				t.Fatal(err)
			}
			expected := before
			expected[index] = tc.want
			slot := state.Slots[0]
			clientBase := [6]uint16{slot.Str, slot.Dex, slot.Rec, slot.Int, slot.Wis, slot.Luc}
			if attributes(saved) != expected || clientBase != expected {
				t.Fatalf("%s base persistence/client=%v/%v want=%v", command, attributes(saved), clientBase, expected)
			}
			effective := s.stats()
			if !state.StatsKnown || state.Stats.Str != effective.Str || state.Stats.Dex != effective.Dex || state.Stats.Rec != effective.Rec || state.Stats.Int != effective.Int || state.Stats.Wis != effective.Wis || state.Stats.Luc != effective.Luc || state.Details.MaxHP != effective.MaxHP || state.Details.MaxMP != effective.MaxMP || state.Details.MaxSP != effective.MaxSP {
				t.Fatal("derived stats/pools stale", command)
			}
			if saved.StatPoint != 17 || saved.KillLevel != 20 || saved.DeadLevel != 12 || saved.Kills != 123 || saved.Deaths != 456 || saved.Money != 1000 || state.Details.StatPoint != 17 || state.PvPRewards.KillStage != 21 || state.PvPRewards.DeathStage != 13 {
				t.Fatal("set stats altered points/counters/claims", command)
			}
		}
	}
	s.handleGMCommand("/set str 31999")
	applyUntilReply()
	s.handleSetStatPoint(proto.StatPointsPacket(proto.StatPoints{1, 0, 0, 0, 0, 0}))
	s.handleGMCommand("/set money 1000")
	applyUntilReply() // Flush normal allocation response.
	saved, err := db.CharacterByID(id)
	if err != nil || saved.Str != 32000 || saved.StatPoint != 16 || state.Slots[0].Str != 32000 || state.Details.StatPoint != 16 {
		t.Fatal("allocation to the exact cap failed", saved.Str, saved.StatPoint, err)
	}
	before := *s.char
	for _, bad := range []string{"/set str -1", "/set dex 1.5", "/set int +2", "/set rec 42 target", "/set wis 1x"} {
		s.handleGMCommand(bad)
		applyUntilReply()
		if !reflect.DeepEqual(*s.char, before) {
			t.Fatal("invalid stat command mutated character", bad)
		}
	}
	s.isAdmin = false
	s.handleGMCommand("/set luc 300")
	applyUntilReply()
	if *s.char != before {
		t.Fatal("non-GM set stats")
	}
}

func TestNormalAllocationCannotOverflowGMBaseStatCap(t *testing.T) {
	s, _ := gmWireSession(t, db.Character{ID: 990077, Str: 32000, Dex: 100, StatPoint: 10}, false)
	before := *s.char
	// One valid DEX point plus one overflowing STR point must be atomic.
	p := proto.StatPointsPacket(proto.StatPoints{1, 1, 0, 0, 0, 0})
	s.handleSetStatPoint(p)
	if *s.char != before {
		t.Fatal("over-cap request spent points or changed other stats")
	}
}

func TestStatAllocationPublishesCurrentDerivedStats(t *testing.T) {
	s, wire := gmWireSession(t, db.Character{ID: 990078, Str: 20, StatPoint: 10}, false)
	s.handleSetStatPoint(proto.StatPointsPacket(proto.StatPoints{5}))
	waitWorldPacket(t, wire, proto.SetStatPoint)
	stats, err := proto.DecodeAdditionalStats(waitWorldPacket(t, wire, proto.CharacterAdditionalStats))
	if err != nil || stats.Str != 25 || stats.MinAttack != s.stats().MinAttack {
		t.Fatalf("allocation stats = %+v, err = %v", stats, err)
	}
}
