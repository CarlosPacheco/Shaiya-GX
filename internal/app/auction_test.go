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
	x, y, _ := a.auctionPanel(1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 50, MouseY: y + 110, Clicked: true}, 1024, 768)
	if a.auction.source != a.state.Inventory[key] || a.auction.fields[0] != "9" || a.auction.token == "" {
		t.Fatal("auction did not accept inventory drop", a.auction.source)
	}
}
