//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestWarehouseMovesAgainstRapture(t *testing.T) {
	st, w, _, wait := testworld.Enter(t)
	wait(func() bool { return st.WarehouseReady })
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.arc = arc
	a.state = st
	a.world = w
	bank, err := ui.NewWarehouse(inputTextures{})
	if err != nil {
		t.Fatal(err)
	}
	a.bankUI = bank
	var npc uint32
	wait(func() bool {
		for id, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 6 {
				npc = id
				return true
			}
		}
		return false
	})
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	a.openNPC(npc)
	if a.shop.npcID != npc {
		t.Fatal("interacting with the server's banker did not open it")
	}
	a.worldControls(&platform.Input{}, 1024, 768)
	if !a.inventory.open || !a.warehouseOpen() {
		t.Fatal("banker did not open warehouse and inventory")
	}
	var source [2]uint8
	found := false
	for key, item := range st.Inventory {
		if key[0] > 0 && item.Type != 0 && item.Count > 0 {
			source = key
			found = true
			break
		}
	}
	if !found {
		t.Fatal("disposable fixture has no inventory item")
	}
	var slot uint8
	for ; slot < 240; slot++ {
		if _, ok := st.Warehouse[slot]; !ok {
			break
		}
	}
	if slot >= 240 {
		t.Fatal("warehouse fixture full")
	}
	a.bank.npcID = npc
	a.bank.page = int(slot) / 40
	a.inventory.open = true
	a.inventory.bag = source[0]
	ix, iy := a.inventoryUI.Origin(1024, 768)
	sx, sy, _, _ := ui.InventoryCell(source[0], source[1])
	bx, by := a.bankUI.Origin(1024, 768)
	dx, dy, _ := ui.WarehouseCell(a.bank.page, slot)
	drag := func(x, y, xx, yy int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: xx, MouseY: yy, Clicked: true}, 1024, 768)
	}
	ack := func() {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for a.inventory.pending {
			a.pumpWorld()
			if time.Now().After(deadline) {
				t.Fatal("warehouse acknowledgement timeout")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	before := st.Inventory[source]
	drag(ix+sx+1, iy+sy+1, bx+dx+1, by+dy+1)
	if !a.inventory.pending || st.Inventory[source] != before {
		t.Fatal("deposit changed state before server response")
	}
	ack()
	if _, ok := st.Inventory[source]; ok {
		t.Fatal("deposit did not clear source")
	}
	if st.Warehouse[slot].Type != before.Type {
		t.Fatal("deposit missing from warehouse")
	}
	originalPage := a.bank.page
	nextPage := (originalPage + 1) % 6
	nextSlot, ok := firstFreeWarehouseSlot(st.Warehouse, nextPage)
	if !ok {
		t.Fatal("disposable target page is full")
	}
	tabX := func(page int) int { return bx + 12 + page*52 }
	drag(bx+dx+1, by+dy+1, tabX(nextPage), by+115)
	if !a.inventory.pending || a.bank.page != nextPage || st.Warehouse[slot].Type != before.Type {
		t.Fatal("dropping on another tab did not request a move")
	}
	ack()
	want := before
	want.Bag, want.Slot = proto.WarehouseBag, nextSlot
	if _, exists := st.Warehouse[slot]; exists || st.Warehouse[nextSlot] != want {
		t.Fatal("cross-page tab drop did not move to the first empty slot")
	}

	nx, ny, _ := ui.WarehouseCell(nextPage, nextSlot)
	drag(bx+nx+1, by+ny+1, bx+nx+1, by+ny+1)
	drag(tabX(originalPage), by+115, tabX(originalPage), by+115)
	if !a.bank.selected || a.bank.source != nextSlot || a.bank.page != originalPage {
		t.Fatal("switching pages lost the selected item")
	}
	drag(bx+dx+1, by+dy+1, bx+dx+1, by+dy+1)
	if !a.inventory.pending {
		t.Fatal("clicking a destination cell on another page did not request a move")
	}
	ack()
	want.Slot = slot
	if _, exists := st.Warehouse[nextSlot]; exists || st.Warehouse[slot] != want {
		t.Fatal("cross-page selected item did not return to its original slot")
	}
	drag(bx+dx+1, by+dy+1, ix+sx+1, iy+sy+1)
	if !a.inventory.pending || st.Warehouse[slot].Type != before.Type {
		t.Fatal("withdrawal changed state before response")
	}
	ack()
	if _, ok := st.Warehouse[slot]; ok {
		t.Fatal("withdrawal did not clear warehouse")
	}
	if st.Inventory[source].Type != before.Type {
		t.Fatal("withdrawal missing from bag")
	}
}

func TestWarehouseDragDrawsOverInventoryGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires OpenGL and client data")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const w, h = 1024, 768
	win, err := platform.Open(platform.Config{Title: "warehouse drag", Width: w, Height: h, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.Preview("world-guild-bank"); err != nil {
		t.Fatal(err)
	}
	a.inventory.open, a.inventory.bag = true, 1
	x, y := a.inventoryUI.Origin(w, h)
	a.lastMouseX, a.lastMouseY = x+55, y+220
	item := a.state.Guild.Bank[0]
	def := a.itemsItem(item.Type, item.TypeID)
	if def == nil {
		t.Fatal("warehouse test item definition is missing")
	}
	gl.Viewport(0, 0, w, h)
	draw := func(dragging, reference bool) [32 * 32 * 4]byte {
		t.Helper()
		a.bank.selected, a.bank.dragging = dragging, dragging
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(w, h)
		a.drawCharacters(w, h)
		if reference {
			a.drawItemIcon(def, a.lastMouseX-16, a.lastMouseY-16, 32, [4]float32{1, 1, 1, .85}, &item)
		}
		a.sprites.End()
		var pixels [32 * 32 * 4]byte
		gl.ReadPixels(int32(a.lastMouseX-16), int32(h-a.lastMouseY-16), 32, 32, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	plain, dragged, over := draw(false, false), draw(true, false), draw(false, true)
	if plain == over || dragged != over {
		t.Fatal("dragged warehouse item is not visible in front of inventory")
	}
}
