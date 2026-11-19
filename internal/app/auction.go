package app

import (
	"crypto/rand"
	"fmt"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type auctionView struct {
	position                       ui.WindowPosition
	npc                            uint32
	sequence                       uint64
	sort                           int
	view, page, category, selected int
	exactType, exactID             uint8
	query, message, modal          string
	minLevel, maxLevel             string
	fields                         [3]string
	focus                          int
	duration                       uint8
	source                         proto.InventoryItem
	token                          string
	rows                           []proto.AuctionRow
	more, pending                  bool
	categoryOpen                   bool
	nextRefresh                    float64
}

func (a *App) auctionBoard(npc uint32) bool {
	e := a.state.Entities[npc]
	return e != nil && e.NpcType == 1 && a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}].MerchantType == proto.AuctionBoard
}

func (a *App) requestAuction(req proto.AuctionRequest) {
	v := &a.auction
	if a.world == nil || v.pending || v.npc == 0 {
		return
	}
	req.NPC, req.View, req.Page = v.npc, v.view, v.page
	a.auctionSequence++
	v.sequence = a.auctionSequence
	req.Sequence, req.Sort = v.sequence, v.sort
	if v.view == proto.AuctionBrowse {
		req.Category, req.Query = v.category, v.query
		minLevel, _ := strconv.Atoi(v.minLevel)
		maxLevel, _ := strconv.Atoi(v.maxLevel)
		if minLevel > 65535 || maxLevel > 65535 || maxLevel > 0 && minLevel > maxLevel {
			v.message = "Enter a valid level range."
			return
		}
		req.MinLevel, req.MaxLevel = minLevel, maxLevel
	}
	if req.Action != "post" {
		req.ItemType, req.ItemID = v.exactType, v.exactID
	}
	if packet := proto.AuctionPacket(req); packet != nil {
		if err := a.world.Send(packet); err != nil {
			a.backToLogin("Unable to contact the market.")
			return
		}
		v.pending = true
		v.nextRefresh = a.clock + 10
	}
}

func (a *App) applyAuction(p []byte) {
	var r proto.AuctionReply
	v := &a.auction
	if proto.DecodeAuction(p, 12002, &r) != nil || !v.pending || v.npc == 0 || r.Sequence != v.sequence {
		return
	}
	v.pending = false
	v.nextRefresh = a.clock + 15
	if r.NPC == 0 {
		v.message = r.Error
		v.nextRefresh = a.clock + 30
		return
	}
	if r.NPC != v.npc || r.View != v.view {
		return
	}
	var selected int64
	if row := v.row(); row != nil {
		selected = row.ID
	}
	v.page, v.more, v.rows = r.Page, r.More, r.Rows
	v.selected = -1
	for i, row := range v.rows {
		if row.ID == selected {
			v.selected = i
			break
		}
	}
	v.message = r.Message
	if r.Error != "" {
		v.message = r.Error
	}

}

func (a *App) beginAuction(npc uint32) {
	a.auction = auctionView{npc: npc, selected: -1, focus: -1}
	a.requestAuction(proto.AuctionRequest{})
}

func (a *App) auctionPanel(w, h int) (int, int, int) {
	width := max(380, min(710, w-24))
	x := max(8, (w-width)/4)
	if a.inventory.open && a.inventoryUI != nil {
		left, _ := a.inventoryUI.Origin(w, h)
		if left >= 520 {
			x = 8
			width = min(width, left-16)
		}
	}
	x, y := a.auction.position.Origin(w, h, width, 418, x, max(8, (h-418)/4))
	return x, y, width
}

func (a *App) auctionBox(x, y, w, h int, c [4]float32) {
	a.sprites.Draw(a.white, float32(x), float32(y), float32(w), float32(h), 0, 0, 1, 1, c)
}

var auctionPanelColor = [4]float32{.065, .095, .10, .98}
var auctionFieldColor = [4]float32{.12, .17, .18, 1}
var auctionSelectedColor = [4]float32{.24, .37, .35, 1}
var auctionTextColor = [4]float32{.90, .94, .92, 1}

type auctionButton struct {
	label   string
	x, y, w int
	action  string
}

