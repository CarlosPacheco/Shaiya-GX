package app

import (
	"strings"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func trackerTestApp() *App {
	a := inputApp()
	a.questUI, _ = ui.NewQuestLog(inputTextures{})
	a.questConfirm, _ = ui.NewMessage(inputTextures{})
	a.questTrackerUI, _ = ui.NewQuestTracker(inputTextures{}, nil)
	a.questDataLoaded = true
	a.font = &ui.Font{}
	return a
}

func TestQuestTrackerFollowsConfirmedQuestState(t *testing.T) {
	a := trackerTestApp()
	a.questDefs = []sdata.Quest{
		{Name: "Patrol", EndType: 2, MobIDs: [2]uint16{12}, MobCounts: [2]uint8{2}},
		{Name: "Defense", EndType: 3, PVPKills: 3},
		{Name: "Report", EndNPCType: 4, EndNPCID: 20},
	}
	a.npcDefs[sdata.NPCKey{Type: 4, ID: 20}] = sdata.NPC{Name: "Captain"}
	for id := uint16(1); id <= 3; id++ {
		a.applyWorldPacket(proto.QuestStartPacket(0, id))
	}

	click := func(x, y int) {
		t.Helper()
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		release := a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true, Down: "W"}, 1024, 768)
		if release.Clicked || release.Down != "W" {
			t.Fatal("panel click reached the world or stopped keyboard movement")
		}
	}
	a.worldControls(&platform.Input{Chars: []rune{'u'}}, 1024, 768)
	jx, jy := a.questUI.Origin(1024, 768)
	if a.questTracker.open {
		t.Fatal("opening the journal alone opened the tracker")
	}
	click(jx+55, jy+70)
	if !a.questTracker.open || !a.questTracker.expanded[1] {
		t.Fatal("clicking the journal row should open the tracker and expand that quest")
	}
	click(jx+55, jy+100)
	if a.questLog.selected != 2 || a.questTracker.expanded[2] {
		t.Fatal("journal selection changed an already open tracker's expansion")
	}
	a.worldControls(&platform.Input{Chars: []rune{'u'}}, 1024, 768)
	if a.questLog.open || !a.questTracker.open {
		t.Fatal("closing the journal also closed the tracker")
	}
	x, y := a.questTrackerUI.Origin(1024, 768)
	click(x+70, y+34+2*23+10)
	click(x+70, y+34+4*23+10)
	a.refreshQuestTracker()
	entries := a.questTracker.entries
	if len(entries) != 3 || entries[0].Detail != "Creature 12: 0 / 2" ||
		entries[1].Detail != "Defeat players: 0 / 3" || entries[2].Detail != "Speak with Captain" {
		t.Fatalf("expanded entries = %+v", entries)
	}

	counter := proto.NewWriter(proto.QuestCounter)
	counter.U16(1)
	counter.U8(0)
	counter.U8(2)
	a.applyWorldPacket(counter.Bytes())
	a.refreshQuestTracker()
	if entry := a.questTracker.entries[0]; entry.Detail != "Creature 12: 2 / 2" || !entry.Ready {
		t.Fatalf("confirmed completion count did not update the tracker: %+v", entry)
	}

	end := func(id uint16, success uint8) {
		p := proto.NewWriter(proto.QuestEnd)
		p.U32(0)
		p.U16(id)
		p.U8(success)
		p.Raw(make([]byte, 13))
		a.applyWorldPacket(p.Bytes())
	}
	end(1, 1)
	end(2, 0)
	a.refreshQuestTracker()
	if len(a.questTracker.entries) != 1 || a.questTracker.entries[0].ID != 3 {
		t.Fatalf("finished or abandoned quests remained visible: %+v", a.questTracker.entries)
	}
	a.applyWorldPacket(proto.QuestStartPacket(0, 1))
	a.refreshQuestTracker()
	if a.questTracker.entries[0].Expanded {
		t.Fatal("restarted quest inherited its previous expansion")
	}
	click(x+ui.QuestTrackerWidth-15, y+14)
	if a.questTracker.open {
		t.Fatal("close button left tracker open")
	}
}

func TestQuestTrackerUsesBagContentsAndShowsTime(t *testing.T) {
	a := trackerTestApp()
	a.questDefs = []sdata.Quest{{
		Name: "Supplies", EndType: 1, MaxMinutes: 2,
		FarmItems: [3]sdata.QuestItem{{Type: 27, ID: 1, Count: 2}},
	}}
	a.applyWorldPacket(proto.QuestStartPacket(0, 1))
	p := proto.NewWriter(proto.CharacterItems)
	p.U8(1)
	p.Raw([]byte{2, 4, 27, 1})
	p.U16(0)
	p.Raw(make([]byte, 6))
	p.Count(2)
	p.Raw(make([]byte, 21))
	a.applyWorldPacket(p.Bytes())
	a.showQuestTracker(1)
	entry := a.questTracker.entries[0]
	if !entry.Ready || !strings.Contains(entry.Detail, "remaining") || !strings.Contains(entry.Detail, "2 / 2 (in bags)") {
		t.Fatalf("timed item quest had incorrect bag or timer display: %+v", entry)
	}

	remove := proto.NewWriter(proto.RemoveItem)
	remove.Raw([]byte{2, 4, 27, 1})
	remove.Count(0)
	a.applyWorldPacket(remove.Bytes())
	a.refreshQuestTracker()
	if entry = a.questTracker.entries[0]; entry.Ready || !strings.Contains(entry.Detail, "0 / 2 (in bags)") {
		t.Fatalf("removing a quest item did not update the tracker: %+v", entry)
	}
	a.questStartedAt[1] = time.Now().Add(-3 * time.Minute)
	a.questTracker.refreshAt = time.Now().Add(-time.Second)
	a.refreshQuestTracker()
	if entry = a.questTracker.entries[0]; !strings.Contains(entry.Detail, "Time limit reached; awaiting server.") {
		t.Fatalf("timer expiry did not refresh while the tracker was open: %+v", entry)
	}
}

func TestQuestTrackerLayoutIsIndependent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	a := trackerTestApp()
	a.cfg.RememberLayout = true
	a.questDefs = []sdata.Quest{{Name: "Patrol"}}
	a.applyWorldPacket(proto.QuestStartPacket(0, 1))
	a.showQuestTracker(1)
	x, y := a.questTrackerUI.Origin(1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 70, MouseY: y + 12, Pressed: true, MouseDown: true}, 1024, 768)
	if a.windowDrag.position != &a.questTrackerUI.Position {
		t.Fatal("tracker header did not own the drag")
	}
	a.worldControls(&platform.Input{MouseX: x - 40, MouseY: y + 45, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: x - 40, MouseY: y + 45, Clicked: true}, 1024, 768)
	wantX, wantY := a.questTrackerUI.Origin(1024, 768)
	restored := trackerTestApp()
	restored.cfg.RememberLayout = true
	restored.loadUILayout()
	gotX, gotY := restored.questTrackerUI.Origin(1024, 768)
	if gotX != wantX || gotY != wantY || gotX != x-110 || gotY != y+33 || restored.questUI.Position.Moved {
		t.Fatalf("tracker position = %d,%d, want %d,%d independently of journal", gotX, gotY, x-110, y+33)
	}
}
