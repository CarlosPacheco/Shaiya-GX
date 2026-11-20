package proto

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/textwire"
)

// Personal markets (setup market): the request and reply opcodes of
// CUser::PacketMyShop (0x2301-0x230e) and the bubble broadcast 0x0523.
const (
	MyShopBegin  Opcode = 0x2301 // setup; the reply carries a result
	MyShopCancel Opcode = 0x2302 // cancel; also the notice that the market ended
	MyShopOpen   Opcode = 0x2303 // sell, with the market message
	MyShopStop   Opcode = 0x2304
	MyShopAdd    Opcode = 0x2305
	MyShopRemove Opcode = 0x2306
	MyShopPrice  Opcode = 0x2307
	MyShopSold   Opcode = 0x2308 // to the seller: index, count, gold
	MyShopEnter  Opcode = 0x2309
	MyShopLeave  Opcode = 0x230a
	MyShopList   Opcode = 0x230b
	MyShopClosed Opcode = 0x230c // to buyers
	MyShopBuy    Opcode = 0x230d
	MyShopStock  Opcode = 0x230e // to buyers: index, new count
	ShopBubble   Opcode = 0x0523

	MyShopSlots      = 20
	MyShopTitleMax   = 60
	MyShopTitleBytes = 120
	MyShopReach      = 10
	MyShopPriceMax   = 4_000_000_000
)

const (
	MyShopInactive = iota
	MyShopSetup
	MyShopSelling
)

// MyShop begin results.
const (
	MyShopBeginOK       = 0
	MyShopBeginWrongMap = 1
	MyShopBeginBusy     = 2
)

// MyShop buy results.
const (
	MyShopBuyInvalid   = 0
	MyShopBuyOK        = 1
	MyShopBuyNoMoney   = 2
	MyShopBuySellerCap = 3
	MyShopBuyNoRoom    = 4
)

const (
	MyShopEnterFailed = 0
	MyShopEnterOK     = 1
	MyShopEnterFull   = 2
)

func MyShopOpenPacket(title string) []byte {
	w := NewWriter(MyShopOpen)
	text := []byte(textwire.Prefix(title, MyShopTitleBytes))
	w.U8(uint8(len(text)))
	w.Raw(text)
	return w.Bytes()
}

func MyShopAddPacket(index, bag, slot uint8, price uint32) []byte {
	w := NewWriter(MyShopAdd)
	w.U8(index)
	w.U8(bag)
	w.U8(slot)
	w.U32(price)
	return w.Bytes()
}

func MyShopRemovePacket(index uint8) []byte {
	w := NewWriter(MyShopRemove)
	w.U8(index)
	return w.Bytes()
}

func MyShopPricePacket(index uint8, price uint32) []byte {
	w := NewWriter(MyShopPrice)
	w.U8(index)
	w.U32(price)
	return w.Bytes()
}

func MyShopEnterPacket(seller uint32) []byte {
	w := NewWriter(MyShopEnter)
	w.U32(seller)
	return w.Bytes()
}

func MyShopBuyPacket(index, count uint8) []byte {
	w := NewWriter(MyShopBuy)
	w.U8(index)
	w.U8(count)
	return w.Bytes()
}

// MyShopItem is one entry of the list 0x230b.
type MyShopItem struct {
	Index byte
	Price uint32
	Item  InventoryItem
}

func DecodeMyShopList(p []byte) ([]MyShopItem, error) {
	r := NewReader(p)
	n := int(r.U8())
	if err := r.Err(); err != nil {
		return nil, err
	}
	const recordSize = 37
	if Op(p) != MyShopList || n > MyShopSlots || n*recordSize != r.Remaining() {
		return nil, fmt.Errorf("market list: invalid count %d", n)
	}
	out := make([]MyShopItem, n)
	var seen [MyShopSlots]bool
	for i := range out {
		e := &out[i]
		e.Index, e.Price = r.U8(), r.U32()
		e.Item.Type, e.Item.TypeID, e.Item.Count = r.U8(), r.U8(), uint16(r.U8())
		e.Item.Quality = r.U16()
		for j := range e.Item.Gems {
			e.Item.Gems[j] = r.U8()
		}
		e.Item.Craft = r.Fixed(21)
		if int(e.Index) >= MyShopSlots || seen[e.Index] || e.Price == 0 || e.Price > MyShopPriceMax || e.Item.Type == 0 || e.Item.Count == 0 {
			return nil, fmt.Errorf("market list: invalid item at index %d", e.Index)
		}
		seen[e.Index] = true
	}
	return out, r.Err()
}

// DecodeShopBubble is 0x0523: the seller's id and, when the market is open,
// its message.
func DecodeShopBubble(p []byte) (id uint32, title string, err error) {
	r := NewReader(p)
	id = r.U32()
	on := r.U8()
	if on != 0 {
		n := int(r.U8())
		if on != 1 || n > MyShopTitleBytes || n != r.Remaining() {
			return id, "", fmt.Errorf("invalid market message length %d", n)
		}
		title = textwire.Decode(r.Bytes(n))
	} else if r.Remaining() != 0 {
		return id, "", fmt.Errorf("trailing bytes in closed market sign")
	}
	return id, title, r.Err()
}
