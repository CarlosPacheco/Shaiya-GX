package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"
)

type inputTextures struct{}

func (inputTextures) Interface(string) (*render.Texture, error) { return nil, nil }

func inputApp() *App {
	message, err := ui.NewMessage(inputTextures{})
	if err != nil {
		panic(err)
	}
	market, err := ui.NewMarket(inputTextures{})
	if err != nil {
		panic(err)
	}
	quantity, err := ui.NewQuantity(inputTextures{})
	if err != nil {
		panic(err)
	}
	book, err := ui.NewSkillBook(inputTextures{})
	if err != nil {
		panic(err)
	}
	bottom, err := ui.NewBottomBar(inputTextures{})
	if err != nil {
		panic(err)
	}
	chat, err := ui.NewChatPanel(inputTextures{})
	if err != nil {
		panic(err)
	}
	status, err := ui.NewStatus(inputTextures{})
	if err != nil {
		panic(err)
	}
	sub, err := ui.NewSubActions(inputTextures{})
	if err != nil {
		panic(err)
	}
	menu, err := ui.NewGameMenu(inputTextures{})
	if err != nil {
		panic(err)
	}
	a := &App{gameMenu: menu, menuConfirm: message, statusUI: status, chatUI: chat, bottom: bottom, subActionsUI: sub, accountMessage: message, skillBook: book, quantityUI: quantity, state: game.New(), wv: &worldView{}, login: &ui.Login{}, actionBar: &ui.ActionBar{Hover: -1}, inventoryUI: &ui.Inventory{}, marketUI: market, skillDefs: map[sdata.SkillKey]sdata.Skill{}, npcDefs: map[sdata.NPCKey]sdata.NPC{}}
	a.inventoryUI.DockOrigin = a.inventoryDockOrigin
	a.marketUI.MerchantOrigin = a.merchantOrigin
	return a
}

func TestBottomMenuSharesPanelTogglesAndOwnsPointer(t *testing.T) {
	a := inputApp()
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true, Down: "W"}, 1024, 768)
		controls := a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true, Down: "W"}, 1024, 768)
		if controls.Clicked || controls.Down != "W" {
			t.Fatal("menu click reached world or stopped held movement")
		}
	}
	a.skills.open = true
	a.skills.assign = &proto.QuickSlot{Bag: 100, Number: 1}
	click(717, 715) // Native inventory: (646+2*35, h-64+10).
	if !a.inventory.open || !a.skills.open || a.skills.assign != nil {
		t.Fatal("inventory menu closed skills or retained assignment")
	}
	click(752, 715)
	if a.skills.open || !a.inventory.open || a.skills.assignMode {
		t.Fatal("skills menu did not close only the skill book")
	}
	click(752, 715)
	if !a.skills.open {
		t.Fatal("second skill click did not reopen the book")
	}
	click(717, 715) // Close inventory independently before checking release ownership.
	// The release of a world press cannot open a menu; a menu press released
	// over the world cannot select a target or begin camera movement.
	a.worldControls(&platform.Input{MouseX: 400, MouseY: 400, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: 717, MouseY: 715, Clicked: true}, 1024, 768)
	if a.inventory.open {
		t.Fatal("world press activated the menu on release")
	}
	a.worldControls(&platform.Input{MouseX: 717, MouseY: 715, Pressed: true, MouseDown: true}, 1024, 768)
	controls := a.worldControls(&platform.Input{MouseX: 400, MouseY: 400, Clicked: true}, 1024, 768)
	if controls.Clicked || a.inventory.open {
		t.Fatal("menu press escaped its pointer owner")
	}
}
func TestOpenInventoryAllowsWorldKeysAndActionBar(t *testing.T) {
	a := inputApp()
	a.inventory.open = true
	in := platform.Input{MouseX: 10, MouseY: 10, Keys: []platform.Key{platform.KeyTab}, Chars: []rune{'f'}}
	controls := a.worldControls(&in, 1024, 768)
	if !controls.HasKey(platform.KeyTab) || len(controls.Chars) != 1 {
		t.Fatal("inventory swallowed target/attack controls")
	}
	in.Chars = []rune{'1'}
	in.Keys = nil
	controls = a.worldControls(&in, 1024, 768)
	if len(controls.Chars) != 0 {
		t.Fatal("action bar was not handled while inventory was open")
	}
	in.Chars = nil
	in.Keys = []platform.Key{platform.KeyEscape}
	controls = a.worldControls(&in, 1024, 768)
	if a.inventory.open || controls.HasKey(platform.KeyEscape) {
		t.Fatal("escape should close inventory without logging out")
	}
}

