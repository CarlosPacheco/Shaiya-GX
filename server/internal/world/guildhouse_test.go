package world

import (
	"path/filepath"
	"testing"
)

func TestGuildHouseDefaultsMatchExecutable(t *testing.T) {
	h := defaultGuildHouse()
	if h.BuyMoney != 50000000 || h.DefaultKeep != 2000 {
		t.Fatalf("defaults %d %d", h.BuyMoney, h.DefaultKeep)
	}
	if got := h.service(guildNpcSmith).Cost; got != [7]uint16{0, 500, 1000, 2000, 10000, 15000, 0} {
		t.Fatalf("smith costs %v", got)
	}
	if h.MixRate != [6]float64{1, 1, 0.95, 0.95, 0.95, 0.9} || h.MixPercent != [6]int32{0, 0, 0, 0, 1, 2} {
		t.Fatalf("mix %v %v", h.MixRate, h.MixPercent)
	}
}

func TestGuildHouseINIOverridesAndKeepsMissing(t *testing.T) {
	ini, err := readIni(filepath.Join("..", "..", "data", "game", "GuildHouse.ini"))
	if err != nil {
		t.Fatal(err)
	}
	h := parseGuildHouse(ini)
	if h.ShopRate != [4]float64{0.95, 0.90, 0.85, 0.80} || h.PotionRate != [4]float64{0.90, 0.80, 0.70, 0.60} {
		t.Fatalf("shop rates %v %v", h.ShopRate, h.PotionRate)
	}
	if got := h.service(guildNpcGeneral).Cost; got != [7]uint16{0, 150, 200, 400, 0, 0, 0} {
		t.Fatalf("general costs %v", got)
	}
	if got := h.service(guildNpcGate).Keep; got != [6]uint16{0, 200, 500, 800, 0, 0} {
		t.Fatalf("gate keep %v", got)
	}
	if h.MixPercent != [6]int32{0, 0, 1, 1, 2, 2} || h.BankSlots != [6]uint16{40, 80, 120, 160, 200, 240} {
		t.Fatalf("mix %v bank %v", h.MixPercent, h.BankSlots)
	}
	if h.NpcIDs[0][1][1][0] != 196 || h.NpcIDs[0][1][2][0] != 198 || h.NpcIDs[1][2][2][0] != 212 {
		t.Fatalf("NPC ids %v", h.NpcIDs)
	}
	// Absent keys (the potion shop tables) keep the executable values.
	if got := h.service(guildNpcPotion).Cost; got != [7]uint16{0, 2000, 5000, 10000, 0, 0, 0} {
		t.Fatalf("potion costs %v", got)
	}
}

func TestGuildHouseKeepEtinAndList(t *testing.T) {
	old := guildHouse
	guildHouse = defaultGuildHouse()
	defer func() { guildHouse = old }()
	g := &guild{}
	g.npcs[0] = guildNpcSlot{Type: guildNpcGate, Level: 2}
	g.npcs[3] = guildNpcSlot{Type: guildNpcSmith, Level: 1}
	g.calculateKeepEtin()
	if g.KeepEtin != 2000+500+100 {
		t.Fatalf("keep etin %d", g.KeepEtin)
	}
	list := g.npcListPacket()
	want := []byte{13, 0, 0x24, 0x0d, 2, 2, 0, 2, 0, 3, 0, 1, 3}
	if string(list) != string(want) {
		t.Fatalf("list %v, want %v", list, want)
	}
	if (&guild{}).npcListPacket() != nil {
		t.Fatal("empty guild sent a list")
	}
	if g.bankSlots() != 40 {
		t.Fatalf("bank slots %d", g.bankSlots())
	}
	g.npcs[1] = guildNpcSlot{Type: guildNpcBank, Level: 3}
	if g.bankSlots() != 160 {
		t.Fatalf("bank slots %d", g.bankSlots())
	}
}
