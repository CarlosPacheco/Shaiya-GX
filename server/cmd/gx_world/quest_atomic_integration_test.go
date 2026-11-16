//go:build integration

package main

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"github.com/Spelunkern/Shaiya-GX/server/internal/world"
)

func TestQuestWriteFailureRetryAndDurableAckAgainstSQLite(t *testing.T) {
	setupWorld(t)
	id, uid := newProtocolCharacter(t, 0)
	var endNPC world.MapNpc
	for _, npc := range world.EntitiesOnMap(1).NPCs {
		if npc.Type == 8 && npc.TypeID == 87 {
			endNPC = npc
			break
		}
	}
	if endNPC.ID == 0 {
		t.Fatal("quest 1 end NPC 8/87 missing")
	}
	ch, err := db.CharacterDetail(uid, id)
	if err != nil {
		t.Fatal(err)
	}
	ch.Level, ch.Exp, ch.Money = 1, 0, 1000
	ch.PosX, ch.PosY, ch.PosZ = endNPC.PosX, endNPC.PosY, endNPC.PosZ
	if err = db.SaveCharacter(&ch); err != nil {
		t.Fatal(err)
	}
	if err = db.AddCharacterQuest(id, db.CharQuest{QuestID: 1, Counts: [3]uint8{5, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	pool, _ := db.Pool(db.GameData)
	if _, err = pool.Exec(fmt.Sprintf("CREATE TRIGGER network_quest_failure BEFORE UPDATE OF Money ON PS_GameData_Chars WHEN OLD.CharID=%d BEGIN SELECT RAISE(ABORT,'injected character reward failure'); END", id)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec("DROP TRIGGER IF EXISTS network_quest_failure") })
	wire, seen := selectProtocolCharacter(t, id, uid)
	finish := packet.New(packet.QuestEnd)
	finish.U32(endNPC.ID)
	finish.U16(1)
	request := finish.MustFinish()[2:]
	wire.write(request)
	wire.write(msg(packet.Ping))
	wire.until(packet.Ping, seen)
	if seen[packet.QuestEnd] != 0 {
		t.Fatal("failed transaction sent quest completion")
	}
	var gold, exp int
	if err = pool.QueryRow("SELECT Money,Exp FROM PS_GameData_Chars WHERE CharID=?", id).Scan(&gold, &exp); err != nil {
		t.Fatal(err)
	}
	if gold != 1000 || exp != 0 {
		t.Fatal("failed hand-in persisted reward", gold, exp)
	}
	active, err := db.CharacterQuests(id)
	if err != nil || len(active) != 1 {
		t.Fatal("failed hand-in lost active quest", active, err)
	}
	if _, err = pool.Exec("DROP TRIGGER network_quest_failure"); err != nil {
		t.Fatal(err)
	}
	wire.write(request)
	reply := wire.until(packet.QuestEnd, seen)
	if len(reply) < 22 || reply[8] != 1 {
		t.Fatalf("quest retry rejected: %x", reply)
	}
	rewardExp, rewardGold := binary.LittleEndian.Uint32(reply[10:]), binary.LittleEndian.Uint32(reply[14:])
	check := func() {
		t.Helper()
		saved, err := db.CharacterByID(id)
		if err != nil {
			t.Fatal(err)
		}
		if saved.Money != 1000+rewardGold || saved.Exp != rewardExp {
			t.Fatalf("reward not durable at acknowledgement: gold=%d exp=%d expected=%d/%d", saved.Money, saved.Exp, 1000+rewardGold, rewardExp)
		}
		finished, err := db.FinishedQuests(id)
		if err != nil || len(finished) != 1 || finished[0].QuestID != 1 || finished[0].Success != 1 {
			t.Fatal("reward acknowledged without durable completion", finished, err)
		}
		if reply[20] != 0 {
			items, err := db.CharacterItems(id)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, it := range items {
				if it.Bag == reply[18] && it.Slot == reply[19] && it.Type == reply[20] && it.TypeID == reply[21] {
					found = true
				}
			}
			if !found {
				t.Fatal("reward item missing at acknowledgement")
			}
		}
	}
	check()
	completed := seen[packet.QuestEnd]
	wire.write(request)
	wire.write(msg(packet.Ping))
	wire.until(packet.Ping, seen)
	if seen[packet.QuestEnd] != completed {
		t.Fatal("duplicate hand-in completed twice")
	}
	check()
	wire.conn.Close()
	wire, _ = selectProtocolCharacter(t, id, uid)
	defer wire.conn.Close()
	check()
	t.Log("failed quest write sent no success; retry committed gold, exp, item and completion before acknowledgement; duplicate/relog preserved reward")
}
