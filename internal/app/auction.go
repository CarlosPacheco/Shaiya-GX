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

type auctionList struct {
	page, selected int
	rows           []proto.AuctionRow
	more           bool
}

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
	keypad                         bool
	nextRefresh                    float64
	other                          auctionList
	requestView                    int
}

func (a *App) auctionBoard(npc uint32) bool {
	e := a.state.Entities[npc]
	return e != nil && e.NpcType == 1 && a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}].MerchantType == proto.AuctionBoard
}

func (a *App) requestAuction(req proto.AuctionRequest) {
	a.sendAuction(req, a.auction.view, a.auction.page)
}

func (a *App) sendAuction(req proto.AuctionRequest, view, page int) {
	v := &a.auction
	if a.world == nil || v.pending || v.npc == 0 {
		return
	}
	req.NPC, req.View, req.Page = v.npc, view, page
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
		v.requestView = view
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
	if r.NPC != v.npc || r.View != v.requestView {
		return
	}
	if r.View != v.view {
		v.other.replace(r)
		if r.Error != "" {
			v.message = r.Error
		}
		return
	}
	list := v.list()
	list.replace(r)
	v.setList(list)
	v.message = r.Message
	if r.Error != "" {
		v.message = r.Error
		return
	}
	if other := v.otherView(); other >= 0 {
		a.sendAuction(proto.AuctionRequest{}, other, v.other.page)
	}
}

func (l *auctionList) replace(r proto.AuctionReply) {
	var id int64
	if l.selected >= 0 && l.selected < len(l.rows) {
		id = l.rows[l.selected].ID
	}
	l.page, l.more, l.rows, l.selected = r.Page, r.More, r.Rows, -1
	for i, row := range l.rows {
		if id != 0 && row.ID == id {
			l.selected = i
			break
		}
	}
}

func (v *auctionView) list() auctionList {
	return auctionList{page: v.page, selected: v.selected, rows: v.rows, more: v.more}
}

func (v *auctionView) setList(l auctionList) {
	v.page, v.selected, v.rows, v.more = l.page, l.selected, l.rows, l.more
}

func (v *auctionView) otherView() int {
	first := auctionPair(v.view)
	if first < 0 {
		return -1
	}
	return first + (first + 1 - v.view)
}

func (v *auctionView) selectPane(view int) {
	if view != v.view {
		list := v.list()
		v.view = view
		v.setList(v.other)
		v.other = list
	}
}

func (a *App) beginAuction(npc uint32) {
	a.auction = auctionView{npc: npc, selected: -1, focus: -1}
	a.requestAuction(proto.AuctionRequest{})
}

const auctionMinHeight = 418

func auctionHeight(viewportHeight int) int {
	return max(auctionMinHeight, min(586, viewportHeight-64))
}

func auctionRowPitch(height int) int {
	return 26 + (height-auctionMinHeight)/proto.AuctionPageSize
}

