package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestWarehouseCallClickReopensPersonalBankAndStillAllowsDragging(t *testing.T) {
	a := inputApp()
	var err error
	a.bankUI, err = ui.NewWarehouse(inputTextures{})
	if err != nil {
		t.Fatal(err)
	}
	a.state.MapID = 51
	active := proto.ActiveBuff{Skill: warehouseCallSkill, Instance: 3}
	a.state.Buffs = []proto.ActiveBuff{active}
	a.buffs.rows = []buffBadge{{buff: active}}
	a.bank = warehouseView{npcID: 1, guild: true, page: 4}
	a.state.Warehouse = map[uint8]proto.InventoryItem{0: {Type: 25, TypeID: 4}}
	a.state.Guild.Bank = map[uint8]proto.InventoryItem{0: {Type: 5, TypeID: 1}}
	const w, h = 1024, 768

	clickBuff := func(dx int) {
		x, y := a.buffSlot(w, h, 0)
		a.worldControls(&platform.Input{MouseX: x + 5, MouseY: y + 5, Pressed: true, MouseDown: true}, w, h)
		a.worldControls(&platform.Input{MouseX: x + 5 + dx, MouseY: y + 5, Clicked: true}, w, h)
	}
	clickBuff(0)
	if !a.remoteWarehouseOpen() || !a.inventory.open || a.bank.page != 0 {
		t.Fatal("clicking Warehouse Call did not open the personal warehouse and inventory")
	}
	a.updateWarehouse(&platform.Input{}, w, h)
	if a.warehouseBag() != proto.WarehouseBag || a.warehouseItems()[0] != a.state.Warehouse[0] {
		t.Fatal("remote warehouse used guild storage inside a guild house")
	}
	a.closeWarehouse()
	clickBuff(-80)
	if a.remoteWarehouseOpen() || !a.buffLayout.Position[0].Moved {
		t.Fatal("dragging the Warehouse Call buff opened the warehouse or failed to move the strip")
	}
	clickBuff(0)
	if !a.remoteWarehouseOpen() {
		t.Fatal("the moved buff did not reopen the warehouse")
	}
	a.state.Buffs = nil
	if a.remoteWarehouseOpen() {
		t.Fatal("the window remained open after the effect expired")
	}
}
