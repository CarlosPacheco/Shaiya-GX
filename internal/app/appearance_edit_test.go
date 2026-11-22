package app

import (
	"bytes"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestAppearanceRequestWireLayout(t *testing.T) {
	got := proto.ChangeAppearancePacket(1, 2, 3, 4, 2, 1)
	want := []byte{0x26, 0x02, 1, 2, 3, 4, 2, 1}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %x, want %x", got, want)
	}
}

func TestAppearanceConsumableOpensFromActionBar(t *testing.T) {
	a := inputApp()
	a.appearanceUI = &ui.AppearanceEditor{}
	a.world = &session.World{}
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1}
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 25, TypeID: 7}: {Special: 45}}
	a.state.Inventory[[2]uint8{1, 2}] = proto.InventoryItem{Bag: 1, Slot: 2, Type: 25, TypeID: 7, Count: 1}
	a.state.QuickSlots = []proto.QuickSlot{{Bar: 0, Slot: 0, Bag: 1, Number: 2}}
	a.useAction(0)
	if a.appearanceEdit == nil || !a.appearanceUI.GenderEnabled {
		t.Fatal("action bar bypassed editor")
	}
}

func TestAppearanceEditorUsesConfirmedItemAndCancelsStaleDraft(t *testing.T) {
	a := inputApp()
	a.appearanceUI = &ui.AppearanceEditor{}
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Hair: 2, Face: 1, Height: 3, Gender: 1}
	a.inventory.open = true
	key := [2]uint8{1, 2}
	item := proto.InventoryItem{Bag: 1, Slot: 2, Type: 25, TypeID: 7, Count: 1}
	a.state.Inventory[key] = item
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 25, TypeID: 7}: {Special: 44}}
	if !a.openAppearanceEditor(key) || a.appearanceUI.Values != [4]uint8{2, 1, 3, 1} || a.appearanceUI.GenderEnabled {
		t.Fatal("appearance item did not open correct draft")
	}
	if a.state.Slots[0].Hair != 2 || a.state.Inventory[key] != item {
		t.Fatal("opening editor mutated server state")
	}
	a.state.Inventory[key] = proto.InventoryItem{}
	a.updateAppearanceEditor(&platform.Input{}, 800, 600)
	if a.appearanceEdit != nil {
		t.Fatal("stale item retained editor")
	}
	a.state.Inventory[key] = item
	a.items[sdata.ItemKey{Type: 25, TypeID: 7}].Special = 45
	if !a.openAppearanceEditor(key) || !a.appearanceUI.GenderEnabled {
		t.Fatal("sex change item did not enable sex")
	}
	a.updateAppearanceEditor(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 800, 600)
	if a.appearanceEdit != nil || a.state.Inventory[key] != item {
		t.Fatal("cancel changed item")
	}
}

func TestAppearanceWindowDragsFromItsFrame(t *testing.T) {
	a := inputApp()
	a.appearanceUI, _ = ui.NewAppearanceEditor(inputTextures{})
	a.appearanceEdit = &appearanceEdit{}
	x, y := a.appearanceUI.Origin(1024, 768)
	// Empty frame area, away from every control and the preview.
	if !a.updateWindowDrag(&platform.Input{MouseX: x + 40, MouseY: y + 250, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("press on the free frame did not start a drag")
	}
	a.updateWindowDrag(&platform.Input{MouseX: x + 140, MouseY: y + 290, MouseDown: true}, 1024, 768)
	a.updateWindowDrag(&platform.Input{MouseX: x + 140, MouseY: y + 290, Clicked: true}, 1024, 768)
	nx, ny := a.appearanceUI.Origin(1024, 768)
	if nx != x+100 || ny != y+40 {
		t.Fatalf("window moved to %d,%d from %d,%d", nx, ny, x, y)
	}
}