func (a *App) auctionPanel(w, h int) (int, int, int) {
	available := w
	if a.inventory.open && a.inventoryUI != nil {
		left, _ := a.inventoryUI.Origin(w, h)
		if left >= 396 {
			available = left
		}
	}
	width := max(380, min(710, available-16))
	height := auctionHeight(h)
	x, y := a.auction.position.Origin(w, h, width, height, (available-width)/2, (h-height)/2)
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

func (v *auctionView) actions(width int) []auctionButton {
	buttons := v.actionList()
	end := buttons[len(buttons)-1]
	if edge := end.x + end.w - 14; edge > width-28 {
		for i := range buttons {
			b := &buttons[i]
			b.x = 14 + (b.x-14)*(width-28)/edge
			b.w = b.w * (width - 28) / edge
		}
	}
	return buttons
}

func (v *auctionView) actionList() []auctionButton {
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
		v.keypad = false
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
		v.modal, v.focus, v.keypad = action, -1, false
		v.message = ""
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
	if in.HasKey(platform.KeyDelete) {
		*text = ""
	}
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
	height := auctionHeight(h)
	rowPitch := auctionRowPitch(height)
	extra := height - auctionMinHeight
	rx, ry := in.MouseX-x, in.MouseY-y
	if v.focus >= 0 && !v.pending {
		if v.modal == "setup" || v.modal == "bid" {
			editAuctionField(&v.fields[v.focus], in, true)
			if in.HasKey(platform.KeyEnter) {
				if v.keypad {
					v.keypad = false
				} else {
					a.confirmAuction()
				}
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
		if v.keypad {
			v.keypad = false
		} else {
			v.modal, v.focus = "", -1
		}
		return true
	}
	inside := rx >= 0 && ry >= 0 && rx < width && ry < height
	if !inside {
		return v.focus >= 0 && !in.Pressed && !in.Clicked && (len(in.Chars) > 0 || len(in.Keys) > 0)
	}
	hit := func(bx, by, bw, bh int) bool { return rx >= bx && rx < bx+bw && ry >= by && ry < by+bh }
	if v.modal != "" {
		a.updateAuctionDialog(in, x, y, width, height)
		return true
	}
	if v.pending {
		return true
	}
	if v.modal == "" && auctionPair(v.view) >= 0 && ry >= 81 &&
		(in.Pressed || in.Wheel != 0) {
		a.updateAuctionPanes(rx, ry, width, height, in.Wheel)
		return true
	}
	if in.Wheel != 0 && rx >= auctionListX(v.view, width) && ry >= 138 && ry < 138+proto.AuctionPageSize*rowPitch {
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

	v.focus = -1
	listX := auctionListX(v.view, width)
	switch {
	case hit(width-52, 10, 38, 26):
		*v = auctionView{}
		a.shop.npcID = 0
		if a.world != nil {
			_ = a.world.Send(proto.NpcTalkPacket(0))
		}
	case hit(14, 48, width-28, 28):
		idx := min(3, (rx-14)*4/(width-28))
		next := [...]int{proto.AuctionBrowse, proto.AuctionSelling, proto.AuctionItems, proto.AuctionWatch}[idx]
		if next == v.view || next == auctionPair(v.view) {
			return true
		}
		v.view, v.other = next, auctionList{selected: -1}
		v.setList(auctionList{selected: -1})
		v.sort = 0
		a.requestAuction(proto.AuctionRequest{})
	case v.view == proto.AuctionBrowse && ry >= 83 && ry < 109:
		for _, field := range auctionBrowseControls(width) {
			if !hit(field.x, 83, field.width, 26) {
				continue
			}
			switch field.id {
			case -1:
				v.page, v.selected, v.exactType = 0, -1, 0
				a.requestAuction(proto.AuctionRequest{})
			default:
				v.focus = field.id
			}
			break
		}
	case v.view == proto.AuctionBrowse && hit(14, 138, listX-21, len(proto.AuctionCategories)*26):
		v.category = (ry - 138) / 26
		v.page, v.selected, v.exactType = 0, -1, 0
		a.requestAuction(proto.AuctionRequest{})
	case hit(listX, 113, width-listX-14, 23):
		sort := 1
		for i, column := range auctionColumns(listX, width) {
			if rx >= column.x {
				sort = i + 1
			}
		}
		if v.sort == sort {
			sort = -sort
		}
		v.sort, v.page = sort, 0
		a.requestAuction(proto.AuctionRequest{})
	case hit(listX, 138, width-listX-14, proto.AuctionPageSize*rowPitch):
		i := (ry - 138) / rowPitch
		if i < len(v.rows) {
			v.selected = i
		}
	case hit(width-112, 382+extra, 42, 26) && v.page > 0:
		v.page--
		v.selected = -1
		a.requestAuction(proto.AuctionRequest{})
	case hit(width-62, 382+extra, 48, 26) && v.more:
		v.page++
		v.selected = -1
		a.requestAuction(proto.AuctionRequest{})
	default:
		for _, b := range v.actions(width) {
			if hit(b.x, b.y+extra, b.w, 26) {
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
	height := auctionHeight(h)
	rowPitch := auctionRowPitch(height)
	extra := height - auctionMinHeight
	a.clearTooltipUnderPanel(x, y, width, height)
	a.auctionBox(x, y, width, height, auctionPanelColor)
	a.auctionFrame(x, y, width, height)
	text := func(label string, rx, ry, maxWidth int) {
		a.font.Draw(a.sprites, label, x+rx, y+ry, auctionTextColor, maxWidth)
	}
	button := func(label string, rx, ry, bw int, selected bool) {
		a.drawAuctionControl(label, x+rx, y+ry, bw, selected, true)
	}
	a.font.Draw(a.sprites, "Auction House", x+14, y+17, auctionGoldText, width-100)
	button("×", width-52, 10, 38, false)
	if v.modal != "" {
		defer a.drawAuctionDialog(x, y, width, height, w, h)
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
		query := v.query
		if query == "" && v.focus != 0 {
			query = "Item or seller"
			if v.exactType != 0 {
				query = "Exact item"
			}
		}
		labels := [...]string{query, "Min level " + v.minLevel, "Max level " + v.maxLevel, "Search"}
		for i, field := range auctionBrowseControls(width) {
			if field.id == -1 && a.auctionSpriteButton("auction_buy_search_botton", 0,
				a.auctionButtonState(x+field.x, y+83, field.width, !v.pending),
				64, 32, 47, 19, x+field.x, y+83, field.width) {
				continue
			}
			button(labels[i], field.x, 83, field.width, field.id >= 0 && v.focus == field.id)
		}
		a.drawAuctionCategories(x, y, width)
	}
	if auctionPair(v.view) >= 0 {
		a.drawAuctionPanes(x, y, width, height, text)
		return
	}
	listX := auctionListX(v.view, width)
	listWidth := width - listX - 14
	columns := auctionColumns(listX, width)
	a.auctionBox(x+listX, y+138, listWidth, proto.AuctionPageSize*rowPitch, [4]float32{.09, .09, .086, 1})
	for i := range proto.AuctionPageSize {
		a.auctionBox(x+listX, y+137+(i+1)*rowPitch, listWidth, 1, [4]float32{.22, .19, .14, 1})
	}
	for i, column := range columns {
		label := column.label
		if v.sort == i+1 {
			label += " ↑"
		} else if v.sort == -(i + 1) {
			label += " ↓"
		}
		a.font.Draw(a.sprites, label, x+column.x, y+116, auctionLabelText, column.width)
	}
	for i, r := range v.rows {
		ry := 138 + i*rowPitch
		lineY := ry + (rowPitch-18)/2
		color := auctionFieldColor
		if v.selected == i {
			color = auctionSelectedColor
		}
		a.auctionBox(x+listX, y+ry, listWidth, rowPitch-2, color)
		labelX := columns[0].x
		if d := a.itemsItem(r.Type, r.TypeID); d != nil {
			a.drawItemIcon(d, x+labelX, y+ry+(rowPitch-22)/2, 22, auctionTextColor)
			labelX += 25
		}
		if space := columns[0].x + columns[0].width - labelX; space > 0 {
			text(fmt.Sprintf("%s ×%d", r.Name, r.Count), labelX, lineY, space)
		}
		text(strconv.Itoa(int(r.Level)), columns[1].x, lineY, columns[1].width)
		text(auctionRemaining(r.Ends), columns[2].x, lineY, columns[2].width)
		value := strconv.FormatUint(uint64(r.Price), 10)
		if r.Gold > 0 {
			value = fmt.Sprintf("%d · %s", r.Gold, r.Reason)
		} else if r.Reason != "" {
			value = r.Reason
		}
		a.font.Draw(a.sprites, value, x+columns[3].x, y+lineY, auctionGoldText, columns[3].width)
	}
	if len(v.rows) == 0 {
		text("No listings or results in this view.", listX+6, 145, listWidth-12)
	}
	for _, b := range v.actions(width) {
		if b.x+b.w > width-12 {
			continue
		}
		enabled := v.actionAvailable(b.action)
		if file, col := v.auctionActionArt(b.action); file != "" &&
			a.auctionSpriteButton(file, col, a.auctionButtonState(x+b.x, y+b.y+extra, b.w, enabled), 128, 32, 88, 30, x+b.x, y+b.y+extra, b.w) {
			continue
		}
		if enabled {
			button(b.label, b.x, b.y+extra, b.w, false)
		} else {
			a.auctionBox(x+b.x, y+b.y+extra, b.w, 26, auctionFieldColor)
			a.font.Draw(a.sprites, b.label, x+b.x+6, y+b.y+extra+5, [4]float32{.55, .60, .59, 1}, b.w-9)
		}
	}
	message := v.message
	if v.pending {
		message = "Loading market…"
	}
	if message == "" {
		message = fmt.Sprintf("Page %d", v.page+1)
	}
	if v.view == proto.AuctionBrowse {
		gold := "Gold  " + strconv.FormatUint(uint64(a.state.Details.Money), 10)
		a.font.Draw(a.sprites, gold, x+14, y+387+extra, auctionGoldText, 126)
		text(message, 145, 387+extra, width-272)
	} else {
		text(message, 14, 387+extra, width-135)
	}
	a.drawAuctionControl("Prev", x+width-112, y+382+extra, 42, false, !v.pending && v.page > 0)
	a.drawAuctionControl("Next", x+width-62, y+382+extra, 48, false, !v.pending && v.more)
}

func auctionListX(view, width int) int {
	if view == proto.AuctionBrowse {
		return 14 + min(136, width/4) + 7
	}
	return 14
}

func (a *App) drawAuctionCategories(x, y, width int) {
	railWidth := auctionListX(proto.AuctionBrowse, width) - 21
	a.font.Draw(a.sprites, "Categories", x+14, y+116, auctionLabelText, railWidth)
	for i, name := range proto.AuctionCategories {
		rowY := 138 + i*26
		selected := i == a.auction.category
		state := a.auctionButtonState(x+14, y+rowY, railWidth, true)
		if selected {
			a.auctionBox(x+14, y+rowY, railWidth, 26, auctionSelectedColor)
		}
		if i == 0 || !a.auctionSpriteButton("auction_completion", i-1, state, 128, 32, 127, 28, x+14, y+rowY, railWidth) {
			a.drawAuctionControl(name, x+14, y+rowY, railWidth, selected, true)
		}
	}
}

type auctionBrowseControl struct {
	id, x, width int
}

func auctionBrowseControls(width int) [4]auctionBrowseControl {
	level := min(84, width/5)
	search := 64
	return [4]auctionBrowseControl{
		{0, 14, width - 46 - 2*level - search},
		{1, width - 90 - 2*level, level},
		{2, width - 84 - level, level},
		{-1, width - 78, search},
	}
}

type auctionColumn struct {
	label    string
	x, width int
}

func auctionColumns(listX, width int) [4]auctionColumn {
	priceWidth := min(102, (width-listX-16)/4)
	priceX := width - 16 - priceWidth
	endsX := priceX - 52
	levelX := endsX - 38
	return [4]auctionColumn{
		{"Item", listX + 4, levelX - listX - 8},
		{"Level", levelX, 34},
		{"Time left", endsX, 48},
		{"Price", priceX, priceWidth},
	}
}

func auctionPair(view int) int {
	switch view {
	case proto.AuctionSelling, proto.AuctionBidding:
		return proto.AuctionSelling
	case proto.AuctionItems, proto.AuctionGold:
		return proto.AuctionItems
	default:
		return -1
	}
}

var auctionKeys = [...]string{"7", "8", "9", "4", "5", "6", "1", "2", "3", "Clear", "0", "⌫"}

func (a *App) updateAuctionKeypad(rx, ry, width int) {
	v := &a.auction
	kx, ky := width-242, 98
	if rx < kx || rx >= kx+216 || ry < ky || ry >= ky+178 {
		v.keypad = false
		return
	}
	i := (ry-ky)/36*3 + (rx-kx)/72
	field := &v.fields[v.focus]
	switch {
	case i >= len(auctionKeys):
		v.keypad = false
	case i == 9:
		*field = ""
	case i == 11:
		if len(*field) > 0 {
			*field = (*field)[:len(*field)-1]
		}
	default:
		if len(*field) < 10 {
			*field += auctionKeys[i]
		}
	}
}

func (a *App) drawAuctionKeypad(x, y, width int, button func(string, int, int, int, bool)) {
	kx, ky := width-242, 98
	a.auctionBox(x+kx-4, y+ky-32, 224, 210, auctionPanelColor)
	button(a.auction.fields[a.auction.focus], kx, ky-28, 216, true)
	for i, key := range auctionKeys {
		button(key, kx+(i%3)*72, ky+(i/3)*36, 68, false)
	}
	button("Done", kx, ky+144, 216, false)
}

func (a *App) drawAuctionInputButton(x, y, width, row int, button func(string, int, int, int, bool)) {
	rx := width - 77
	if !a.auctionSpriteButton("auction_tender_enter_button", 0, a.auctionButtonState(x+rx, y+row, 62, true), 64, 32, 47, 19, x+rx, y+row, 62) {
		button("Input", rx, row, 62, false)
	}
}
