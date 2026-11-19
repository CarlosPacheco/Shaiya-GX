package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestGuildHousePackets(t *testing.T) {
	s := New()
	s.Inventory[[2]uint8{1, 4}] = proto.InventoryItem{Bag: 1, Slot: 4, Type: 43, TypeID: 3, Count: 2}
	list := proto.NewWriter(proto.GuildNpcList)
	list.U8(2)
	list.U16(2)
	list.U8(1)
	list.U8(0)
	list.U16(0x401)
	list.U8(3)
	list.U8(4)
	if _, err := s.Apply(list.Bytes()); err != nil {
		t.Fatal(err)
	}
	if got := s.Guild.Npcs[4]; got.Type != 0x401 || got.Level != 3 || len(s.Guild.Npcs) != 2 {
		t.Fatalf("npcs %v", s.Guild.Npcs)
	}
	up := proto.NewWriter(proto.GuildNpcUpgrade)
	up.U8(0)
	up.U16(2)
	up.U8(2)
	up.U8(0)
	if _, err := s.Apply(up.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Guild.Npcs[0].Level != 2 || s.Guild.NpcSerial != 1 {
		t.Fatalf("upgrade not applied: %v", s.Guild.Npcs)
	}
	buy := proto.NewWriter(proto.GuildHouseBuy)
	buy.U8(0)
	buy.U32(777)
	if _, err := s.Apply(buy.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Details.Money != 777 || !s.Guild.HouseBought || s.Guild.HouseSerial != 1 {
		t.Fatalf("buy: money %d bought %v", s.Details.Money, s.Guild.HouseBought)
	}
	don := proto.NewWriter(proto.GuildDonate)
	don.U8(1)
	don.U8(1)
	don.U8(4)
	if _, err := s.Apply(don.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(s.Inventory) != 0 || s.Guild.Donated != 1 || s.Guild.DonateSerial != 1 {
		t.Fatalf("donation: %v", s.Inventory)
	}
}
