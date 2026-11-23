package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestBlacksmithDragAndConfirmation(t *testing.T) {
	a := inputApp()
	a.items = map[sdata.ItemKey]*sdata.Item{
		{Type: 1, TypeID: 1}:   {Type: 1, TypeID: 1, Slot: 6},
		{Type: 30, TypeID: 1}:  {Type: 30, TypeID: 1, Slot: 1, MinLevel: 1},
		{Type: 100, TypeID: 1}: {Type: 100, TypeID: 1, Special: 36},
	}
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.gemConfirm, _ = ui.NewMessage(inputTextures{})
	a.shop.npcID = 42
	a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 3, ID: 42}
	a.updateBlacksmith(&platform.Input{}, 1024, 768)
	a.blacksmithUI.Mode = ui.ForgeLapis
	if !a.inventory.open {
		t.Fatal("blacksmith did not open inventory")
	}
	keys := [][2]uint8{{1, 22}, {1, 21}}
	a.state.Inventory[keys[0]] = proto.InventoryItem{Bag: 1, Slot: 22, Type: 1, TypeID: 1, Count: 1}
	a.state.Inventory[keys[1]] = proto.InventoryItem{Bag: 1, Slot: 21, Type: 30, TypeID: 1, Count: 2}
	ix, iy := a.inventoryUI.Origin(1024, 768)
	bx, by := a.blacksmithUI.Origin(1024, 768)
	for slot, key := range keys {
		sx, sy, _, _ := ui.InventoryCell(key[0], key[1])
		dx, dy := ui.BlacksmithSlot(slot)
		a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: bx + dx + 2, MouseY: by + dy + 2, Clicked: true}, 1024, 768)
	}
	if !a.blacksmithReady() {
		t.Fatalf("drag did not select equipment and lapis: %+v", a.blacksmith)
	}
	a.worldControls(&platform.Input{MouseX: bx + 300, MouseY: by + 380, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: bx + 300, MouseY: by + 380, Clicked: true}, 1024, 768)
	if a.gemConfirm.Visible || !a.blacksmith.quote.pending {
		t.Fatalf("confirmation must await authoritative quote: visible=%t quote=%+v selected=%v npc=%d", a.gemConfirm.Visible, a.blacksmith.quote, a.blacksmith.selected, a.shop.npcID)
	}
	a.confirmedForgeQuote(testForgeQuotePacket(proto.ForgeQuote{Serial: a.blacksmith.quote.request.Serial, Kind: proto.ForgeQuoteLink, Valid: true, Rate: 250000, Gold: 0}))
	if !a.gemConfirm.Visible || a.shop.pending {
		t.Fatal("link must request confirmation without changing inventory")
	}
	a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.gemConfirm.Visible || a.shop.pending || !a.blacksmithReady() {
		t.Fatal("cancelling confirmation changed items")
	}
	item := a.state.Inventory[keys[1]]
	item.Count--
	a.state.Inventory[keys[1]] = item
	if a.blacksmithReady() {
		t.Fatal("stale selection remained eligible")
	}
}

func TestBlacksmithOptionalHammerCanBeRemoved(t *testing.T) {
	a := inputApp()
	a.items = map[sdata.ItemKey]*sdata.Item{
		{Type: 1, TypeID: 1}:   {Type: 1, TypeID: 1, Slot: 6},
		{Type: 30, TypeID: 1}:  {Type: 30, TypeID: 1, Slot: 1, MinLevel: 1},
		{Type: 100, TypeID: 1}: {Type: 100, TypeID: 1, Special: 36},
	}
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.shop.npcID = 42
	a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 3, ID: 42}
	a.updateBlacksmith(&platform.Input{}, 1024, 768)
	a.blacksmithUI.Mode = ui.ForgeLapis
	key := [2]uint8{1, 19}
	a.state.Inventory[key] = proto.InventoryItem{Bag: 1, Slot: 19, Type: 100, TypeID: 1, Count: 1}
	ix, iy := a.inventoryUI.Origin(1024, 768)
	bx, by := a.blacksmithUI.Origin(1024, 768)
	sx, sy, _, _ := ui.InventoryCell(key[0], key[1])
	dx, dy := ui.BlacksmithSlot(3)
	a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: bx + dx + 2, MouseY: by + dy + 2, Clicked: true}, 1024, 768)
	if !a.blacksmith.selected[3] || a.blacksmith.keys[3] != key {
		t.Fatal("optional hammer was not selected")
	}
	a.worldControls(&platform.Input{MouseX: bx + dx + 2, MouseY: by + dy + 2, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: bx + dx + 2, MouseY: by + dy + 2, Clicked: true}, 1024, 768)
	if a.blacksmith.selected[3] || a.state.Inventory[key].Count != 1 {
		t.Fatal("removing hammer selection changed inventory")
	}
}