func TestPersonalLightTogglesOncePerPress(t *testing.T) {
	a := inputApp()
	press := platform.Input{Keys: []platform.Key{platform.KeyPersonalLight}}
	a.updateWorld(&press, .1)
	if !a.wv.personalLight {
		t.Fatal("light did not turn on over a frame with multiple world steps")
	}
	a.updateWorld(&platform.Input{}, .1)
	if !a.wv.personalLight {
		t.Fatal("light did not stay on")
	}
	blocked := press
	blocked.SuppressAutoRun = true
	a.updateWorld(&blocked, .1)
	if !a.wv.personalLight {
		t.Fatal("blocked world input toggled the light")
	}
	a.updateWorld(&press, .1)
	if a.wv.personalLight {
		t.Fatal("second L press did not turn the light off")
	}
}
func TestInventoryDoesNotBlockShopScrollingOrWorldZoom(t *testing.T) {
	a := inputApp()
	a.inventory.open = true
	a.shop.npcID = 42
	a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 1, TypeID: 1}
	a.npcDefs[sdata.NPCKey{Type: 1, ID: 1}] = sdata.NPC{Items: make([]sdata.ItemKey, 61)}
	mx, my := a.marketUI.Origin(1024, 768)
	in := platform.Input{MouseX: mx + 18, MouseY: my + 138, Wheel: -120}
	controls := a.worldControls(&in, 1024, 768)
	if a.shop.page != 1 || controls.Wheel != 0 {
		t.Fatal("shop did not own its wheel with inventory open")
	}
	in.MouseX = mx - 40
	in.MouseY = 400
	controls = a.worldControls(&in, 1024, 768)
	if controls.Wheel != -120 || a.shop.page != 1 {
		t.Fatal("world zoom changed the shop page")
	}
}
func TestPanelPressCannotActivateAnotherPanelOrWorld(t *testing.T) {
	a := inputApp()
	a.inventory.open = true
	a.state.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1}
	a.inventory.bag = 1
	x, y := ui.InventoryOrigin(1024, 768)
	cx, cy, _, _ := ui.InventoryCell(1, 0)
	px, py := x+cx+1, y+cy+1
	in := platform.Input{MouseX: px, MouseY: py, Pressed: true, MouseDown: true}
	a.worldControls(&in, 1024, 768)
	in = platform.Input{MouseX: 10, MouseY: 10, Clicked: true}
	controls := a.worldControls(&in, 1024, 768)
	if controls.Clicked || a.inventory.selected {
		t.Fatal("release outside inventory activated the world")
	}
	// A press in the world followed by a release on an item must not select it.
	in = platform.Input{MouseX: 10, MouseY: 10, Pressed: true, MouseDown: true}
	a.worldControls(&in, 1024, 768)
	in = platform.Input{MouseX: px, MouseY: py, Clicked: true}
	controls = a.worldControls(&in, 1024, 768)
	if controls.Clicked || a.inventory.selected {
		t.Fatal("world press activated inventory on release")
	}
	// An ordinary click uses the item and clears selection.
	in = platform.Input{MouseX: px, MouseY: py, Pressed: true, MouseDown: true}
	a.worldControls(&in, 1024, 768)
	in.Pressed, in.MouseDown, in.Clicked = false, false, true
	a.worldControls(&in, 1024, 768)
	if a.inventory.selected {
		t.Fatal("normal inventory click retained the old move selection")
	}
}
func TestPanelTogglePreservesOtherWindowsAndCancelsAssignment(t *testing.T) {
	a := inputApp()
	a.skills.open = true
	a.skills.assign = &proto.QuickSlot{Bag: 100, Number: 1}
	a.worldControls(&platform.Input{Chars: []rune{'i'}}, 1024, 768)
	if !a.inventory.open || !a.skills.open || a.skills.assign != nil {
		t.Fatal("inventory closed skills or retained an active assignment")
	}
	a.worldControls(&platform.Input{Chars: []rune{'k'}}, 1024, 768)
	if a.skills.open || !a.inventory.open {
		t.Fatal("closing skills closed inventory")
	}
}
func TestClosedPanelStillChecksPendingTimeout(t *testing.T) {
	a := inputApp()
	a.clock = 11
	a.inventory.pending = true
	a.inventory.deadline = 10
	if a.checkPendingActions() || a.state != nil || a.login.Message == "" {
		t.Fatal("closed inventory hid an expired request")
	}
}

