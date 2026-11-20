package world

import (
	"fmt"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// Personal markets (setup market), after CUser::PacketMyShop 0x478bd0 and
// stMyShop: a seated player sells items from the bags at a price per unit in
// a map where markets are allowed. Buyers browse the list and choose how many
// units to take. Moving, standing, dying, leaving the map or logging out ends
// the market.
const (
	opMyShopBegin  packet.Opcode = 0x2301 // enter setup
	opMyShopCancel packet.Opcode = 0x2302 // cancel / market closed
	opMyShopOpen   packet.Opcode = 0x2303 // start selling with a message
	opMyShopStop   packet.Opcode = 0x2304 // stop selling, back to setup
	opMyShopAdd    packet.Opcode = 0x2305
	opMyShopRemove packet.Opcode = 0x2306
	opMyShopPrice  packet.Opcode = 0x2307
	opMyShopSold   packet.Opcode = 0x2308 // to the seller: a sale
	opMyShopEnter  packet.Opcode = 0x2309 // a buyer opens a market
	opMyShopLeave  packet.Opcode = 0x230a
	opMyShopList   packet.Opcode = 0x230b
	opMyShopClosed packet.Opcode = 0x230c // to buyers: the market ended
	opMyShopBuy    packet.Opcode = 0x230d
	opMyShopStock  packet.Opcode = 0x230e // to buyers: an item's new count
	opShopBubble   packet.Opcode = 0x0523 // the market's bubble over the seller

	myShopSlots     = 20
	myShopTitleMax  = 120
	myShopBuyersMax = 16
)

// Maps where a market can be set up (GX Server configuration for now).
var myShopMaps = map[uint16]bool{35: true, 36: true, 42: true}

// Results of the begin request (client message in the reply).
const (
	myShopBeginOK       = 0
	myShopBeginWrongMap = 1
	myShopBeginBusy     = 2
)

// Results of a purchase.
const (
	myShopBuyOK         = 1
	myShopBuyInvalid    = 0
	myShopBuyNoMoney    = 2
	myShopBuySellerCap  = 3
	myShopBuyNoRoom     = 4
	myShopEnterOK       = 1
	myShopEnterFailed   = 0
	myShopEnterTooMany  = 2
	myShopStatusNone    = 0
	myShopStatusSetup   = 1
	myShopStatusSelling = 2
)

type shopSlot struct {
	used      bool
	bag, slot uint8
	price     uint32
}

type myShop struct {
	status int
	slots  [myShopSlots]shopSlot
	title  string
	buyers []uint32
}

func shopAck(op packet.Opcode, body func(w *packet.Writer)) []byte {
	w := packet.New(op)
	if body != nil {
		body(w)
	}
	return w.MustFinish()
}

// shopBubbleFrame is 0x0523: the market message over the seller (off: no text).
func shopBubbleFrame(id uint32, title string) []byte {
	w := packet.New(opShopBubble)
	w.U32(id)
	if title == "" {
		w.U8(0)
		return w.MustFinish()
	}
	w.U8(1)
	w.U8(uint8(len(title)))
	w.Bytes([]byte(title))
	return w.MustFinish()
}

func (p *ConnectedPlayer) setShopTitle(title string) {
	p.shopMu.Lock()
	p.shopTitle = title
	p.shopMu.Unlock()
}

func (p *ConnectedPlayer) currentShopTitle() string {
	p.shopMu.Lock()
	defer p.shopMu.Unlock()
	return p.shopTitle
}

// canOpenMyShop is the setup precondition.
func (s *Session) canOpenMyShop() int {
	ch := s.char
	if !myShopMaps[ch.MapID] {
		return myShopBeginWrongMap
	}
	if s.dead || ch.CurrentHP == 0 || s.player == nil || tradeFor(ch.ID) != nil || s.ridingVehicle() || s.cast != nil || s.dash != nil ||
		s.inCombat() || s.actionBlocked() || s.movement.ladder.Active || s.movement.jumping {
		return myShopBeginBusy
	}
	return myShopBeginOK
}

// handleMyShop covers the seller's requests 0x2301-0x2307 and the buyer's
// 0x2309, 0x230a and 0x230d.
func (s *Session) handleMyShop(op packet.Opcode, p []byte) {
	if s.char == nil || s.player == nil {
		return
	}
	switch op {
	case opMyShopBegin:
		if s.shop.status != myShopStatusNone {
			s.endMyShop() // starting again discards the previous market
		}
		result := s.canOpenMyShop()
		s.send(shopAck(opMyShopBegin, func(w *packet.Writer) { w.U8(uint8(result)) }), "0x2301 market setup")
		if result == myShopBeginOK {
			s.shop = myShop{status: myShopStatusSetup}
		}
	case opMyShopCancel:
		s.endMyShop()
	case opMyShopOpen:
		s.openMyShop(p)
	case opMyShopStop:
		if s.shop.status != myShopStatusSelling {
			return
		}
		s.stopMyShop()
		s.send(shopAck(opMyShopStop, nil), "0x2304 market stopped")
	case opMyShopAdd:
		if len(p) < 9 || s.shop.status != myShopStatusSetup {
			return
		}
		idx, bag, slot := p[2], p[3], p[4]
		price, _ := packet.U32(p[5:])
		if int(idx) >= myShopSlots || s.shop.slots[idx].used || price == 0 || price > db.MoneyCap || bag == 0 || !isValidItemPosition(bag, slot) {
			return
		}
		i := findItem(s.items, bag, slot)
		if i < 0 || realType(s.items[i].Type) == 27 || s.items[i].Count == 0 {
			return
		}
		for _, o := range s.shop.slots {
			if o.used && o.bag == bag && o.slot == slot {
				return
			}
		}
		s.shop.slots[idx] = shopSlot{used: true, bag: bag, slot: slot, price: price}
		s.send(shopAck(opMyShopAdd, func(w *packet.Writer) { w.U8(idx); w.U8(bag); w.U8(slot); w.U32(price) }), "0x2305 market item added")
	case opMyShopRemove:
		if len(p) < 3 || s.shop.status != myShopStatusSetup || int(p[2]) >= myShopSlots || !s.shop.slots[p[2]].used {
			return
		}
		s.shop.slots[p[2]] = shopSlot{}
		s.send(shopAck(opMyShopRemove, func(w *packet.Writer) { w.U8(p[2]) }), "0x2306 market item removed")
	case opMyShopPrice:
		if len(p) < 7 || s.shop.status != myShopStatusSetup || int(p[2]) >= myShopSlots || !s.shop.slots[p[2]].used {
			return
		}
		price, _ := packet.U32(p[3:])
		if price == 0 || price > db.MoneyCap {
			return
		}
		s.shop.slots[p[2]].price = price
		s.send(shopAck(opMyShopPrice, func(w *packet.Writer) { w.U8(p[2]); w.U32(price) }), "0x2307 market price changed")
	case opMyShopEnter:
		if len(p) >= 6 {
			id, _ := packet.U32(p[2:])
			s.enterMyShop(id)
		}
	case opMyShopLeave:
		s.shopViewing = 0
		s.send(shopAck(opMyShopLeave, nil), "0x230a market left")
	case opMyShopBuy:
		if len(p) >= 4 {
			s.buyFromMyShop(p[2], p[3])
		}
	}
}

// openMyShop is 0x2303: the market starts with its message and the seller sits.
func (s *Session) openMyShop(p []byte) {
	if s.shop.status != myShopStatusSetup || len(p) < 3 {
		return
	}
	n := min(int(p[2]), myShopTitleMax, len(p)-3)
	title := strings.TrimSpace(strings.TrimRight(string(p[3:3+n]), "\x00"))
	title = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, title)
	listed := 0
	for _, sl := range s.shop.slots {
		if sl.used && findItem(s.items, sl.bag, sl.slot) >= 0 {
			listed++
		}
	}
	if listed == 0 || title == "" || s.canOpenMyShop() != myShopBeginOK {
		return
	}
	s.shop.status, s.shop.title = myShopStatusSelling, title
	s.player.setShopTitle(title)
	if !s.player.IsSitting() {
		s.player.SetSitting(true)
		s.broadcastNear(buildCharacterMotion(s.char.ID, MotionSitDown), "0x0506 market sit", true)
	}
	s.send(shopAck(opMyShopOpen, nil), "0x2303 market opened")
	s.broadcastNear(shopBubbleFrame(s.char.ID, title), "0x0523 market bubble", true)
}

