package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestAuctionRoutesBoardAndKeepsSelectionOnRefresh(t *testing.T) {
	a := inputApp()
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 1, TypeID: 240}
	a.npcDefs[sdata.NPCKey{Type: 1, ID: 240}] = sdata.NPC{MerchantType: proto.AuctionBoard}
	a.shop.npcID = 42
	a.beginAuction(42)
	a.auction.rows = []proto.AuctionRow{{ID: 3}, {ID: 5}}
	a.auction.selected = 1
	a.auction.pending = true
	a.auction.sequence = 2
	a.applyAuction(proto.AuctionPacket(proto.AuctionReply{Sequence: 1, NPC: 42, Rows: []proto.AuctionRow{{ID: 99}}}))
	if !a.auction.pending || a.auction.row().ID != 5 {
		t.Fatal("stale reply replaced the current listing")
	}
	a.applyAuction(proto.AuctionPacket(proto.AuctionReply{Sequence: 2, NPC: 42, Rows: []proto.AuctionRow{{ID: 5}, {ID: 3}}}))
	if a.auction.pending || a.auction.selected != 0 || a.auction.row().ID != 5 {
		t.Fatal("refresh lost selection")
	}
	a.state.Self.Dead = true
	a.updateAuction(&platform.Input{}, 1024, 768)
	if a.auction.npc != 0 {
		t.Fatal("market stayed open after death")
	}
}

func TestAuctionAcceptsExplicitInventoryDrag(t *testing.T) {
	a := inputApp()
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 1, TypeID: 240}
	a.npcDefs[sdata.NPCKey{Type: 1, ID: 240}] = sdata.NPC{MerchantType: proto.AuctionBoard}
	a.shop.npcID = 42
	a.beginAuction(42)
	a.auction.view = proto.AuctionSelling
	a.auctionAction("setup")
	key := [2]uint8{1, 0}
	a.inventory.bag = 1
	a.state.Inventory[key] = proto.InventoryItem{Bag: 1, Type: 25, TypeID: 1, Count: 9}
	ix, iy := a.inventoryUI.Origin(1024, 768)
	sx, sy, _, _ := ui.InventoryCell(1, 0)
	a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
	if a.auction.source.Type != 0 {
		t.Fatal("inventory click picked a listing source")
	}
	x, y, width := a.auctionPanel(1024, 768)
	slot := auctionDialogBounds(x, y, width, auctionHeight(768), "setup").source()
	a.worldControls(&platform.Input{MouseX: slot.x + 4, MouseY: slot.y + 4, Clicked: true}, 1024, 768)
	if a.auction.source != a.state.Inventory[key] || a.auction.fields[0] != "9" || a.auction.token == "" {
		t.Fatal("auction did not accept inventory drop", a.auction.source)
	}
}

func TestAuctionActionsFitAndKeypadEditsOnlyItsField(t *testing.T) {
	v := auctionView{}
	for view := proto.AuctionBrowse; view <= proto.AuctionWatch; view++ {
		v.view = view
		for _, width := range []int{380, 504, 710} {
			right := 0
			for _, button := range v.actions(width) {
				if button.x < right || button.x+button.w > width-14 {
					t.Fatalf("view %d width %d has overlapping or clipped action %s", view, width, button.action)
				}
				right = button.x + button.w
			}
		}
	}
	a := inputApp()
	a.auction = auctionView{focus: 1, keypad: true, fields: [3]string{"2", "500", "900"}}
	const width = 504
	x := width - 242
	a.updateAuctionKeypad(x+4, 98+108, width)  // Clear
	a.updateAuctionKeypad(x+4, 98+36, width)   // 4
	a.updateAuctionKeypad(x+76, 98+108, width) // 0
	a.updateAuctionKeypad(x+4, 98+144, width)  // Done
	if a.auction.keypad || a.auction.fields != [3]string{"2", "40", "900"} {
		t.Fatal(a.auction.fields, a.auction.keypad)
	}
}