func (v *auctionView) actions() []auctionButton {
	switch v.view {
	case proto.AuctionBrowse:
		return []auctionButton{{"Bid", 14, 351, 62, "bid"}, {"Buy Now", 82, 351, 82, "buy"}, {"Watch", 170, 351, 70, "watch"}, {"Item Search", 246, 351, 102, "exact"}, {"Details", 354, 351, 70, "details"}}
	case proto.AuctionSelling:
		return []auctionButton{{"Setup Auction", 14, 351, 115, "setup"}, {"Cancel Auction", 135, 351, 123, "cancel"}, {"Details", 264, 351, 72, "details"}}
	case proto.AuctionBidding:
		return []auctionButton{{"Buy Now", 14, 351, 85, "buy"}, {"Details", 105, 351, 72, "details"}}
	case proto.AuctionItems, proto.AuctionGold:
		return []auctionButton{{"Collect", 14, 351, 76, "collect"}, {"Collect All", 96, 351, 95, "collectall"}, {"Details", 197, 351, 72, "details"}}
	default:
		return []auctionButton{{"Refresh", 14, 351, 78, "refresh"}, {"Details", 98, 351, 72, "details"}, {"Delete", 176, 351, 70, "unwatch"}, {"Delete All", 252, 351, 84, "clearwatch"}, {"Buy Now", 342, 351, 82, "buy"}, {"Bid", 430, 351, 62, "bid"}}
	}
}

func (v *auctionView) row() *proto.AuctionRow {
	if v.selected < 0 || v.selected >= len(v.rows) {
		return nil
	}
	return &v.rows[v.selected]
}

func (v *auctionView) actionAvailable(action string) bool {
	r := v.row()
	switch action {
	case "setup", "refresh":
		return true
	case "collectall", "clearwatch":
		return len(v.rows) > 0 || v.page > 0
	case "buy", "bid":
		return r != nil && r.Active && !r.Mine && (action == "buy" || !r.HighBidder)
	case "cancel":
		return r != nil && r.Active && r.Mine
	case "watch":
		return r != nil && r.Active
	default:
		return r != nil
	}
}

func (a *App) auctionAction(action string) {
	v := &a.auction
	if !v.actionAvailable(action) {
		return
	}
	r := v.row()
	switch action {
	case "setup":
		v.modal, v.focus = "setup", -1
		v.fields = [3]string{"", "500", "1000"}
		v.message = ""
		v.source = proto.InventoryItem{}
		a.inventory.open = true
	case "refresh", "collectall", "clearwatch":
		if action == "refresh" {
			action = ""
		}
		a.requestAuction(proto.AuctionRequest{Action: action})
	case "exact":
		if r == nil {
			return
		}
		v.exactType, v.exactID, v.page, v.category = r.Type, r.TypeID, 0, 0
		v.query, v.minLevel, v.maxLevel = "", "", ""
		a.requestAuction(proto.AuctionRequest{})
	case "details", "bid", "buy", "cancel":
		v.modal, v.focus = action, -1
		if action == "bid" {
			v.fields[0] = strconv.FormatUint(uint64(r.Price)*110/100, 10)
			v.focus = 0
		}
	default:
		if r != nil {
			a.requestAuction(proto.AuctionRequest{Action: action, ID: r.ID, Version: r.Version})
		}
	}
}

func (a *App) confirmAuction() {
	v := &a.auction
	req := proto.AuctionRequest{Action: v.modal}
	if v.modal == "setup" {
		count, _ := strconv.ParseUint(v.fields[0], 10, 16)
		start, _ := strconv.ParseUint(v.fields[1], 10, 32)
		buy, _ := strconv.ParseUint(v.fields[2], 10, 32)
		source, ok := a.state.Inventory[[2]uint8{v.source.Bag, v.source.Slot}]
		if !ok || source != v.source || count == 0 || count > uint64(source.Count) || start < 500 || start > 3800000000 || buy <= start || buy > 4000000000 {
			v.message = "Choose an item and enter valid prices and quantity."
			return
		}
		req.Action, req.Count, req.Start, req.Buyout = "post", uint16(count), uint32(start), uint32(buy)
		req.Item, req.Duration, req.Source = &source, v.duration, v.token
	} else {
		r := v.row()
		if r == nil || v.modal == "details" {
			v.modal = ""
			v.focus = -1
			return
		}
		req.ID, req.Version = r.ID, r.Version
		if req.Action == "bid" {
			amount, _ := strconv.ParseUint(v.fields[0], 10, 32)
			if amount < uint64(r.Price)*110/100 || amount >= uint64(r.Buyout) {
				v.message = "Enter a bid between the minimum and the buyout."
				return
			}
			req.Amount = uint32(amount)
		}
		if req.Action == "buy" {
			req.Amount = r.Buyout

		}
	}
	v.modal, v.focus = "", -1
	a.requestAuction(req)
}

