package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Every NPC opens the conversation window first; its left button enters the
// service and the right one the NPC's first quest.
func TestNPCConversationLeadsToServiceOrQuest(t *testing.T) {
	for _, kind := range []uint8{1, 2, 3, 6} {
		for _, quest := range []bool{false, true} {
			a := inputApp()
			a.npcTalkUI, _ = ui.NewNPCTalk(inputTextures{})
			a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
			a.bankUI, _ = ui.NewWarehouse(inputTextures{})
			a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
			a.gateUI, _ = ui.NewGatekeeper(inputTextures{})
			a.questDataLoaded = true
			a.questDefs = []sdata.Quest{{Name: "Quest"}}
			e := &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: kind, TypeID: 1}
			a.state.Entities[e.ID] = e
			a.shop.npcID, a.shop.talk = e.ID, true
			a.npcDefs[sdata.NPCKey{Type: kind, ID: 1}] = sdata.NPC{StartQuests: []uint16{1}}
			click := func(x, y int) {
				a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
				a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
			}
			x, y := a.npcTalkUI.Origin(1024, 768)
			a.worldControls(&platform.Input{MouseX: 1, MouseY: 1}, 1024, 768) // one frame places the controls
			if quest {
				click(x+145+10, y+336+8)
				if a.shop.talk || !a.shop.questMode || a.npcQuest.npc != 42 || a.npcQuest.selected != 1 {
					t.Fatalf("type %d: the talk button did not open the quest", kind)
				}
				continue
			}
			click(x+53+10, y+336+8)
			if a.shop.talk || a.shop.questMode || a.shop.npcID != 42 {
				t.Fatalf("type %d: the service button did not open the service", kind)
			}
		}
	}
}