func TestAuctionPairedListsKeepIndependentSelectionAndPage(t *testing.T) {
	a := inputApp()
	a.auction = auctionView{
		npc: 42, view: proto.AuctionItems, selected: 0, page: 1,
		rows:        []proto.AuctionRow{{ID: 101}, {ID: 102}},
		other:       auctionList{selected: -1},
		pending:     true,
		sequence:    4,
		requestView: proto.AuctionGold,
	}
	a.applyAuction(proto.AuctionPacket(proto.AuctionReply{
		Sequence: 4, NPC: 42, View: proto.AuctionGold, More: true,
		Rows: []proto.AuctionRow{{ID: 201}, {ID: 202}},
	}))
	v := &a.auction
	if v.view != proto.AuctionItems || v.row().ID != 101 || v.page != 1 {
		t.Fatal("gold reply changed the selected item claim or its page")
	}
	panes := auctionPanes(proto.AuctionItems, 380)
	left, right := panes[0], panes[1]
	a.updateAuctionPanes(right.x+2, auctionPaneRowY+26, 380, auctionMinHeight, 0)
	if v.view != proto.AuctionGold || v.row().ID != 202 || !v.more {
		t.Fatal("click did not select the gold claim")
	}
	a.updateAuctionPanes(right.x+right.width-2, 365, 380, auctionMinHeight, 0)
	if v.page != 1 || v.row() != nil {
		t.Fatal("next page did not clear the gold selection")
	}
	a.updateAuctionPanes(left.x+2, auctionPaneRowY, 380, auctionMinHeight, 0)
	if v.view != proto.AuctionItems || v.page != 1 || v.row().ID != 101 {
		t.Fatal("gold pagination changed the item claims")
	}
}

func TestAuctionBrowseControlsFitNarrowPanel(t *testing.T) {
	for _, width := range []int{380, 504, 710} {
		right := 14
		for _, field := range auctionBrowseControls(width) {
			if field.x < right || field.width < 40 || field.x+field.width > width-14 {
				t.Fatalf("width %d has clipped or overlapping control %+v", width, field)
			}
			right = field.x + field.width
		}
	}
}

func TestAuctionDialogControlsUseVisibleBounds(t *testing.T) {
	a := inputApp()
	a.auction = auctionView{modal: "setup", focus: -1}
	const x, y, width = 8, 20, 380
	b := auctionDialogBounds(x, y, width, auctionMinHeight, "setup")
	click := func(rx, ry int) {
		a.updateAuctionDialog(&platform.Input{MouseX: b.x + rx, MouseY: b.y + ry, Pressed: true}, x, y, width, auctionMinHeight)
	}
	click(14+3*((b.width-28)/4)+4, auctionDurationY+4)
	if a.auction.duration != 3 {
		t.Fatal("duration button did not select the fourth duration")
	}
	click(b.width-75, auctionDialogFieldRow("setup", 1)+4)
	if !a.auction.keypad || a.auction.focus != 1 {
		t.Fatal("price Input button did not open the keypad")
	}
	click(b.width-110, b.height-28)
	if a.auction.keypad || a.auction.modal != "setup" {
		t.Fatal("outside click should dismiss only the keypad")
	}
	click(b.width-110, b.height-28)
	if a.auction.modal != "" || a.auction.focus != -1 {
		t.Fatal("the visible Cancel button did not close the dialog")
	}
}

func TestAuctionCategoryRailAndResultRowsHaveSeparateHitAreas(t *testing.T) {
	a := inputApp()
	const npc = 42
	a.state.Entities[npc] = &game.Entity{ID: npc}
	a.shop.npcID = npc
	a.auction = auctionView{npc: npc, selected: -1, focus: -1, page: 2, exactType: 18}
	for _, viewport := range [][2]int{{404, 480}, {1024, 768}} {
		w, h := viewport[0], viewport[1]
		x, y, width := a.auctionPanel(w, h)
		click := func(rx, ry int) {
			a.updateAuction(&platform.Input{MouseX: x + rx, MouseY: y + ry, Pressed: true}, w, h)
		}
		a.auction.rows = []proto.AuctionRow{{ID: 1}, {ID: 2}}
		click(18, 138+2*26+5)
		if a.auction.category != 2 || a.auction.page != 0 || a.auction.exactType != 0 || a.auction.selected >= 0 {
			t.Fatal("category click did not reset the results")
		}
		click(auctionListX(proto.AuctionBrowse, width)+4, 138+auctionRowPitch(auctionHeight(h))+4)
		if a.auction.category != 2 || a.auction.selected != 1 {
			t.Fatal("list click did not select the second row independently of the category")
		}
	}
}

func TestAuctionExpandedCategoryScrollAndSubcategorySelection(t *testing.T) {
	a := inputApp()
	a.auction = auctionView{selected: -1}
	a.updateAuctionCategories(auctionCategoryY+auctionCategoryPitch+5, auctionMinHeight, 0)
	if a.auction.category != 1 || a.auction.subcategory != 0 {
		t.Fatal("weapons group did not expand")
	}
	a.updateAuctionCategories(auctionCategoryY+2*auctionCategoryPitch+5, auctionMinHeight, 0)
	if a.auction.subcategory != 1 {
		t.Fatal("first weapon filter did not select")
	}
	a.auction.page = 2
	a.updateAuctionCategories(auctionCategoryY, auctionMinHeight, -120)
	if a.auction.categoryScroll != 1 || a.auction.page != 2 {
		t.Fatal("category scroll changed result paging")
	}
}