func (a *App) auctionSource() bool {
	v := &a.auction
	key := a.inventory.source
	item, ok := a.state.Inventory[key]
	if !ok || key[0] == 0 || a.inventory.pending {
		return false
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return false
	}
	v.source, v.token = item, fmt.Sprintf("%x", nonce)
	v.fields[0] = strconv.Itoa(int(item.Count))
	a.inventory.cancelItemPointer()
	return true
}

func editAuctionField(text *string, in *platform.Input, numeric bool) {
	if in.HasKey(platform.KeyBackspace) && len(*text) > 0 {
		_, n := utf8.DecodeLastRuneInString(*text)
		*text = (*text)[:len(*text)-n]
	}
	limit := 80
	if numeric {
		limit = 10
	}
	for _, r := range in.Chars {
		if len(*text)+utf8.RuneLen(r) <= limit && (numeric && r >= '0' && r <= '9' || !numeric && unicode.IsPrint(r)) {
			*text += string(r)
		}
	}
}

func (a *App) updateAuction(in *platform.Input, w, h int) bool {
	v := &a.auction
	e := a.state.Entities[v.npc]
	if e == nil || a.state.Self.Dead || a.shop.npcID != v.npc {
		*v = auctionView{}
		return false
	}
	dx, dz := e.X-a.wv.pos[0], e.Z-a.wv.pos[2]
	if dx*dx+dz*dz > 100 || in.HasKey(platform.KeyEscape) && v.modal == "" {
		*v = auctionView{}
		a.shop.npcID = 0
		if a.world != nil {
			_ = a.world.Send(proto.NpcTalkPacket(0))
		}
		return true
	}
	if v.pending && a.clock >= v.nextRefresh {
		v.pending = false
		v.message = "The market did not reply. Refresh before retrying."
		v.nextRefresh = a.clock + 15
	}
	if !v.pending && v.modal == "" && v.focus < 0 && a.clock >= v.nextRefresh {
		a.requestAuction(proto.AuctionRequest{})
	}
	x, y, width := a.auctionPanel(w, h)
	rx, ry := in.MouseX-x, in.MouseY-y
	if v.focus >= 0 {
		if v.modal == "setup" || v.modal == "bid" {
			editAuctionField(&v.fields[v.focus], in, true)
			if in.HasKey(platform.KeyEnter) {
				a.confirmAuction()
			}
		} else {
			switch v.focus {
			case 0:
				editAuctionField(&v.query, in, false)
			case 1:
				editAuctionField(&v.minLevel, in, true)
			case 2:
				editAuctionField(&v.maxLevel, in, true)
			}
			if in.HasKey(platform.KeyEnter) {
				v.focus, v.page = -1, 0
				v.exactType = 0
				a.requestAuction(proto.AuctionRequest{})
			}
		}
	}
	if in.HasKey(platform.KeyEscape) && v.modal != "" {
		v.modal, v.focus = "", -1
		return true
	}
	inside := rx >= 0 && ry >= 0 && rx < width && ry < 418
	if !inside {
		return v.focus >= 0 && !in.Pressed && !in.Clicked && (len(in.Chars) > 0 || len(in.Keys) > 0)
	}
	hit := func(bx, by, bw, bh int) bool { return rx >= bx && rx < bx+bw && ry >= by && ry < by+bh }
	if v.modal == "setup" && hit(15, 97, width-30, 36) && in.Clicked && a.inventory.dragging {
		a.auctionSource()
		return true
	}
	if v.pending {
		return true
	}
	if in.Wheel != 0 && v.modal == "" && ry >= 136 && ry < 342 {
		page := v.page
		if in.Wheel > 0 {
			page = max(0, page-1)
		} else if v.more {
			page++
		}
		if page != v.page {
			v.page, v.selected = page, -1
			a.requestAuction(proto.AuctionRequest{})
		}
		return true
	}
	if !in.Pressed {
		return true
	}
	if v.categoryOpen {
		v.categoryOpen = false
		if hit(14, 111, 142, len(proto.AuctionCategories)*25) {
			v.category = (ry - 111) / 25
			v.page, v.selected = 0, -1
			v.exactType = 0
			a.requestAuction(proto.AuctionRequest{})
		}
		return true
	}
	if v.modal != "" {
		switch {
		case hit(width-52, 10, 38, 26):
			v.modal, v.focus = "", -1
		case hit(15, 300, 118, 28):
			a.confirmAuction()
		case v.modal == "setup" && hit(15, 140, width-30, 105):
			v.focus = min(2, (ry-140)/35)
		case v.modal == "setup" && hit(15, 247, 210, 28):
			v.duration = (v.duration + 1) % 4
		case v.modal == "bid" && hit(15, 150, width-30, 28):
			v.focus = 0
		}
		return true
	}
	v.focus = -1
	switch {
	case hit(width-52, 10, 38, 26):
		*v = auctionView{}
		a.shop.npcID = 0
		if a.world != nil {
			_ = a.world.Send(proto.NpcTalkPacket(0))
		}
	case hit(14, 48, width-28, 28):
		idx := min(3, (rx-14)*4/(width-28))
		v.view = []int{proto.AuctionBrowse, proto.AuctionSelling, proto.AuctionItems, proto.AuctionWatch}[idx]
		v.page, v.selected = 0, -1
		a.requestAuction(proto.AuctionRequest{})
	case v.view == proto.AuctionBrowse && hit(14, 83, 116, 26):
		v.categoryOpen = true
	case v.view == proto.AuctionBrowse && hit(136, 83, width-370, 26):
		v.focus = 0
	case v.view == proto.AuctionBrowse && hit(width-228, 83, 64, 26):
		v.focus = 1
	case v.view == proto.AuctionBrowse && hit(width-158, 83, 64, 26):
		v.focus = 2
	case v.view == proto.AuctionBrowse && hit(width-88, 83, 74, 26):
		v.page, v.selected = 0, -1
		v.exactType = 0
		a.requestAuction(proto.AuctionRequest{})
	case (v.view == proto.AuctionSelling || v.view == proto.AuctionBidding || v.view == proto.AuctionItems || v.view == proto.AuctionGold) && hit(14, 83, 240, 26):
		switch v.view {
		case proto.AuctionSelling:
			v.view = proto.AuctionBidding
		case proto.AuctionBidding:
			v.view = proto.AuctionSelling
		case proto.AuctionItems:
			v.view = proto.AuctionGold
		case proto.AuctionGold:
			v.view = proto.AuctionItems
		}
		v.page, v.selected = 0, -1
		a.requestAuction(proto.AuctionRequest{})
	case v.view != proto.AuctionItems && v.view != proto.AuctionGold && hit(14, 113, width-28, 23):
		sort := 1
		if rx >= width/2+150 {
			sort = 4
		} else if rx >= width/2+52 {
			sort = 3
		} else if rx >= width/2 {
			sort = 2
		}
		if v.sort == sort {
			sort = -sort
		}
		v.sort, v.page = sort, 0
		a.requestAuction(proto.AuctionRequest{})
	case hit(14, 138, width-28, proto.AuctionPageSize*26):
		i := (ry - 138) / 26
		if i < len(v.rows) {
			v.selected = i
		}
	case hit(width-112, 382, 42, 24) && v.page > 0:
		v.page--
		v.selected = -1
		a.requestAuction(proto.AuctionRequest{})
	case hit(width-62, 382, 48, 24) && v.more:
		v.page++
		v.selected = -1
		a.requestAuction(proto.AuctionRequest{})
	default:
		for _, b := range v.actions() {
			if hit(b.x, b.y, b.w, 26) {
				a.auctionAction(b.action)
				break
			}
		}
	}
	return true
}

