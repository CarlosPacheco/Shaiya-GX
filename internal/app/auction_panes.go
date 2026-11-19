package app

import (
	"fmt"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

const auctionPaneRowY = 114

type auctionPane struct {
	view, x, width int
	title          string
	actions        []auctionButton
}

func auctionPanes(first, width int) [2]auctionPane {
	gutter := 10
	left := (width - 28 - gutter) / 2
	panes := [2]auctionPane{
		{view: first, x: 14, width: left, title: "Selling", actions: []auctionButton{{label: "List item", action: "setup"}, {label: "Cancel", action: "cancel"}}},
		{view: first + 1, x: 14 + left + gutter, width: width - 28 - gutter - left, title: "Bidding", actions: []auctionButton{{label: "Buy now", action: "buy"}}},
	}
	if first == proto.AuctionItems {
		panes[0].title, panes[1].title = "Item claims", "Gold claims"
		for i := range panes {
			panes[i].actions = []auctionButton{{label: "Collect", action: "collect"}, {label: "Collect all", action: "collectall"}}
		}
	}
	for i := range panes {
		p := &panes[i]
		p.actions = append(p.actions, auctionButton{label: "Details", action: "details"})
		cell := p.width / len(p.actions)
		for j := range p.actions {
			p.actions[j].x = p.x + j*cell
			p.actions[j].y = 330
			p.actions[j].w = cell - 3
		}
	}
	return panes
}

func (a *App) updateAuctionPanes(rx, ry, width, height int, wheel float64) {
	pitch, extra := auctionRowPitch(height), height-auctionMinHeight
	v := &a.auction
	for _, pane := range auctionPanes(auctionPair(v.view), width) {
		inside := func(x, y, w, h int) bool {
			return rx >= x && rx < x+w && ry >= y && ry < y+h
		}
		if rx < pane.x || rx >= pane.x+pane.width {
			continue
		}
		switch {
		case wheel != 0 && (ry < auctionPaneRowY || ry >= auctionPaneRowY+proto.AuctionPageSize*pitch):
			return
		case wheel != 0 || inside(pane.x, 362+extra, 28, 26) || inside(pane.x+pane.width-28, 362+extra, 28, 26):
			v.selectPane(pane.view)
			page := v.page
			if wheel > 0 || wheel == 0 && rx < pane.x+28 {
				page = max(0, page-1)
			} else if v.more {
				page++
			}
			if page != v.page {
				v.page, v.selected = page, -1
				a.requestAuction(proto.AuctionRequest{})
			}
		case inside(pane.x, auctionPaneRowY, pane.width, proto.AuctionPageSize*pitch):
			v.selectPane(pane.view)
			if index := (ry - auctionPaneRowY) / pitch; index < len(v.rows) {
				v.selected = index
			}
		default:
			for _, b := range pane.actions {
				if inside(b.x, b.y+extra, b.w, 26) {
					v.selectPane(pane.view)
					a.auctionAction(b.action)
					return
				}
			}
		}
		return
	}
}

func (a *App) drawAuctionPanes(x, y, width, height int, text func(string, int, int, int)) {
	pitch, extra := auctionRowPitch(height), height-auctionMinHeight
	v := a.auction
	for _, pane := range auctionPanes(auctionPair(v.view), width) {
		v.selectPane(pane.view)
		a.auctionBox(x+pane.x, y+82, pane.width, 30, auctionFieldColor)
		a.font.Draw(a.sprites, pane.title, x+pane.x+7, y+89, auctionGoldText, pane.width-14)
		if len(v.rows) == 0 {
			text("No entries", pane.x+7, 125, pane.width-14)
		}
		for i, r := range v.rows {
			ry := auctionPaneRowY + i*pitch
			color := auctionFieldColor
			if i == v.selected {
				color = auctionSelectedColor
			}
			a.auctionBox(x+pane.x, y+ry, pane.width, pitch-2, color)
			left := pane.x + 6
			if item := a.itemsItem(r.Type, r.TypeID); item != nil {
				a.drawItemIcon(item, x+left, y+ry+(pitch-20)/2, 20, auctionTextColor)
				left += 24
			}
			value := auctionRemaining(r.Ends)
			label := fmt.Sprintf("%s ×%d", r.Name, r.Count)
			right := pane.x + pane.width - 6
			if r.Reason != "" && value == "Ended" {
				value = "Overdue"
			}
			timeX := right - a.font.Width(value)
			text(value, timeX, ry+(pitch-18)/2, right-timeX)
			if r.Gold > 0 {
				label = strconv.FormatUint(uint64(r.Gold), 10) + " · " + r.Reason
			} else if r.Reason == "" && r.Price > 0 {
				price := strconv.FormatUint(uint64(r.Price), 10)
				priceX := timeX - a.font.Width(price) - 8
				if priceX > left+36 {
					text(price, priceX, ry+(pitch-18)/2, timeX-priceX-8)
					timeX = priceX
				}
			}
			if space := timeX - left - 5; space > 0 {
				text(label, left, ry+(pitch-18)/2, space)
			}
		}
		for _, b := range pane.actions {
			enabled := !v.pending && v.actionAvailable(b.action)
			file, col := v.auctionActionArt(b.action)
			if b.action != "details" && file != "" &&
				a.auctionSpriteButton(file, col, a.auctionButtonState(x+b.x, y+b.y+extra, b.w, enabled), 128, 32, 88, 30, x+b.x, y+b.y+extra, b.w) {
				continue
			}
			a.drawAuctionControl(b.label, x+b.x, y+b.y+extra, b.w, false, enabled)
		}
		a.drawAuctionControl("‹", x+pane.x, y+362+extra, 28, false, !v.pending && v.page > 0)
		a.drawAuctionControl("›", x+pane.x+pane.width-28, y+362+extra, 28, false, !v.pending && v.more)
		text(fmt.Sprintf("Page %d", v.page+1), pane.x+36, 366+extra, pane.width-72)
	}
	message := v.message
	if v.pending {
		message = "Loading market…"
	}
	text(message, 14, 394+extra, width-28)
}