func TestCatalogDragCancelsWithoutActivatingDestination(t *testing.T) {
	for _, point := range [][2]int{{10, 10}, {500, 700}, {100, 100}, {800, 140}} {
		a := inputApp()
		a.inventory.open, a.inventory.bag = true, 1
		a.shop.npcID = 42
		a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 1, TypeID: 1}
		a.npcDefs[sdata.NPCKey{Type: 1, ID: 1}] = sdata.NPC{Items: []sdata.ItemKey{{Type: 25, TypeID: 4}}}
		x, y := a.marketUI.Origin(1024, 768)
		a.worldControls(&platform.Input{MouseX: x + 16, MouseY: y + 113, Pressed: true, MouseDown: true}, 1024, 768)
		if !a.shop.dragging {
			t.Fatal("catalog press did not start drag")
		}
		controls := a.worldControls(&platform.Input{MouseX: point[0], MouseY: point[1], Clicked: true}, 1024, 768)
		if a.shop.dragging || a.shop.pending || a.inventory.selected || controls.Clicked {
			t.Fatalf("invalid drop %v activated a destination", point)
		}
	}
}

func TestCatalogDragEscapeCancelsBeforeLogout(t *testing.T) {
	a := inputApp()
	a.shop.npcID, a.shop.dragging = 42, true
	a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 1}
	controls := a.worldControls(&platform.Input{MouseDown: true, Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.shop.dragging || controls.HasKey(platform.KeyEscape) || a.shop.npcID != 42 {
		t.Fatal("escape should cancel the drag before closing the shop or leaving the world")
	}
}

func TestInventoryDropDoesNotSellThroughOverlappingWindow(t *testing.T) {
	a := inputApp()
	a.inventory.open, a.inventory.bag, a.inventory.selected, a.inventory.dragging = true, 1, true, true
	a.inventory.source = [2]uint8{1, 0}
	a.state.Inventory[a.inventory.source] = proto.InventoryItem{Bag: 1, Slot: 0, Type: 25, TypeID: 4, Count: 1}
	a.shop.npcID = 42
	a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 1}
	a.wv.uiLeft = 3
	controls := a.worldControls(&platform.Input{MouseX: 150, MouseY: 200, Clicked: true}, 400, 768)
	if a.inventory.pending || a.inventory.dragging || a.inventory.selected || controls.Clicked {
		t.Fatal("drop sold through the overlapping inventory or reached the world")
	}
}

func TestTradeQuantityCapturesBarKeysAndRejectsChangedItem(t *testing.T) {
	a := inputApp()
	a.shop.npcID = 42
	a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 1}
	source := [2]uint8{1, 0}
	it := proto.InventoryItem{Bag: 1, Slot: 0, Type: 25, TypeID: 4, Count: 3}
	a.state.Inventory[source] = it
	a.shop.quantity = &tradeQuantity{npcID: 42, source: source, item: it}
	a.quantityUI.Open(400, 300, 1024, 768, 3)
	a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: a.quantityUI.X + 30, MouseY: a.quantityUI.Y + 40}, 1024, 768)
	typed := a.worldControls(&platform.Input{Chars: []rune{'2'}, KeypadChars: "2", Actions: []int{11}}, 1024, 768)
	if a.quantityUI.Value() != 2 || len(typed.Actions) != 0 {
		t.Fatal("keypad digit did not edit the sale quantity or leaked to the action bar")
	}
	controls := a.worldControls(&platform.Input{Chars: []rune{'1'}, Keys: []platform.Key{platform.KeyEnter}, Clicked: true, MouseX: 5, MouseY: 5}, 1024, 768)
	if controls.Clicked || len(controls.Chars) > 0 || len(controls.Keys) > 0 || a.shop.quantity == nil {
		t.Fatal("quantity leaked an input or Enter confirmed without OK")
	}
	delete(a.state.Inventory, source)
	a.worldControls(&platform.Input{}, 1024, 768)
	if a.shop.quantity != nil || a.inventory.pending {
		t.Fatal("stale quantity dialog was not canceled")
	}
}

