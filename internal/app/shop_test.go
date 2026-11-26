package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestNDoesNotSelectNearbyNPC(t *testing.T) {
	a := &App{state: game.New(), wv: &worldView{}, npcDefs: map[sdata.NPCKey]sdata.NPC{}}
	a.state.Entities[10] = &game.Entity{ID: 10, Kind: game.KindNpc, X: 5}
	for _, key := range []rune{'n', 'N'} {
		if a.updateShop(&platform.Input{Chars: []rune{key}}, 1024, 768) ||
			a.shop.npcID != 0 || a.wv.approachTarget != 0 {
			t.Fatalf("%c activated NPC interaction", key)
		}
	}
}

func TestNPCWindowsCloseAtNativeDistance(t *testing.T) {
	for _, tc := range []struct {
		name        string
		kind        uint8
		talk, quest bool
	}{
		{name: "conversation", kind: 1, talk: true},
		{name: "merchant", kind: 1},
		{name: "gatekeeper", kind: 2},
		{name: "blacksmith", kind: 3},
		{name: "reward", kind: 4},
		{name: "warehouse", kind: 6},
		{name: "quest", kind: 1, quest: true},
		{name: "guild manager", kind: 11},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &App{state: game.New(), wv: &worldView{}, gateConfirm: &ui.Message{Visible: true}}
			e := &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: tc.kind}
			a.state.Entities[e.ID] = e
			a.shop = shopView{npcID: e.ID, talk: tc.talk, questMode: tc.quest, quantity: &tradeQuantity{}}
			a.gate = gatekeeperView{npc: e.ID, pending: true}
			a.shop.pending = true
			a.inventory.open = true
			a.wv.pos = render.V3{3, 0, 0}
			a.checkNPCWindowRange()
			if a.shop.npcID != e.ID || a.shop.quantity == nil || !a.gateConfirm.Visible {
				t.Fatal("NPC window closed at the allowed boundary")
			}
			a.wv.pos[1] = 1
			a.checkNPCWindowRange()
			if a.shop.npcID != 0 || a.shop.talk || a.shop.questMode || a.shop.quantity != nil || a.gateConfirm.Visible {
				t.Fatal("an out-of-range NPC window or confirmation remained open")
			}
			if !a.shop.pending || !a.gate.pending || !a.inventory.open {
				t.Fatal("closing NPC discarded a pending response or closed the inventory")
			}
		})
	}
}

func TestNPCWindowRangeTracksMovingNPCAndLeavesRemoteWarehouseOpen(t *testing.T) {
	a := &App{state: game.New(), wv: &worldView{}}
	e := &game.Entity{ID: 42, Kind: game.KindNpc}
	a.state.Entities[e.ID] = e
	a.shop.npcID = e.ID
	a.checkNPCWindowRange()
	if a.shop.npcID != e.ID {
		t.Fatal("nearby NPC window closed")
	}
	e.Z = 4
	a.checkNPCWindowRange()
	if a.shop.npcID != 0 {
		t.Fatal("window remained open after NPC moved away")
	}
	a.remoteBank = true
	a.checkNPCWindowRange()
	if !a.remoteBank {
		t.Fatal("NPC range closed remote warehouse")
	}
}

func TestWarehouseInventoryDocksAndDragsWithoutSavingServicePosition(t *testing.T) {
	a := inputApp()
	a.bankUI, _ = ui.NewWarehouse(inputTextures{})
	const w, h = 1024, 768
	a.inventoryUI.Position.Move(740, 190, w, h, ui.InventoryWidth, ui.InventoryHeight)
	saved := a.inventoryUI.Position
	e := &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 6}
	a.state.Entities[e.ID] = e
	a.shop.npcID, a.shop.talk = e.ID, true
	a.enterNPCService(e)
	if !a.inventory.open {
		t.Fatal("warehouse did not open inventory")
	}
	a.bankUI.Position.Move(600, 110, w, h, ui.WarehouseWidth, ui.WarehouseHeight)
	ix, iy := a.inventoryUI.Origin(w, h)
	if ix+ui.InventoryWidth != 600 || iy != 110 || a.inventoryUI.Position != saved {
		t.Fatal("warehouse dock changed the inventory's saved position")
	}
	a.worldControls(&platform.Input{MouseX: ix + 40, MouseY: iy + 15, Pressed: true, MouseDown: true}, w, h)
	a.worldControls(&platform.Input{MouseX: ix + 70, MouseY: iy + 45, Clicked: true}, w, h)
	a.bankUI.Position.Move(400, 90, w, h, ui.WarehouseWidth, ui.WarehouseHeight)
	if x, y := a.inventoryUI.Origin(w, h); x != ix+30 || y != iy+30 || a.inventoryUI.Position != saved {
		t.Fatalf("warehouse inventory drag landed at %d,%d", x, y)
	}
	a.closeNPC()
	if x, y := a.inventoryUI.Origin(w, h); x != 740 || y != 190 {
		t.Fatalf("inventory returned to %d,%d", x, y)
	}
}

func TestMerchantAnchorsToSavedInventoryAndMovesIndependently(t *testing.T) {
	const w, h = 1024, 768
	for _, tc := range []struct{ inventoryX, merchantX int }{{740, 484}, {80, 336}} {
		a := inputApp()
		a.inventoryUI.Position.Move(tc.inventoryX, 190, w, h, ui.InventoryWidth, ui.InventoryHeight)
		a.marketUI.Position.Move(210, 110, w, h, ui.MarketWidth, ui.MarketHeight)
		e := &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 1}
		a.state.Entities[e.ID] = e
		open := func() {
			a.shop.npcID, a.shop.talk = e.ID, true
			a.enterNPCService(e)
		}
		open()
		mx, my := a.marketUI.Origin(w, h)
		if mx != tc.merchantX || my != 190 || !a.inventory.open {
			t.Fatalf("merchant opened at %d,%d with inventory at x=%d", mx, my, tc.inventoryX)
		}
		a.worldControls(&platform.Input{MouseX: mx + 50, MouseY: my + 15, Pressed: true, MouseDown: true}, w, h)
		a.worldControls(&platform.Input{MouseX: mx + 65, MouseY: my + 5, Clicked: true}, w, h)
		if x, y := a.marketUI.Origin(w, h); x != mx+15 || y != my-10 {
			t.Fatalf("merchant did not move independently: %d,%d", x, y)
		}
		if x, y := a.inventoryUI.Origin(w, h); x != tc.inventoryX || y != 190 {
			t.Fatalf("dragging merchant moved inventory to %d,%d", x, y)
		}
		a.toggleInventory()
		mx, my = a.marketUI.Origin(w, h)
		var closeCues int
		in := platform.Input{MouseX: mx + 232, MouseY: my + 10, Pressed: true, MouseDown: true}
		in.WindowClosed = func() { closeCues++ }
		a.worldControls(&in, w, h)
		in.Pressed, in.MouseDown, in.Clicked = false, false, true
		a.worldControls(&in, w, h)
		if a.shop.npcID != 0 || a.inventory.open || closeCues != 1 {
			t.Fatal("merchant close changed the inventory or missed its close cue", closeCues)
		}
		open()
		if x, y := a.marketUI.Origin(w, h); x != tc.merchantX || y != 190 {
			t.Fatalf("reopening merchant reused last drag location: %d,%d", x, y)
		}
	}
}
