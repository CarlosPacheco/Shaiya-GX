package app

import (
	"fmt"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/itemtype"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type marketDialog struct {
	op     proto.Opcode
	index  uint8
	item   proto.InventoryItem
	price  uint32
	seller uint32
}

type personalMarketView struct {
	selected int
	title    ui.TextField
	caret    ui.ChatCaret
	editing  bool
	dialog   marketDialog
	pending  proto.Opcode
	deadline float64
	purchase marketDialog
}

func (v *personalMarketView) modal() bool { return v.editing || v.dialog.op != 0 }

func (a *App) personalMarketOpen() bool {
	return a.personalMarketUI != nil && a.state != nil && (a.state.MyShop.Phase != proto.MyShopInactive || a.state.ShopView.Open)
}

func (a *App) sendPersonalMarket(p []byte) {
	if a.world == nil || a.personalMarket.pending != 0 {
		return
	}
	if err := a.world.Send(p); err != nil {
		a.backToLogin("Unable to send market request.")
		return
	}
	a.personalMarket.pending, a.personalMarket.deadline = proto.Op(p), a.clock+8
}

func (a *App) setupPersonalMarket() {
	if a.personalMarketUI == nil || a.state.Exchange.Active || a.state.ShopView.Open {
		return
	}
	if a.state.MyShop.Phase == proto.MyShopInactive {
		a.sendPersonalMarket(proto.NewWriter(proto.MyShopBegin).Bytes())
	} else {
		a.inventory.open = true
	}
}

func (a *App) viewPersonalMarket(id uint32) {
	if a.canViewPersonalMarket(id) {
		a.sendPersonalMarket(proto.MyShopEnterPacket(id))
	}
}

func (a *App) canViewPersonalMarket(id uint32) bool {
	if a.state.Self.Dead || a.state.Exchange.Active || a.state.MyShop.Phase != proto.MyShopInactive || id == a.state.SelfID {
		return false
	}
	e := a.state.Entities[id]
	if e == nil || e.Kind != game.KindUser || e.ShopTitle == "" {
		return false
	}
	if a.state.MapID != 42 && e.Shape != nil && e.Shape.Faction <= 1 && e.Shape.Faction != a.state.Faction.Faction {
		return false
	}
	x, z := a.state.Self.X, a.state.Self.Z
	if a.wv != nil {
		x, z = a.wv.pos[0], a.wv.pos[2]
	}
	dx, dz := x-e.X, z-e.Z
	return dx*dx+dz*dz <= proto.MyShopReach*proto.MyShopReach
}

func (a *App) personalMarketNotice(id int, fallback string, args map[string]string) {
	if id != 0 && a.msg.Get(id) != "" {
		a.systemMessage(7, id, args)
	} else {
		a.questMessage(fallback)
	}
}

func (a *App) confirmedPersonalMarket(op proto.Opcode) {
	v := &a.personalMarket
	if op == v.pending {
		v.pending = 0
	}
	s, buyer := a.state.MyShop, a.state.ShopView
	switch op {
	case proto.MyShopBegin:
		switch s.BeginResult {
		case proto.MyShopBeginOK:
			*v = personalMarketView{selected: -1, title: ui.TextField{Max: proto.MyShopTitleBytes}}
			a.inventory.open = true
			if a.personalMarketUI != nil {
				a.personalMarketUI.CancelHeld()
			}
		case proto.MyShopBeginWrongMap:
			id := 459
			if a.state.Faction.Faction == 1 {
				id = 460
			}
			a.personalMarketNotice(id, "Markets can be set up in Apulune, Iris and the Auction House.", nil)
		default:
			a.personalMarketNotice(0, "You cannot set up a market right now.", nil)
		}
	case proto.MyShopOpen:
		v.editing = false
		a.personalMarketNotice(409, "Market opened.", nil)
	case proto.MyShopStop:
		a.personalMarketNotice(410, "Market closed.", nil)
	case proto.MyShopCancel:
		*v = personalMarketView{selected: -1}
		a.personalMarketNotice(410, "Market closed.", nil)
	case proto.MyShopEnter:
		if buyer.EnterResult == proto.MyShopEnterOK {
			*v = personalMarketView{selected: -1}
			if a.personalMarketUI != nil {
				a.personalMarketUI.CancelHeld()
			}
		} else if buyer.EnterResult == proto.MyShopEnterFull {
			a.personalMarketNotice(0, "This market has too many buyers.", nil)
		} else {
			a.personalMarketNotice(0, "This market is unavailable.", nil)
		}
	case proto.MyShopSold:
		name := a.personalMarketItemName(s.SoldItem)
		gold := fmt.Sprint(s.SoldGold)
		a.personalMarketNotice(412, fmt.Sprintf("Sold %d × %s for %s gold.", s.SoldCount, name, gold), map[string]string{"p": fmt.Sprintf("%d × %s", s.SoldCount, name), "b": gold})
	case proto.MyShopBuy:
		buy := v.purchase
		v.purchase = marketDialog{}
		switch buyer.BuyResult {
		case proto.MyShopBuyOK:
			name := a.personalMarketItemName(buy.item)
			gold := fmt.Sprint(uint64(buy.price) * uint64(buy.item.Count))
			a.personalMarketNotice(415, fmt.Sprintf("Purchased %d × %s for %s gold.", buy.item.Count, name, gold), map[string]string{"p": fmt.Sprintf("%d × %s", buy.item.Count, name), "b": gold})
		case proto.MyShopBuyNoMoney:
			a.personalMarketNotice(0, "You do not have enough gold.", nil)
		case proto.MyShopBuySellerCap:
			a.personalMarketNotice(469, "The seller cannot hold any more gold.", nil)
		case proto.MyShopBuyNoRoom:
			a.personalMarketNotice(406, "Your inventory is full.", nil)
		default:
			a.personalMarketNotice(413, "Unable to purchase this item.", nil)
		}
	case proto.MyShopClosed:
		*v = personalMarketView{selected: -1}
		a.personalMarketNotice(447, "The market closed.", nil)
	case proto.MyShopLeave:
		*v = personalMarketView{selected: -1}
	}
	if v.selected >= 0 && (op == proto.MyShopRemove && !s.Slots[v.selected].Used || op == proto.MyShopStock && !buyer.Present[v.selected]) {
		v.selected = -1
	}
}

