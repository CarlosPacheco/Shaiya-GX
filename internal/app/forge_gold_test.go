package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/forgecost"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
	"testing"
)

func TestForgeGoldUsesOperationSysMsgAndExactThreshold(t *testing.T) {
	for _, id := range []int{10010, 10013, 10102} {
		a := inputApp()
		a.msg = text.SysMsg{id: "Native insufficient gold: <b>"}
		cost := forgecost.Gold(96)
		if !a.insufficientForgeGold(cost-1, cost, id) {
			t.Fatal("insufficient balance accepted")
		}
		if len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != id || a.state.Chat[0].Text != "Native insufficient gold: 2500000" {
			t.Fatal("generic/missing native notice", a.state.Chat)
		}
		if a.insufficientForgeGold(cost, cost, id) || len(a.state.Chat) != 1 {
			t.Fatal("exact price refused or additional message")
		}
	}
}

func TestForgeRequestsRejectInsufficientGoldWithNativeNotice(t *testing.T) {
	for _, operation := range []string{"enchant", "link", "extract"} {
		a := inputApp()
		a.world = &session.World{}
		a.msg = text.SysMsg{10010: "Link needs <b>", 10013: "Extract needs <b>", 10102: "Enchant needs <b>"}
		a.items = map[sdata.ItemKey]*sdata.Item{{Type: 95, TypeID: 1}: {ReqIg: 96}, {Type: 30, TypeID: 1}: {ReqIg: 30}, {Type: 100, TypeID: 140}: {Special: 58}}
		a.state.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Type: 1, TypeID: 1, Count: 1, Gems: [6]uint8{1}}
		a.state.Inventory[[2]uint8{1, 1}] = proto.InventoryItem{Type: 95, TypeID: 1, Count: 255}
		a.state.Inventory[[2]uint8{1, 2}] = proto.InventoryItem{Type: 25, TypeID: 43, Count: 255}
		a.state.Inventory[[2]uint8{1, 3}] = proto.InventoryItem{Type: 30, TypeID: 1, Count: 255}
		a.state.Inventory[[2]uint8{1, 4}] = proto.InventoryItem{Type: 100, TypeID: 140, Count: 255}
		var sent bool
		want := 10102
		switch operation {
		case "enchant":
			sent = a.sendEnchant([2]uint8{1, 1}, [2]uint8{1, 2}, [2]uint8{1, 0}, 1)
		case "link":
			sent = a.sendGemLink([2]uint8{1, 3}, [2]uint8{1, 0})
			want = 10010
		case "extract":
			a.blacksmith.selected[0] = true
			a.blacksmith.keys[0] = [2]uint8{1, 0}
			a.blacksmith.extractSlot = 0
			sent = a.sendGemRemove([2]uint8{1, 0}, [2]uint8{1, 4})
			want = 10013
		}
		if sent || a.shop.pending || len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != want {
			t.Fatal(operation, "gold rejection", a.state.Chat)
		}
	}
}
