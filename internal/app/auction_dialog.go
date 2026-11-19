package app

import (
	"fmt"
	"image"
	"strconv"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type auctionBounds struct {
	x, y, width, height int
}

func (b auctionBounds) contains(x, y int) bool {
	return x >= b.x && y >= b.y && x < b.x+b.width && y < b.y+b.height
}

func (b auctionBounds) source() auctionBounds {
	return auctionBounds{b.x + 14, b.y + 43, b.width - 28, 33}
}

func auctionDialogBounds(x, y, width, panelHeight int, modal string) auctionBounds {
	height := 260
	switch modal {
	case "setup", "details":
		height = 330
	case "buy", "cancel":
		height = 204
	}
	dialogWidth := min(width-24, 322)
	return auctionBounds{x + (width-dialogWidth)/2, y + (panelHeight-height)/2, dialogWidth, height}
}

const auctionDurationY = 223

func auctionDialogFieldRow(modal string, i int) int {
	if modal == "bid" {
		return 142
	}
	return 88 + i*38
}

func (a *App) updateAuctionDialog(in *platform.Input, x, y, width, height int) {
	v := &a.auction
	if v.pending || !in.Pressed {
		return
	}
	b := auctionDialogBounds(x, y, width, height, v.modal)
	rx, ry := in.MouseX-b.x, in.MouseY-b.y
	hit := func(x, y, w, h int) bool {
		return (auctionBounds{x, y, w, h}).contains(rx, ry)
	}
	if v.keypad {
		a.updateAuctionKeypad(rx, ry+22, b.width)
		return
	}
	switch {
	case hit(b.width-39, 8, 26, 26), hit(b.width-118, b.height-39, 104, 26):
		v.modal, v.focus = "", -1
	case v.modal != "details" && hit(14, b.height-39, 110, 26):
		a.confirmAuction()
	case v.modal == "setup" && hit(14, auctionDurationY, b.width-28, 26):
		cell := (b.width - 28) / 4
		if offset := rx - 14; offset < cell*4 && offset%cell < cell-3 {
			v.duration = uint8(offset / cell)
		}
	case v.modal == "setup" || v.modal == "bid":
		count := 1
		if v.modal == "setup" {
			count = 3
		}
		for i := range count {
			row := auctionDialogFieldRow(v.modal, i)
			if hit(117, row, b.width-198, 26) || hit(b.width-77, row, 62, 26) {
				v.focus = i
				v.keypad = rx >= b.width-77
				return
			}
		}
	}
}

var auctionGoldText = [4]float32{1, .83, .38, 1}
var auctionLabelText = [4]float32{.80, .77, .68, 1}
var auctionWarningText = [4]float32{1, .60, .45, 1}

func (a *App) drawAuctionControl(label string, x, y, width int, selected, enabled bool) {
	color, ink := auctionFieldColor, auctionTextColor
	if !enabled {
		ink = [4]float32{.54, .56, .54, 1}
	} else if selected || a.auctionButtonState(x, y, width, true) == 1 {
		color, ink = auctionSelectedColor, auctionGoldText
	}
	a.auctionBox(x, y, width, 26, color)
	labelWidth := a.font.Width(label)
	scale := min(1, float32(width-8)/float32(max(1, labelWidth)))
	tx := float32(x) + (float32(width)-float32(labelWidth)*scale)/2
	a.font.DrawScaled(a.sprites, label, tx, float32(y+5), scale, ink, 0)
}

func (a *App) drawAuctionDialog(x, y, width, height, w, h int) {
	v := &a.auction
	b := auctionDialogBounds(x, y, width, height, v.modal)
	a.auctionBox(x, y, width, height, [4]float32{0, 0, 0, .48})
	a.auctionBox(b.x, b.y, b.width, b.height, auctionPanelColor)
	a.auctionBox(b.x, b.y, b.width, 36, [4]float32{.18, .15, .10, 1})
	art := map[string]string{"setup": "auction_commission", "bid": "auction_tender", "details": "auction_detailview"}[v.modal]
	hasTitleArt := art != "" && a.auctionArt(art, image.Rect(1, 1, 255, 37), b.x+(b.width-254)/2, b.y, 254, 36, 1)
	a.auctionBox(b.x, b.y+b.height-2, b.width, 2, [4]float32{.35, .29, .16, 1})
	text := func(label string, rx, ry int, color [4]float32) {
		a.font.Draw(a.sprites, label, b.x+rx, b.y+ry, color, b.width-rx-14)
	}
	control := func(label string, rx, ry, bw int, selected bool) {
		a.drawAuctionControl(label, b.x+rx, b.y+ry, bw, selected, !v.pending)
	}
	titles := map[string]string{"setup": "Setup Auction", "bid": "Place Bid", "buy": "Buy Now", "cancel": "Cancel Auction", "details": "Item Details"}
	if !hasTitleArt {
		text(titles[v.modal], 14, 11, auctionGoldText)
	}
	control("×", b.width-39, 8, 26, false)

	field := func(label, value string, i int) {
		row := auctionDialogFieldRow(v.modal, i)
		text(label, 15, row+5, auctionLabelText)
		fieldX := 117
		control(value, fieldX, row, b.width-fieldX-81, v.focus == i)
		a.drawAuctionInputButton(b.x, b.y, b.width, row, control)
	}
	if v.modal == "setup" {
		source := b.source()
		a.auctionBox(source.x, source.y, source.width, source.height, auctionFieldColor)
		if item := a.itemsItem(v.source.Type, v.source.TypeID); item != nil {
			a.drawItemIcon(item, source.x+5, source.y+4, 25, auctionTextColor, &v.source)
			text(item.Name, 48, 52, auctionTextColor)
		} else {
			text("Drag an inventory item here", 23, 52, auctionLabelText)
		}
		for i, label := range [...]string{"Quantity", "Starting price", "Buyout price"} {
			field(label, v.fields[i], i)
		}
		start, _ := strconv.ParseUint(v.fields[1], 10, 32)
		text("Duration", 14, 204, auctionLabelText)
		deposit := fmt.Sprintf("Deposit: %d gold", proto.AuctionDeposit(uint32(start), v.duration))
		text(deposit, max(95, b.width-14-a.font.Width(deposit)), 204, auctionGoldText)
		cell := (b.width - 28) / 4
		for i := range 4 {
			control(fmt.Sprintf("%d h", proto.AuctionHours(uint8(i))), 14+i*cell, auctionDurationY, cell-3, int(v.duration) == i)
		}
	} else if r := v.row(); r != nil {
		text(fmt.Sprintf("%s ×%d", r.Name, r.Count), 14, 49, auctionTextColor)
		if r.Reason == "" {
			text(fmt.Sprintf("Current bid: %d gold", r.Price), 14, 74, auctionGoldText)
			text(fmt.Sprintf("Buyout price: %d gold", r.Buyout), 14, 96, auctionGoldText)
		}
		switch v.modal {
		case "bid":
			text(fmt.Sprintf("Minimum bid: %d gold", uint64(r.Price)*110/100), 14, 118, auctionLabelText)
			field("Your bid", v.fields[0], 0)
		case "cancel":
			text("The item will move to Ended Auctions.", 14, 122, auctionWarningText)
		case "details":
			text("Seller: "+r.Seller, 14, 120, auctionLabelText)
			if r.Reason != "" {
				text("Result: "+r.Reason, 14, 143, auctionLabelText)
				text("Claim by: "+time.Unix(r.Ends, 0).Format(time.DateTime), 14, 166, auctionLabelText)
			}
			if r.Gold > 0 {
				text(fmt.Sprintf("Gold to collect: %d", r.Gold), 14, 190, auctionGoldText)
			} else if d := a.itemsItem(r.Type, r.TypeID); d != nil {
				a.drawItemDetails(d, &proto.InventoryItem{Type: r.Type, TypeID: r.TypeID, Count: r.Count, Quality: r.Quality, Gems: r.Gems, Craft: r.Craft}, r.Count, b.x+b.width+6, b.y+44, w, h, "")
			}
		}
	}
	confirm := map[string]string{"setup": "List item", "bid": "Place bid", "buy": "Buy now", "cancel": "Cancel auction"}
	footerY := b.height - 39
	setupButton := func(column, rx, bw int) bool {
		return v.modal == "setup" && a.auctionSpriteButton("auction_commission_button", column,
			a.auctionButtonState(b.x+rx, b.y+footerY, bw, !v.pending), 128, 32, 86, 30,
			b.x+rx, b.y+footerY, bw)
	}
	if v.modal != "details" && !setupButton(0, 14, 110) {
		control(confirm[v.modal], 14, footerY, 110, true)
	}
	closeLabel := "Back"
	if v.modal == "setup" || v.modal == "bid" {
		closeLabel = "Cancel"
	}
	if !setupButton(1, b.width-118, 104) {
		control(closeLabel, b.width-118, footerY, 104, false)
	}
	if v.message != "" {
		text(v.message, 14, b.height-61, auctionWarningText)
	}
	if v.keypad {
		a.drawAuctionKeypad(b.x, b.y-22, b.width, control)
	}
}