func (a *App) personalMarketItemName(item proto.InventoryItem) string {
	if item.Type != 0 && (a.arc != nil || a.items != nil) {
		if d := a.itemsItem(item.Type, item.TypeID); d != nil {
			return d.Name
		}
	}
	return "Item"
}

func (a *App) personalMarketTimeout() bool {
	v := &a.personalMarket
	if v.pending == 0 || a.clock < v.deadline {
		return false
	}
	if v.pending == proto.MyShopBuy {
		a.backToLogin("Market purchase response timed out. Reconnect to refresh your inventory.")
		return true
	}
	v.pending = 0
	a.personalMarketNotice(0, "The market request was not accepted.", nil)
	return false
}

func (a *App) updatePersonalMarket(in *platform.Input, w, h int) bool {
	if !a.personalMarketOpen() {
		return false
	}
	v := &a.personalMarket
	buyer := a.state.ShopView.Open
	x, y := a.personalMarketUI.Origin(w, h)
	capture := panelPointer(in, x, y, ui.PersonalMarketWidth, ui.PersonalMarketHeight)
	if v.pending != 0 {
		return capture
	}
	if in.RightPressed && !buyer && a.state.MyShop.Phase == proto.MyShopSetup {
		if i := ui.PersonalMarketCellAt(in.MouseX-x, in.MouseY-y); i >= 0 && a.state.MyShop.Slots[i].Used {
			a.sendPersonalMarket(proto.MyShopRemovePacket(uint8(i)))
			return true
		}
	}
	action := a.personalMarketUI.Update(in, w, h, buyer)
	if action == ui.PersonalMarketClose {
		op := proto.MyShopCancel
		if buyer {
			op = proto.MyShopLeave
		}
		a.sendPersonalMarket(proto.NewWriter(op).Bytes())
		return true
	}
	if buyer {
		if action == ui.PersonalMarketCancel {
			a.sendPersonalMarket(proto.NewWriter(proto.MyShopLeave).Bytes())
			return true
		}
		if action >= 0 && action < proto.MyShopSlots && a.state.ShopView.Present[action] {
			entry := a.state.ShopView.Items[action]
			v.selected = action
			v.dialog = marketDialog{op: proto.MyShopBuy, index: entry.Index, price: entry.Price, item: entry.Item, seller: a.state.ShopView.Seller}
			a.quantityUI.OpenWithValue(in.MouseX, in.MouseY, w, h, int(entry.Item.Count), 1)
		}
		return capture
	}
	if action == ui.PersonalMarketCancel {
		a.sendPersonalMarket(proto.NewWriter(proto.MyShopCancel).Bytes())
		return true
	}
	if a.state.MyShop.Phase == proto.MyShopSelling {
		if action == ui.PersonalMarketToggle {
			a.sendPersonalMarket(proto.NewWriter(proto.MyShopStop).Bytes())
		}
		return capture
	}
	switch action {
	case ui.PersonalMarketMessage:
		v.caret.End(v.title.Text)
		v.editing = true
	case ui.PersonalMarketToggle:
		if strings.TrimSpace(v.title.Text) == "" {
			a.personalMarketNotice(10001, "Enter a market message.", nil)
			return true
		}
		for _, item := range a.state.MyShop.Slots {
			if item.Used {
				a.sendPersonalMarket(proto.MyShopOpenPacket(strings.TrimSpace(v.title.Text)))
				return true
			}
		}
		a.personalMarketNotice(10002, "Add an item before opening the market.", nil)
	default:
		if action >= 0 && action < proto.MyShopSlots && a.state.MyShop.Slots[action].Used {
			entry := a.state.MyShop.Slots[action]
			v.selected = action
			v.dialog = marketDialog{op: proto.MyShopPrice, index: uint8(action), item: entry.Item}
			a.exchangeMoney.Open(w, h, entry.Price)
		}
	}
	return capture
}