// stopMyShop goes back to setup: buyers are told, the bubble goes away.
func (s *Session) stopMyShop() {
	if s.shop.status != myShopStatusSelling {
		return
	}
	s.shop.status = myShopStatusSetup
	s.closeMyShopBuyers()
	s.player.setShopTitle("")
	s.broadcastNear(shopBubbleFrame(s.char.ID, ""), "0x0523 market bubble off", true)
}

func (s *Session) closeMyShopBuyers() {
	for _, id := range s.shop.buyers {
		if p := FindPlayerByID(id); p != nil {
			p.Send(shopAck(opMyShopClosed, nil), "0x230c market closed")
		}
	}
	s.shop.buyers = nil
}

// endMyShop closes the market for good (stand up, move, die, leave...).
func (s *Session) endMyShop() {
	if s.shop.status == myShopStatusNone || s.player == nil || s.char == nil {
		return
	}
	selling := s.shop.status == myShopStatusSelling
	if selling {
		s.closeMyShopBuyers()
		s.player.setShopTitle("")
		s.broadcastNear(shopBubbleFrame(s.char.ID, ""), "0x0523 market bubble off", true)
	}
	s.shop = myShop{}
	s.send(shopAck(opMyShopCancel, nil), "0x2302 market ended")
}

// enterMyShop is 0x2309: a buyer opens the seller's market.
func (s *Session) enterMyShop(id uint32) {
	reply := func(result uint8) {
		s.send(shopAck(opMyShopEnter, func(w *packet.Writer) { w.U8(result); w.U32(id) }), "0x2309 market enter")
	}
	seller := FindPlayerByID(id)
	if seller == nil || seller == s.player || seller.owner == nil || seller.MapID() != s.char.MapID || seller.Faction() != s.faction && s.char.MapID != 42 || s.dead {
		reply(myShopEnterFailed)
		return
	}
	other := seller.owner
	if !other.stateMu.TryLock() {
		reply(myShopEnterFailed)
		return
	}
	defer other.stateMu.Unlock()
	if other.player != seller || other.char == nil || other.shop.status != myShopStatusSelling {
		reply(myShopEnterFailed)
		return
	}
	if len(other.shop.buyers) >= myShopBuyersMax {
		reply(myShopEnterTooMany)
		return
	}
	known := false
	for _, b := range other.shop.buyers {
		known = known || b == s.char.ID
	}
	if !known {
		other.shop.buyers = append(other.shop.buyers, s.char.ID)
	}
	s.shopViewing = id
	reply(myShopEnterOK)
	s.send(other.myShopList(), "0x230b market list")
}