func auctionRemaining(ends int64) string {
	left := time.Until(time.Unix(ends, 0))
	if left <= 0 {
		return "Ended"
	}
	if left >= time.Hour {
		return fmt.Sprintf("%dh %02dm", int(left.Hours()), int(left.Minutes())%60)
	}
	return fmt.Sprintf("%dm", max(1, int(left.Minutes())))
}

func (a *App) drawAuction(w, h int) {
	v := &a.auction
	x, y, width := a.auctionPanel(w, h)
	a.clearTooltipUnderPanel(x, y, width, 418)
	a.auctionBox(x, y, width, 418, auctionPanelColor)
	a.auctionFrame(x, y, width)
	text := func(label string, rx, ry, maxWidth int) {
		a.font.Draw(a.sprites, label, x+rx, y+ry, auctionTextColor, maxWidth)
	}
	button := func(label string, rx, ry, bw int, selected bool) {
		color := auctionFieldColor
		if selected {
			color = auctionSelectedColor
		}
		a.auctionBox(x+rx, y+ry, bw, 26, color)
		text(label, rx+6, ry+5, bw-9)
	}
	text("Auction House", 14, 17, width-100)
	button("×", width-52, 10, 38, false)
	if v.modal != "" {
		titles := map[string]string{"setup": "Setup Auction", "bid": "Place Bid", "buy": "Buy Now", "cancel": "Cancel Auction", "details": "Item Details"}
		text(titles[v.modal], 14, 57, width-28)
		if v.modal == "setup" {
			label := "Drag an inventory item here"
			button(label, 15, 97, width-30, false)
			if def := a.itemsItem(v.source.Type, v.source.TypeID); def != nil {
				a.auctionBox(x+15, y+97, width-30, 26, auctionFieldColor)
				a.drawItemIcon(def, x+17, y+99, 22, auctionTextColor, &v.source)
				text(def.Name, 45, 102, width-60)
			}
			names := []string{"Quantity", "Starting Price", "Buyout Price"}
			for i, name := range names {
				button(name+": "+v.fields[i], 15, 140+i*35, width-30, v.focus == i)
			}
			button(fmt.Sprintf("Duration: %d hours", proto.AuctionHours(v.duration)), 15, 247, 210, false)
			start, _ := strconv.ParseUint(v.fields[1], 10, 32)
			text(fmt.Sprintf("Deposit: %d gold", proto.AuctionDeposit(uint32(start), v.duration)), 240, 251, width-254)
		} else if r := v.row(); r != nil {
			text(fmt.Sprintf("%s ×%d", r.Name, r.Count), 15, 96, width-30)
			text(fmt.Sprintf("Current: %d gold    Buyout: %d gold", r.Price, r.Buyout), 15, 123, width-30)
			if v.modal == "bid" {
				text(fmt.Sprintf("Minimum bid: %d gold", uint64(r.Price)*110/100), 15, 187, width-30)
				button("Bid: "+v.fields[0], 15, 150, width-30, true)
			}
			if v.modal == "cancel" {
				text("The item will move to Ended Auctions.", 15, 158, width-30)
			}
			if v.modal == "details" {
				text("Seller: "+r.Seller, 15, 156, width-30)
				if d := a.itemsItem(r.Type, r.TypeID); d != nil {
					a.drawItemDetails(d, &proto.InventoryItem{Type: r.Type, TypeID: r.TypeID, Count: r.Count, Quality: r.Quality, Gems: r.Gems, Craft: r.Craft}, r.Count, x+150, y+143, w, h, "")
				}
			}
		}
		label := "Confirm"
		if v.modal == "details" {
			label = "Back"
		}
		file, col, cell, artW, artH := "", 0, 128, 88, 30
		switch v.modal {
		case "setup":
			file = "auction_commission_button"
		case "bid":
			file, cell, artW, artH = "auction_tender_enter_button", 64, 47, 19
		case "details":
			file, col = "auction_commission_button", 1
		}
		if file == "" || !a.auctionSpriteButton(file, col, a.auctionButtonState(x+15, y+300, 118, true), cell, 32, artW, artH, x+15, y+300, 118) {
			button(label, 15, 300, 118, false)
		}
		text(v.message, 15, 385, width-30)
		return
	}
	tabs := []string{"Search Items", "Auction Progress", "Ended Auctions", "Interest List"}
	selected := 0
	switch v.view {
	case proto.AuctionSelling, proto.AuctionBidding:
		selected = 1
	case proto.AuctionItems, proto.AuctionGold:
		selected = 2
	case proto.AuctionWatch:
		selected = 3
	}
	for i, label := range tabs {
		tx := 14 + i*(width-28)/4
		tw := (width-28)/4 - 4
		state := a.auctionButtonState(x+tx, y+48, tw, true)
		if selected == i {
			state = 1
		}
		if !a.auctionSpriteButton("auction_top_manual_button", i, state, 128, 64, 126, 32, x+tx, y+48, tw) {
			button(label, tx, 48, tw, selected == i)
		}
	}
	if v.view == proto.AuctionBrowse {
		button(proto.AuctionCategories[v.category], 14, 83, 116, false)
		query := v.query
		if query == "" && v.focus != 0 {
			query = "Item or seller"
			if v.exactType != 0 {
				query = "Exact item"
			}
		}
		button(query, 136, 83, width-370, v.focus == 0)
		button("Min "+v.minLevel, width-228, 83, 64, v.focus == 1)
		button("Max "+v.maxLevel, width-158, 83, 64, v.focus == 2)
		if !a.auctionSpriteButton("auction_buy_search_botton", 0, a.auctionButtonState(x+width-88, y+83, 74, true), 64, 32, 47, 19, x+width-88, y+83, 74) {
			button("Search", width-88, 83, 74, false)
		}
	} else if selected == 1 || selected == 2 {
		name := map[int]string{proto.AuctionSelling: "Selling", proto.AuctionBidding: "Bidding", proto.AuctionItems: "Item Claims", proto.AuctionGold: "Gold Claims"}[v.view]
		button(name+"  /  Switch", 14, 83, 240, true)
	}
	text("Item", 16, 116, width/3)
	text("Level", width/2, 116, 46)
	heading := "Time left"
	if v.view == proto.AuctionItems || v.view == proto.AuctionGold {
		heading = "Claim by"
	}
	text(heading, width/2+52, 116, 90)
	text("Gold / Result", width/2+150, 116, width/2-165)
	for i, r := range v.rows {
		ry := 138 + i*26
		color := auctionFieldColor
		if v.selected == i {
			color = auctionSelectedColor
		}
		a.auctionBox(x+14, y+ry, width-28, 24, color)
		if d := a.itemsItem(r.Type, r.TypeID); d != nil {
			a.drawItemIcon(d, x+16, y+ry+1, 22, auctionTextColor)
		}
		text(fmt.Sprintf("%s ×%d", r.Name, r.Count), 44, ry+4, width/2-49)
		text(strconv.Itoa(int(r.Level)), width/2, ry+4, 46)
		remaining := auctionRemaining(r.Ends)
		if heading == "Claim by" && remaining == "Ended" {
			remaining = "Overdue"
		}
		text(remaining, width/2+52, ry+4, 90)
		value := strconv.FormatUint(uint64(r.Price), 10)
		if r.Gold > 0 {
			value = fmt.Sprintf("%d · %s", r.Gold, r.Reason)
		} else if r.Reason != "" {
			value = r.Reason
		}
		text(value, width/2+150, ry+4, width/2-165)
	}
	if len(v.rows) == 0 {
		text("No listings or results in this view.", 16, 155, width-32)
	}
	for _, b := range v.actions() {
		if b.x+b.w > width-12 {
			continue
		}
		enabled := v.actionAvailable(b.action)
		if file, col := v.auctionActionArt(b.action); file != "" &&
			a.auctionSpriteButton(file, col, a.auctionButtonState(x+b.x, y+b.y, b.w, enabled), 128, 32, 88, 30, x+b.x, y+b.y, b.w) {
			continue
		}
		if enabled {
			button(b.label, b.x, b.y, b.w, false)
		} else {
			a.auctionBox(x+b.x, y+b.y, b.w, 26, auctionFieldColor)
			a.font.Draw(a.sprites, b.label, x+b.x+6, y+b.y+5, [4]float32{.55, .60, .59, 1}, b.w-9)
		}
	}
	message := v.message
	if v.pending {
		message = "Loading market…"
	}
	if message == "" {
		message = fmt.Sprintf("Page %d", v.page+1)
	}
	text(message, 14, 387, width-135)
	button("Prev", width-112, 382, 42, false)
	button("Next", width-62, 382, 48, false)
	if v.categoryOpen {
		for i, name := range proto.AuctionCategories {
			rowY := 111 + i*25
			state := a.auctionButtonState(x+14, y+rowY, 142, true)
			if i == v.category {
				state = 1
			}
			if i == 0 || !a.auctionSpriteButton("auction_completion", i-1, state, 128, 32, 127, 28, x+14, y+rowY, 142) {
				button(name, 14, rowY, 142, i == v.category)
			}
		}
	}
}