func TestDropQuantityConsumesKeypadDigit(t *testing.T) {
	a := inputApp()
	item := proto.InventoryItem{Bag: 1, Slot: 0, Type: 25, TypeID: 4, Count: 5}
	key := [2]uint8{item.Bag, item.Slot}
	a.state.Inventory[key] = item
	a.drop = itemDrop{active: true, stack: true, source: key, item: item}
	a.quantityUI.Open(400, 300, 1024, 768, 5)
	a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: a.quantityUI.X + 30, MouseY: a.quantityUI.Y + 40}, 1024, 768)
	in := &platform.Input{Chars: []rune{'3'}, KeypadChars: "3", Actions: []int{12}}
	if controls := a.worldControls(in, 1024, 768); a.quantityUI.Value() != 3 || len(controls.Actions) != 0 {
		t.Fatal("keypad digit did not edit the drop quantity or leaked to the action bar")
	}
}

func TestCameraRightDragKeepsOwnershipAcrossPanels(t *testing.T) {
	a := inputApp()
	start := a.worldControls(&platform.Input{MouseX: 400, MouseY: 300, RightPressed: true, RightDown: true}, 1024, 768)
	if !start.RightDown {
		t.Fatal("world did not receive initial camera press")
	}
	a.wv.mouselook = true
	a.inventory.open = true
	// Cross the chat, inventory and bottom bar with the virtual pointer.
	for _, p := range [][2]int{{20, 500}, {800, 200}, {717, 715}} {
		in := platform.Input{MouseX: p[0], MouseY: p[1], RightDown: true, Down: "W"}
		controls := a.worldControls(&in, 1024, 768)
		if !controls.RightDown || controls.Down != "W" || a.wv.uiRight {
			t.Fatal("panel stole the camera drag", p, controls)
		}
	}
	controls := a.worldControls(&platform.Input{MouseX: 20, MouseY: 500}, 1024, 768)
	if controls.RightDown {
		t.Fatal("camera drag survived release")
	}
	// A right press that starts over chat still belongs to chat on exit.
	a.wv.mouselook = false
	a.worldControls(&platform.Input{MouseX: 20, MouseY: 500, RightPressed: true, RightDown: true}, 1024, 768)
	controls = a.worldControls(&platform.Input{MouseX: 400, MouseY: 300, RightDown: true}, 1024, 768)
	if controls.RightDown {
		t.Fatal("UI press escaped into camera orbit")
	}
}

func TestCatalogIDsToggleAndItemTooltip(t *testing.T) {
	a := inputApp()
	a.state.GroundItems[1] = proto.GroundItem{X: a.wv.pos[0] + 10, Z: a.wv.pos[2]}
	in := platform.Input{Keys: []platform.Key{platform.KeyShowIDs}, Chars: []rune{'h'}}
	controls := a.worldControls(&in, 1024, 768)
	a.updateCombat(&controls, 1024, 768)
	if a.wv.travel || a.wv.approachTarget != 0 || a.wv.autoRun {
		t.Fatal("H started a pickup or movement")
	}
	if !a.showCatalogIDs {
		t.Fatal("H did not enable catalog IDs")
	}
	a.worldControls(&platform.Input{Chars: []rune{'h'}}, 1024, 768)
	pointer := chatWorldInput(in)
	a.worldControls(&pointer, 1024, 768)
	if !a.showCatalogIDs {
		t.Fatal("text input or key repeat toggled catalog IDs")
	}
	if npc, mob := a.catalogNameSuffix(&game.Entity{Kind: game.KindNpc, NpcType: 6, TypeID: 1}), a.catalogNameSuffix(&game.Entity{Kind: game.KindMob, TypeID: 61}); npc != " (6 1)" || mob != " (61)" {
		t.Fatalf("catalog suffixes did not preserve their fields: %q, %q", npc, mob)
	}
	item := &sdata.Item{Type: 100, TypeID: 120, Name: "Test item"}
	a.drawItemDetails(item, nil, 1, 0, 0, 1024, 768, "")
	if len(a.tooltip) < 2 || a.tooltip[1] != (itemDetailLine{"ID: 100 120", white}) {
		t.Fatalf("full item ID missing from tooltip: %+v", a.tooltip)
	}
	a.worldControls(&in, 1024, 768)
	a.drawItemDetails(item, nil, 1, 0, 0, 1024, 768, "")
	for _, row := range a.tooltip {
		if row.text == "ID: 100 120" {
			t.Fatal("item ID remained after hiding catalog IDs")
		}
	}
}
