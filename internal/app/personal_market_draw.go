package app

import (
	"fmt"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *App) drawPersonalMarket(w, h int) {
	if !a.personalMarketOpen() {
		return
	}
	v, p := &a.personalMarket, a.personalMarketUI
	s, b := &a.state.MyShop, &a.state.ShopView
	title := v.title.Text
	if b.Open {
		if e := a.state.Entities[b.Seller]; e != nil {
			title = e.ShopTitle
		}
	} else if s.Phase == proto.MyShopSelling {
		title = a.state.Self.ShopTitle
	} else if v.editing {
		title += "|"
	} else if title == "" {
		title = "Click to enter a message"
	}
	var items [proto.MyShopSlots]proto.InventoryItem
	var prices [proto.MyShopSlots]uint32
	for i := range items {
		if b.Open && b.Present[i] {
			items[i], prices[i] = b.Items[i].Item, b.Items[i].Price
		} else if !b.Open && s.Slots[i].Used {
			entry := s.Slots[i]
			items[i], prices[i] = a.state.Inventory[[2]uint8{entry.Bag, entry.Slot}], entry.Price
		}
	}
	x, y := p.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.PersonalMarketWidth, ui.PersonalMarketHeight)
	p.Draw(a.sprites, a.font, w, h, title, b.Open, s.Phase == proto.MyShopSelling, v.pending != 0)
	hover := ui.PersonalMarketCellAt(a.lastMouseX-x, a.lastMouseY-y)
	for i, it := range items {
		if it.Type == 0 {
			continue
		}
		def := a.itemsItem(it.Type, it.TypeID)
		if def == nil {
			continue
		}
		cx, cy := ui.PersonalMarketCell(i)
		a.drawItemIcon(def, x+cx, y+cy, 32, white, &it)
		if count := a.itemCount(it.Type, it.TypeID, it.Count); count > 1 {
			text := strconv.Itoa(int(count))
			a.font.Draw(a.sprites, text, x+cx+31-a.font.Width(text), y+cy+18, white, 32)
		}
		if hover == i && !v.modal() && !a.inventory.dragging {
			a.drawItemDetails(def, &it, it.Count, x+ui.PersonalMarketWidth+5, y+cy, w, h, fmt.Sprintf("%d gold per unit", prices[i]))
		}
	}
}

func (a *App) drawPersonalMarketModal(w int) {
	if !a.personalMarketOpen() {
		return
	}
	d := a.personalMarket.dialog
	switch d.op {
	case proto.MyShopBuy:
		a.tooltip = nil
		q := a.quantityUI
		q.Draw(a.sprites, a.font)
		text := fmt.Sprintf("Total: %d G", uint64(q.Value())*uint64(d.price))
		a.drawMarketDialogCaption(text, q.X, q.Y, ui.QuantityWidth, w)
	case proto.MyShopAdd, proto.MyShopPrice:
		a.tooltip = nil
		m := a.exchangeMoney
		m.Draw(a.sprites, a.font)
		a.drawMarketDialogCaption("Unit price (gold)", m.X, m.Y, 200, w)
	}
}

func (a *App) drawMarketDialogCaption(text string, x, y, width, w int) {
	y = max(y-a.font.Height-4, 0)
	width = min(w-x, width)
	a.sprites.Draw(a.white, float32(x), float32(y), float32(width), float32(a.font.Height+4), 0, 0, 1, 1, ui.RGBColor(0x24201a))
	a.font.Draw(a.sprites, text, x+6, y+2, white, max(0, width-12))
}

func (a *App) drawPersonalMarketSigns(w, h int) {
	if len(a.state.ShopSigns) == 0 || a.wv == nil || a.scene3d == nil {
		return
	}
	f := a.chatFont()
	vp := a.scene3d.View.Mul(a.scene3d.Proj)
	for id, title := range a.state.ShopSigns {
		pos, ok := a.effectPosition(id)
		if !ok || pos.Sub(a.wv.pos).Len() > 50 {
			continue
		}
		pos = pos.Add(render.V3{0, 1.5, 0})
		if actor := a.playerActor(id); actor != nil {
			if anchor, ok := actor.NameAnchor(); ok {
				pos = anchor
			}
		}
		x, y, _, ok := projectLabel(pos, vp, w, h)
		if !ok {
			continue
		}
		// 0x44c12a: the common bubble routine 0x41c5c0 with its own colours.
		textWidth := min(292, f.Width(title))
		width, height := textWidth+20, 30
		left, top := x-width/2, y-height-40
		if a.scene3d.OccludedLabel(pos, left, top, width, height, w, h) {
			continue
		}
		a.sprites.Draw(a.white, float32(left), float32(top), float32(width), float32(height), 0, 0, 1, 1, [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 154.0 / 255}) // 0x44c186: 9a1d1f24
		f.Draw(a.sprites, title, left+10, top+(height-f.Height)/2, ui.RGBColor(0xccff66), textWidth)                                                                   // 0x44c181: ffccff66
	}
}
