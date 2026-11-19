package proto

import (
	"encoding/json"
	"errors"
)

const GXAuction Opcode = 0x7F1F
const AuctionBoard = 117
const AuctionPageSize = 8

const (
	AuctionBrowse = iota
	AuctionSelling
	AuctionBidding
	AuctionItems
	AuctionGold
	AuctionWatch
)

var auctionHours = [...]int64{7, 24, 72, 168}

func AuctionHours(duration uint8) int64 {
	if int(duration) >= len(auctionHours) {
		return 0
	}
	return auctionHours[duration]
}

func AuctionDeposit(start uint32, duration uint8) uint32 {
	return uint32(AuctionHours(duration)) * 5 * uint32((uint64(start)+499999)/500000)
}

type AuctionRequest struct {
	Item                         *InventoryItem
	Sequence                     uint64
	Sort                         int
	Action                       string
	NPC                          uint32
	ID, Version                  int64
	View, Page                   int
	Query                        string
	Category, MinLevel, MaxLevel int
	ItemType, ItemID             uint8
	Duration                     uint8
	Count                        uint16
	Start, Buyout, Amount        uint32
	Source                       string
}

type AuctionRow struct {
	ID, Version                 int64
	Name, Seller, Craft, Reason string
	Type, TypeID                uint8
	Level, Count, Quality       uint16
	Gems                        [6]uint8
	Price, Buyout, Gold         uint32
	Ends                        int64
	Active, Mine, HighBidder    bool
}

type AuctionReply struct {
	Sequence       uint64
	NPC            uint32
	View, Page     int
	More           bool
	Error, Message string
	Rows           []AuctionRow
}

func AuctionPacket(value any) []byte {
	body, err := json.Marshal(value)
	if err != nil || len(body) > 12000 {
		return nil
	}
	w := NewWriter(GXAuction)
	w.Raw(body)
	return w.Bytes()
}

func DecodeAuction(p []byte, limit int, value any) error {
	if len(p) < 3 || len(p) > limit || Op(p) != GXAuction {
		return errors.New("invalid auction packet")
	}
	return json.Unmarshal(p[2:], value)
}

var AuctionCategories = [...]string{"All", "Weapons", "Armor", "Accessories", "Lapis", "Miscellaneous"}

func AuctionCategory(realType, itemType uint8) int {
	switch {
	case realType >= 1 && realType <= 15:
		return 1
	case realType >= 16 && realType <= 21 || realType == 24:
		return 2
	case realType == 22 || realType == 23 || realType == 25:
		return 3
	case itemType == 30 || itemType == 95:
		return 4
	default:
		return 5
	}
}
