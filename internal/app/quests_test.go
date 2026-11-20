package app

import (
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestQuestCommandsDisplayNPCLinksAndRejectInvalidActions(t *testing.T) {
	a := inputApp()
	a.questDataLoaded = true
	a.questDefs = []sdata.Quest{{Name: "First quest"}, {Name: "Second quest"}}
	a.state.Quests = map[uint16]proto.ActiveQuest{2: {ID: 2, Counters: [3]uint8{1, 2, 3}}}
	a.npcDefs[sdata.NPCKey{Type: 8, ID: 3}] = sdata.NPC{StartQuests: []uint16{1}, EndQuests: []uint16{2}}
	a.state.Entities[7] = &game.Entity{ID: 7, Kind: game.KindNpc, NpcType: 8, TypeID: 3}
	a.shop.npcID = 7
	if !a.questCommand("/quest list") {
		t.Fatal("quest command not recognized")
	}
	var output string
	for _, line := range a.state.Chat {
		output += line.Text + "\n"
	}
	for _, want := range []string{"Second quest (1/2/3)", "NPC quest: 1: First quest", "Hand in at NPC: 2: Second quest"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in %s", want, output)
		}
	}
	before := a.state.Quests[2]
	for _, cmd := range []string{"/quest start zero", "/quest choose 2 0", "/quest bad 2"} {
		if !a.questCommand(cmd) {
			t.Fatal(cmd)
		}
	}
	if a.state.Quests[2] != before {
		t.Fatal("invalid request changed authoritative quest state")
	}
	if a.questCommand("/party hello") {
		t.Fatal("intercepted unrelated command")
	}
}

func TestItemQuestOfferFiltersEligibilityAndCanBeCancelled(t *testing.T) {
	a := inputApp()
	a.questUI, _ = ui.NewQuestLog(inputTextures{})
	a.questConfirm, _ = ui.NewMessage(inputTextures{})
	a.questDataLoaded = true
	a.state.SelfID = 7
	a.state.Slots[0] = proto.CharacterSlot{ID: 7, Level: 20, Gender: 1}
	q := sdata.Quest{ID: 1, MaxLevel: 80, Female: true, Jobs: [6]uint8{1}, StartType: 4, StartItemType: 29, StartItemID: 6}
	a.questDefs = []sdata.Quest{q}
	item := sdata.ItemKey{Type: 29, TypeID: 6}
	a.questItemLinks = map[sdata.ItemKey]sdata.QuestItemLinks{item: {Start: []uint16{1}}}
	if a.offerItemQuest([2]uint8{1, 3}, item, 3) || !a.offerItemQuest([2]uint8{1, 3}, item, 4) {
		t.Fatal("inventory use should open only the matching offer")
	}
	a.updateQuestLog(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.questConfirm.Visible || a.questItemOffer.id != 0 || len(a.state.Quests) != 0 {
		t.Fatal("cancelling changed the quest or kept the offer")
	}
	a.state.Slots[0] = proto.CharacterSlot{ID: 7, Level: 20, Gender: 0}
	if a.offerItemQuest([2]uint8{1, 3}, item, 4) {
		t.Fatal("wrong gender received an offer")
	}
}

func TestQuestFollowUpUsesConfirmedVariantAndNPCContext(t *testing.T) {
	a := inputApp()
	a.questDefs = []sdata.Quest{
		{Results: [3]sdata.QuestResult{{NextQuest: 2}, {NextQuest: 3}}},
		{StartType: 1, StartNPCType: 8, StartNPCID: 5},
		{StartType: 0, PrevQuest: 1},
	}
	a.state.FinishedQuests = map[uint16]uint8{1: 1}
	npc := &game.Entity{ID: 9, NpcType: 8, TypeID: 4}
	result := &proto.QuestEndResult{ID: 1, Variant: 1, Success: 1}
	if next := a.npcQuestFollowUp(result, npc); next != 3 {
		t.Fatalf("expected variant's unlisted follow-up, got %d", next)
	}
	result.Variant = 0
	if next := a.npcQuestFollowUp(result, npc); next != 0 {
		t.Fatalf("follow-up requires a different NPC, got %d", next)
	}
}

func TestAutomaticQuestResultUsesConfirmedReward(t *testing.T) {
	a := inputApp()
	a.questDataLoaded = true
	a.questDefs = []sdata.Quest{{Name: "Journey", Completion: [3]string{"First", "Second"}}}
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 1, TypeID: 4}: {Name: "Sword"}}
	packet := proto.NewWriter(proto.QuestEnd)
	packet.U32(0)
	packet.U16(1)
	packet.U8(1)
	packet.U8(1)
	packet.U32(200)
	packet.U32(990)
	packet.Raw([]byte{1, 0, 1, 4})
	a.confirmedQuest(packet.Bytes())
	if len(a.state.Chat) != 2 || a.state.Chat[1].Text != "Second\n\nReceived: 200 XP, 990 gold\nSword" {
		t.Fatal("automatic quest result missing or incorrect", a.state.Chat)
	}
}