// myShopList is 0x230b: count and, per listed item, its index, price and record.
func (s *Session) myShopList() []byte {
	var idxs []int
	for idx, sl := range s.shop.slots {
		if sl.used && findItem(s.items, sl.bag, sl.slot) >= 0 {
			idxs = append(idxs, idx)
		}
	}
	w := packet.New(opMyShopList)
	w.U8(uint8(len(idxs)))
	for _, idx := range idxs {
		sl := s.shop.slots[idx]
		it := &s.items[findItem(s.items, sl.bag, sl.slot)]
		w.U8(uint8(idx))
		w.U32(sl.price)
		w.U8(it.Type)
		w.U8(it.TypeID)
		w.U8(uint8(min(it.Count, 255)))
		w.U16(it.Quality)
		w.Bytes(it.Gems[:])
		w.FixedString(it.CraftName, 21)
	}
	return w.MustFinish()
}

// buyFromMyShop is 0x230d: take count units of the listed item idx.
func (s *Session) buyFromMyShop(idx, count uint8) {
	reply := func(result uint8) {
		s.send(shopAck(opMyShopBuy, func(w *packet.Writer) { w.U8(result) }), fmt.Sprintf("0x230d market buy result %d", result))
	}
	seller := FindPlayerByID(s.shopViewing)
	if seller == nil || seller.owner == nil || seller.owner == s || count == 0 || int(idx) >= myShopSlots || s.dead {
		reply(myShopBuyInvalid)
		return
	}
	other := seller.owner
	if !other.stateMu.TryLock() {
		reply(myShopBuyInvalid)
		return
	}
	defer other.stateMu.Unlock()
	if other.player != seller || other.char == nil || other.shop.status != myShopStatusSelling || !other.shop.slots[idx].used {
		reply(myShopBuyInvalid)
		return
	}
	sl := other.shop.slots[idx]
	si := findItem(other.items, sl.bag, sl.slot)
	if si < 0 || other.items[si].Count < uint16(count) {
		reply(myShopBuyInvalid)
		return
	}
	total := uint64(sl.price) * uint64(count)
	switch {
	case total > uint64(s.char.Money):
		reply(myShopBuyNoMoney)
		return
	case uint64(other.char.Money)+total > db.MoneyCap:
		reply(myShopBuySellerCap)
		return
	}
	a := db.CharacterChange{Before: *s.char, After: *s.char, ItemsBefore: s.items, ItemsAfter: append([]db.Item(nil), s.items...)}
	b := db.CharacterChange{Before: *other.char, After: *other.char, ItemsBefore: other.items, ItemsAfter: append([]db.Item(nil), other.items...)}
	a.After.Money = uint32(uint64(s.char.Money) - total)
	b.After.Money = uint32(uint64(other.char.Money) + total)
	tpl := b.ItemsAfter[si]
	b.ItemsAfter[si].Count -= uint16(count)
	sellerPacket := buildRemoveItem(&b.ItemsAfter[si], b.ItemsAfter[si].Count == 0)
	remaining := b.ItemsAfter[si].Count
	if remaining == 0 {
		b.ItemsAfter = append(b.ItemsAfter[:si], b.ItemsAfter[si+1:]...)
	}
	adds, left := planGiveItem(&a.ItemsAfter, &tpl, uint16(count), maxStackOf(tpl.Type, tpl.TypeID))
	if left != 0 {
		reply(myShopBuyNoRoom)
		return
	}
	if err := db.CommitGameplay([]db.CharacterChange{a, b}, nil); err != nil {
		s.logErr("market purchase transaction", err)
		reply(myShopBuyInvalid)
		return
	}
	*s.char, *other.char = a.After, b.After
	s.items, other.items = a.ItemsAfter, b.ItemsAfter
	for _, o := range []*Session{s, other} {
		o.syncPlayer()
		o.player.UpdateItems(o.items)
	}
	for _, pk := range adds {
		s.send(pk, "0x0205 market purchase item")
	}
	s.send(buildSetMoney(s.char.Money), "0x0213 market purchase money")
	other.send(sellerPacket, "0x0204 market sale item")
	other.send(buildSetMoney(other.char.Money), "0x0213 market sale money")
	reply(myShopBuyOK)
	if remaining == 0 {
		other.shop.slots[idx] = shopSlot{}
	}
	stock := shopAck(opMyShopStock, func(w *packet.Writer) { w.U8(idx); w.U8(uint8(min(remaining, 255))) })
	for _, id := range other.shop.buyers {
		if p := FindPlayerByID(id); p != nil {
			p.Send(stock, "0x230e market stock")
		}
	}
	other.send(shopAck(opMyShopSold, func(w *packet.Writer) { w.U8(idx); w.U8(count); w.U32(uint32(total)) }), "0x2308 market sale")
	s.log.Printf("GX Server World market sale seller=%d buyer=%d item=%d/%d count=%d total=%d", other.char.ID, s.char.ID, tpl.Type, tpl.TypeID, count, total)
	// Nothing left to sell: the market closes.
	empty := true
	for _, o := range other.shop.slots {
		empty = empty && !o.used
	}
	if empty {
		other.endMyShop()
	}
}