func (a *App) dropPersonalMarketItem(in *platform.Input, w, h int) bool {
	if a.personalMarketUI == nil || a.state.MyShop.Phase != proto.MyShopSetup || !in.Clicked || !(a.inventory.dragging || a.inventory.selected) {
		return false
	}
	x, y := a.personalMarketUI.Origin(w, h)
	index := ui.PersonalMarketCellAt(in.MouseX-x, in.MouseY-y)
	if index < 0 {
		return false
	}
	key := a.inventory.source
	a.inventory.cancelItemPointer()
	v := &a.personalMarket
	if v.pending != 0 || a.inventory.pending {
		return true
	}
	item, ok := a.state.Inventory[key]
	if key[0] == 0 {
		a.personalMarketNotice(444, "Move the item to your bag first.", nil)
	} else if !ok || item.Count == 0 || itemtype.Real(item.Type) == 27 {
		a.personalMarketNotice(443, "This item cannot be listed.", nil)
	} else if a.state.MyShop.Slots[index].Used {
		a.personalMarketNotice(445, "That listing slot is occupied.", nil)
	} else {
		for _, listed := range a.state.MyShop.Slots {
			if listed.Used && key == [2]uint8{listed.Bag, listed.Slot} {
				a.personalMarketNotice(446, "That item is already listed.", nil)
				return true
			}
		}
		item.Bag, item.Slot = key[0], key[1]
		v.dialog = marketDialog{op: proto.MyShopAdd, index: uint8(index), item: item}
		a.exchangeMoney.Open(w, h, 1)
	}
	return true
}

func (a *App) updatePersonalMarketModal(in *platform.Input, w, h int) bool {
	v := &a.personalMarket
	if !a.personalMarketOpen() || a.state.Self.Dead {
		v.editing, v.dialog = false, marketDialog{}
		return false
	}
	if v.editing {
		x, y := a.personalMarketUI.Origin(w, h)
		if in.Pressed && !pointInside(in, x+16, y+62, 158, 57) {
			v.editing = false
			return false
		}
		if in.HasKey(platform.KeyEnter) || in.HasKey(platform.KeyEscape) {
			v.editing = false
			return true
		}
		v.title.EditAt(in, &v.caret)
		runes := []rune(v.title.Text)
		if len(runes) > proto.MyShopTitleMax {
			v.title.Text = string(runes[:proto.MyShopTitleMax])
			v.caret.End(v.title.Text)
		}
		return true
	}
	d := v.dialog
	switch d.op {
	case proto.MyShopBuy:
		b := &a.state.ShopView
		entry := b.Items[d.index]
		if !b.Open || b.Seller != d.seller || !b.Present[d.index] || entry.Price != d.price || entry.Item != d.item {
			v.dialog = marketDialog{}
			a.personalMarketNotice(413, "The selected listing changed.", nil)
			return true
		}
		count := a.quantityUI.Update(in)
		if count == 0 {
			return true
		}
		v.dialog = marketDialog{}
		if count > 0 {
			d.item.Count = uint16(count)
			v.purchase = d
			a.sendPersonalMarket(proto.MyShopBuyPacket(d.index, uint8(count)))
		}
	case proto.MyShopAdd, proto.MyShopPrice:
		price, done, cancel := a.exchangeMoney.Update(in)
		if !done {
			return true
		}
		if !cancel && (price == 0 || price > proto.MyShopPriceMax) {
			a.personalMarketNotice(0, "Enter a unit price between 1 and 4,000,000,000 gold.", nil)
			return true
		}
		v.dialog = marketDialog{}
		if cancel || a.state.MyShop.Phase != proto.MyShopSetup {
			return true
		}
		listing := a.state.MyShop.Slots[d.index]
		if d.op == proto.MyShopAdd {
			if !listing.Used && a.state.Inventory[[2]uint8{d.item.Bag, d.item.Slot}] == d.item {
				a.sendPersonalMarket(proto.MyShopAddPacket(d.index, d.item.Bag, d.item.Slot, uint32(price)))
			}
		} else if listing.Used && listing.Item == d.item {
			a.sendPersonalMarket(proto.MyShopPricePacket(d.index, uint32(price)))
		}
	}
	return d.op != 0
}