func TestForgeModeSwitchClearsSelectionsWithoutInventoryMutation(t *testing.T) {
	a := inputApp()
	a.items = map[sdata.ItemKey]*sdata.Item{
		{Type: 1, TypeID: 1}:   {Type: 1, TypeID: 1, Slot: 6},
		{Type: 30, TypeID: 1}:  {Type: 30, TypeID: 1, Slot: 1, MinLevel: 1},
		{Type: 100, TypeID: 1}: {Type: 100, TypeID: 1, Special: 36},
	}
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.shop.npcID = 42
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 3}
	a.updateBlacksmith(&platform.Input{}, 1024, 768)
	x, y := a.blacksmithUI.Origin(1024, 768)
	if a.blacksmithUI.Mode != ui.ForgeMenu {
		t.Fatal("the blacksmith must open on its menu")
	}
	click := func(px, py int) {
		in := platform.Input{MouseX: x + px, MouseY: y + py, Pressed: true, MouseDown: true}
		a.updateBlacksmith(&in, 1024, 768)
		in.Pressed, in.MouseDown, in.Clicked = false, false, true
		a.updateBlacksmith(&in, 1024, 768)
	}
	// Menu buttons: recreation (32,295), link/extract (31,357), enchant (147,357).
	for _, step := range []struct{ x, y, mode int }{{40, 300, ui.ForgeRemake}, {40, 362, ui.ForgeLapis}, {150, 362, ui.ForgeEnchant}} {
		a.blacksmithUI.Mode = ui.ForgeMenu
		x, y = a.blacksmithUI.Origin(1024, 768)
		a.blacksmith.selected[0] = true
		click(step.x, step.y)
		if a.blacksmithUI.Mode != step.mode || a.blacksmith.selected != [4]bool{} || a.shop.pending {
			t.Fatal("menu switch retained stale selection or issued request", a.blacksmith, a.blacksmithUI.Mode)
		}
		// Cancel returns to the menu from every page.
		x, y = a.blacksmithUI.Origin(1024, 768)
		click(100, 306)
		if a.blacksmithUI.Mode != ui.ForgeMenu {
			t.Fatalf("cancel left page %d", step.mode)
		}
	}
}

func TestBlacksmithCloseResetsToMenuForSameNPC(t *testing.T) {
	a := inputApp()
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 3}

	for _, closeAction := range []string{"close_button", "escape", "range_exit"} {
		a.shop.npcID = 42
		a.updateBlacksmith(&platform.Input{}, 1024, 768)
		a.blacksmithUI.Mode = ui.ForgeLapis
		a.blacksmith.selected[0] = true

		switch closeAction {
		case "close_button":
			bx, by := a.blacksmithUI.Origin(1024, 768)
			a.updateBlacksmith(&platform.Input{MouseX: bx + 490, MouseY: by + 10, Pressed: true, MouseDown: true}, 1024, 768)
			a.updateBlacksmith(&platform.Input{MouseX: bx + 490, MouseY: by + 10, Clicked: true}, 1024, 768)
		case "escape":
			a.updateBlacksmith(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
		case "range_exit":
			a.closeNPC()
		}

		if a.shop.npcID != 0 || a.blacksmith.npcID != 0 || a.blacksmith.selected != [4]bool{} || a.blacksmithUI.Mode != ui.ForgeMenu {
			t.Fatalf("%s did not reset blacksmith state: npc=%d view=%+v mode=%d", closeAction, a.shop.npcID, a.blacksmith, a.blacksmithUI.Mode)
		}

		a.shop.npcID = 42
		a.updateBlacksmith(&platform.Input{}, 1024, 768)
		if a.blacksmithUI.Mode != ui.ForgeMenu || a.blacksmith.selected != [4]bool{} {
			t.Fatalf("%s reopened same blacksmith on stale page/selection: mode=%d selected=%v", closeAction, a.blacksmithUI.Mode, a.blacksmith.selected)
		}
	}
}
