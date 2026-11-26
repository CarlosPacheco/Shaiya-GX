package world

import (
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestGuildHouseDefaultsMatchExecutable(t *testing.T) {
	h := defaultGuildHouse()
	if h.CreateMoney != 1_000_000 || h.BuyMoney != 50000000 || h.DefaultKeep != 2000 {
		t.Fatalf("defaults %d %d %d", h.CreateMoney, h.BuyMoney, h.DefaultKeep)
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
	for shop := 1; shop <= 3; shop++ {
		if h.ShopFactor[shop] != [4]float64{0.95, 0.90, 0.85, 0.80} {
			t.Fatalf("shop %d factors %v", shop, h.ShopFactor[shop])
		}
	}
	if h.ShopFactor[4] != [4]float64{0.90, 0.80, 0.70, 0.60} {
		t.Fatalf("potion factors %v", h.ShopFactor[4])
	}
	if h.GateRate != [6]float64{} || h.RepairRate != [6]float64{0.99, 0.95, 0.95, 0.90, 0.90, 0.90} || h.MixRate != [6]float64{0.90, 0.80, 0.70, 0.60, 0.50, 0.10} {
		t.Fatalf("gate %v repair %v link %v", h.GateRate, h.RepairRate, h.MixRate)
	}
	if got := h.service(guildNpcSmith).Keep; got != [6]uint16{0, 100, 200, 500, 800, 1500} {
		t.Fatalf("smith keep %v", got)
	}
	if h.CreateMoney != 1_000_000 || h.BuyMoney != 50_000_000 || h.DefaultKeep != 2000 {
		t.Fatalf("house %d %d %d", h.CreateMoney, h.BuyMoney, h.DefaultKeep)
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
	if got := h.service(guildNpcPotion).Cost; got != [7]uint16{0, 2000, 5000, 10000, 0, 0, 0} {
		t.Fatalf("potion costs %v", got)
	}
	if got := h.service(guildNpcBank).Cost; got != [7]uint16{0, 200, 500, 1000, 2000, 5000, 0} {
		t.Fatalf("bank costs %v", got)
	}
	// Absent sections keep the built-in values.
	if d := parseGuildHouse(iniFile{}); d.ShopFactor != defaultGuildHouse().ShopFactor {
		t.Fatal("empty file changed the defaults")
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

func TestGuildHouseServicesQuoteUsesServerSettings(t *testing.T) {
	old := guildHouse
	t.Cleanup(func() { guildHouse = old })
	guildHouse = parseGuildHouse(iniFile{
		"HOUSE":       {"HOUSECOST": "54321000", "BASEUPKEEP": "0"},
		"PORTAL":      {"LEVEL1": "Upkeep 187", "LEVEL2": "Upgrade 733"},
		"GENERALSHOP": {"LEVEL3": "Upgrade 0"},
	})
	decode := func(g *guild) proto.GuildHouseServices {
		t.Helper()
		frame := g.houseServicesPacket()
		if proto.Op(frame[2:]) != proto.GXGuildHouseServices {
			t.Fatalf("guild house services frame %x", frame)
		}
		quote, err := proto.DecodeGuildHouseServices(frame[2:])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := proto.DecodeGuildHouseServices(frame[2 : len(frame)-1]); err == nil {
			t.Fatal("truncated guild house services frame accepted")
		}
		return quote
	}
	if quote := decode(nil); quote.GuildID != 0 || quote.BuyGold != 54_321_000 || quote.Upkeep != 0 || quote.Active || len(quote.Rows) != 7 {
		t.Fatalf("initial quote %+v", quote)
	}
	g := &guild{ID: 44, BuyHouse: 1, UseHouse: 1}
	g.npcs[0] = guildNpcSlot{Type: guildNpcGate, Level: 1}
	g.npcs[1] = guildNpcSlot{Type: guildNpcGeneral, Level: 2}
	g.npcs[2] = guildNpcSlot{Type: guildNpcBank, Level: 5}
	g.calculateKeepEtin()
	quote := decode(g)
	if quote.GuildID != g.ID || quote.Upkeep != 1187 || !quote.Active {
		t.Fatalf("owned house quote %+v", quote)
	}
	gate, ok := quote.Service(guildNpcGate)
	if !ok || gate.Level != 1 || gate.Upkeep != 187 || gate.NextCost != 733 {
		t.Fatalf("gate quote %+v", gate)
	}
	for _, npc := range []uint16{guildNpcGeneral, guildNpcBank} {
		if service, ok := quote.Service(npc); !ok || service.NextCost != 0 {
			t.Fatalf("unavailable upgrade quote %+v", service)
		}
	}
}

func TestGuildHouseRates(t *testing.T) {
	old := guildHouse
	guildHouse = defaultGuildHouse()
	defer func() { guildHouse = old }()
	g := &guild{Rank: 5, Country: 0}
	g.npcs[0] = guildNpcSlot{Type: guildNpcWeapon, Level: 1}
	// The discount belongs to the kind of shop, whatever NPC sells.
	if r, res, _ := g.shopBuyRate(1); res != 0 || r != 0.98 {
		t.Fatalf("level-1 weapon shop: %v %d", r, res)
	}
	if r, res, _ := g.shopBuyRate(2); res != 0 || r != 0.99 {
		t.Fatalf("level-0 armor shop: %v %d", r, res)
	}
	if _, res, _ := (&guild{Rank: 31}).shopBuyRate(4); res != 0 {
		t.Fatalf("an unranked guild was refused the potion shop: %d", res)
	}
	g.npcs[1] = guildNpcSlot{Type: guildNpcGate, Level: 3}
	guildHouse.GateRate = [6]float64{1, 0.9, 0.8, 0.5}
	if r := g.gateRate(); r != 0.5 {
		t.Fatalf("level-3 gatekeeper fee factor %v", r)
	}
}

func TestGuildCreateFeeINI(t *testing.T) {
	for value, want := range map[string]uint32{"0": 0, "250000": 250000, "-1": 1000000, "4294967296": 1000000} {
		h := parseGuildHouse(iniFile{"HOUSE": {"CREATEGUILDCOST": value}})
		if h.CreateMoney != want {
			t.Errorf("GuildCreateMoney %q = %d, want %d", value, h.CreateMoney, want)
		}
	}
}

func TestGMGuildHouseIsAlwaysOpenWithoutGRB(t *testing.T) {
	g := &guild{Rank: 31, gmHouse: true, Etin: 0, KeepEtin: 2000}
	g.forceGMHouse()
	if !g.houseEnabled() || g.Rank != 31 {
		t.Fatalf("GM house closed: rank=%d use=%d buy=%d", g.Rank, g.UseHouse, g.BuyHouse)
	}
	g.UseHouse = 2
	g.settleGuildHouse() // no upkeep, never suspended
	if g.UseHouse != 1 || g.Etin != 0 {
		t.Fatal("GM house was suspended or charged")
	}
	normal := &guild{Rank: 31}
	normal.forceGMHouse()
	if normal.houseEnabled() {
		t.Fatal("a guild without a purchased house got one")
	}
	bought := &guild{Rank: 31, BuyHouse: 1, UseHouse: 1}
	if !bought.houseEnabled() {
		t.Fatal("an unranked guild with a bought house is closed")
	}
}
