package world

import (
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// Guild house, guild NPCs and etin follow ps_game.exe EP4.5: CGuild::NpcUpgrade
// 0x430640, CalculateKeepEtin 0x430a50, SendNpcLvList 0x42f0e0, the 0x0D23,
// 0x0D25, 0x0D34 and 0x0D35 branches of CUser::PacketGuild 0x4764a0 and
// CGuildMgr::InitGuildHouseIniFile 0x4319a0. See docs/guild-house-research.md.

// Guild NPC types as stored in the eight CGuild NPC slots.
const (
	guildNpcGate    = 0x002
	guildNpcWeapon  = 0x101
	guildNpcDefense = 0x201
	guildNpcGeneral = 0x301
	guildNpcSmith   = 0x003
	guildNpcPotion  = 0x401
	guildNpcBank    = 0x006

	guildNpcSlots    = 8
	guildNpcServices = 7
	guildEtinSpecial = 52 // Special of the Type 43 etin items: 52..55 are x1, x10, x100, x1000
	guildEtinType    = 43 // '+', the item type CUser::PacketGuild 0x0D35 accepts
)

// Results of 0x0D25 (CGuild::NpcUpgrade).
const (
	guildNpcOK       = 0
	guildNpcNoEtin   = 1
	guildNpcBadLevel = 2
	guildNpcRefused  = 3
)

// Results of 0x0D34.
const (
	guildHouseBought  = 0
	guildHouseOwned   = 1
	guildHouseNoMoney = 2
	guildHouseNoRight = 3
)

// guildNpcSlot is a CGuild NPC slot (+0x47c, type and level words).
type guildNpcSlot struct{ Type, Level uint16 }

// guildServiceTable is one g_GuildNpcUpgradeEtin / g_GuildNpcKeepEtin row:
// index is the NPC level (1-5).
type guildServiceTable struct {
	Type uint16
	Max  uint8
	Cost [7]uint16
	Keep [6]uint16
}

// guildHouseSettings is everything InitGuildHouseIniFile loads, starting from
// the executable's own defaults.
type guildHouseSettings struct {
	CreateMoney, BuyMoney, DefaultKeep uint32
	Services                           [guildNpcServices]guildServiceTable // gate, weapon, defense, general, smith, potion, bank
	// ShopFactor is the price factor of a shop type (1 weapon, 2 defense, 3
	// general, 4 potion) by the guild's level of that service (0-3).
	ShopFactor [5][4]float64
	GateRate   [6]float64 // teleport fee factor by gatekeeper level
	RepairRate [6]float64 // smith repair factor by level
	MixRate    [6]float64 // gem link cost factor by smith level
	MixPercent [6]int32   // gem link success bonus by smith level
	BankSlots  [6]uint16  // usable warehouse slots by bank level
}

func defaultGuildHouse() guildHouseSettings {
	return guildHouseSettings{
		CreateMoney: 1_000_000, BuyMoney: 50000000, DefaultKeep: 2000,
		Services: [guildNpcServices]guildServiceTable{
			{Type: guildNpcGate, Max: 3, Cost: [7]uint16{0, 1000, 1500, 2000}, Keep: [6]uint16{0, 200, 500, 800}},
			{Type: guildNpcWeapon, Max: 3, Cost: [7]uint16{0, 100, 200, 500}},
			{Type: guildNpcDefense, Max: 3, Cost: [7]uint16{0, 100, 200, 500}},
			{Type: guildNpcGeneral, Max: 5, Cost: [7]uint16{0, 50, 100, 200}},
			{Type: guildNpcSmith, Max: 5, Cost: [7]uint16{0, 500, 1000, 2000, 10000, 15000}, Keep: [6]uint16{0, 100, 200, 500, 800, 1000}},
			{Type: guildNpcPotion, Max: 3, Cost: [7]uint16{0, 2000, 5000, 10000}, Keep: [6]uint16{0, 50, 200, 500}},
			{Type: guildNpcBank, Max: 5, Cost: [7]uint16{0, 200, 500, 1000, 2000, 5000}, Keep: [6]uint16{0, 100, 200, 500, 800, 1000}},
		},
		ShopFactor: [5][4]float64{{}, {0.99, 0.98, 0.95, 0.9}, {0.99, 0.98, 0.95, 0.9}, {0.99, 0.98, 0.95, 0.9}, {0.9, 0.8, 0.7, 0.6}},
		RepairRate: [6]float64{0.99, 0.95, 0.95, 0.9, 0.9, 0.9},
		MixRate:    [6]float64{1, 1, 0.95, 0.95, 0.95, 0.9},
		MixPercent: [6]int32{0, 0, 0, 0, 1, 2},
		BankSlots:  [6]uint16{40, 80, 120, 160, 200, 240},
	}
}

// guildHouse is the active configuration.
var guildHouse = defaultGuildHouse()

// iniTokens splits a value like ParseString1/ParseString2: the first 254
// characters on " ,=()[]{}" and white space.
func iniTokens(value string) []string {
	if len(value) > 254 {
		value = value[:254]
	}
	return strings.FieldsFunc(value, func(r rune) bool { return strings.ContainsRune(" ,=()[]{}\t\r\n", r) })
}

// iniUint is the atoi the original parser applies: a decimal prefix.
func iniInt(token string) int64 {
	end := 0
	if end < len(token) && (token[0] == '-' || token[0] == '+') {
		end++
	}
	for end < len(token) && token[end] >= '0' && token[end] <= '9' {
		end++
	}
	n, _ := strconv.ParseInt(token[:end], 10, 64)
	return n
}

// parseGuildHouse applies GuildHouse.ini over the built-in defaults. Missing
// sections, levels or fields keep their default. Each service section lists
// its levels as `LevelN = Field value, Field value`; see the file itself.
func parseGuildHouse(ini iniFile) guildHouseSettings {
	h := defaultGuildHouse()
	number := func(v string) (float64, bool) {
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return n, err == nil
	}
	if n, ok := number(ini["HOUSE"]["CREATEGUILDCOST"]); ok && n >= 0 && n <= math.MaxUint32 {
		h.CreateMoney = uint32(n)
	}
	if n, ok := number(ini["HOUSE"]["HOUSECOST"]); ok && n > 0 && n <= math.MaxUint32 {
		h.BuyMoney = uint32(n)
	}
	if n, ok := number(ini["HOUSE"]["BASEUPKEEP"]); ok && n >= 0 && n <= math.MaxUint32 {
		h.DefaultKeep = uint32(n)
	}
	// levels reads "Level0".."Level5" of a section as field -> value maps.
	levels := func(section string, count int) []map[string]float64 {
		out := make([]map[string]float64, count)
		for level := range out {
			value, found := ini[section]["LEVEL"+strconv.Itoa(level)]
			if !found {
				continue
			}
			out[level] = map[string]float64{}
			for _, part := range strings.Split(value, ",") {
				fields := strings.Fields(part)
				if len(fields) != 2 {
					continue
				}
				if n, ok := number(fields[1]); ok {
					out[level][strings.ToUpper(fields[0])] = n
				}
			}
		}
		return out
	}
	service := func(index int, section string, count int, apply func(level int, row map[string]float64)) {
		for level, row := range levels(section, count) {
			if row == nil {
				continue
			}
			if v, ok := row["UPGRADE"]; ok && level >= 1 {
				h.Services[index].Cost[level] = uint16(max(v, 0))
			}
			if v, ok := row["UPKEEP"]; ok && level >= 1 && level < len(h.Services[index].Keep) {
				h.Services[index].Keep[level] = uint16(max(v, 0))
			}
			if apply != nil {
				apply(level, row)
			}
		}
	}
	shop := func(index int, section string, shopType int) {
		service(index, section, 4, func(level int, row map[string]float64) {
			if v, ok := row["PRICE"]; ok {
				h.ShopFactor[shopType][level] = v
			}
		})
	}
	service(0, "PORTAL", 4, func(level int, row map[string]float64) {
		if v, ok := row["FEE"]; ok {
			h.GateRate[level] = v
		}
	})
	shop(1, "WEAPONSHOP", 1)
	shop(2, "ARMORSHOP", 2)
	shop(3, "GENERALSHOP", 3)
	shop(5, "POTIONSHOP", 4)
	service(4, "BLACKSMITH", 6, func(level int, row map[string]float64) {
		if v, ok := row["REPAIR"]; ok {
			h.RepairRate[level] = v
		}
		if v, ok := row["LINK"]; ok {
			h.MixRate[level] = v
		}
		if v, ok := row["LINKBONUS"]; ok {
			h.MixPercent[level] = int32(v)
		}
	})
	service(6, "WAREHOUSE", 6, func(level int, row map[string]float64) {
		if v, ok := row["SLOTS"]; ok {
			h.BankSlots[level] = uint16(max(v, 0))
		}
	})
	return h
}

func loadGuildHouseIni() {
	ini, err := readIni(filepath.Join(gameDataDir(), "GuildHouse.ini"))
	if err != nil {
		guildHouse = defaultGuildHouse()
		return
	}
	guildHouse = parseGuildHouse(ini)
}

// service is the upgrade table of an NPC type (nil for an unknown type).
func (h *guildHouseSettings) service(npcType uint16) *guildServiceTable {
	for i := range h.Services {
		if h.Services[i].Type == npcType {
			return &h.Services[i]
		}
	}
	return nil
}

// npcLevel is the level of the guild's NPC of that type, 0 without it.
func (g *guild) npcLevel(npcType uint16) uint16 {
	for _, s := range g.npcs {
		if s.Type == npcType {
			return s.Level
		}
	}
	return 0
}

// houseEnabled is CGuild::EnableGuildHouse without the ranking condition: the
// house is active and bought.
func (g *guild) houseEnabled() bool {
	return g.UseHouse == 1 && g.BuyHouse != 0
}

// forceGMHouse keeps a GM-founded guild's house open:
// bought, active and free of upkeep.
func (g *guild) forceGMHouse() {
	if g.gmHouse {
		g.BuyHouse, g.UseHouse = 1, 1
	}
}

// calculateKeepEtin is CGuild::CalculateKeepEtin: the default plus the upkeep
// of every NPC at its level.
func (g *guild) calculateKeepEtin() {
	keep := guildHouse.DefaultKeep
	for _, s := range g.npcs {
		if s.Type == 0 {
			continue
		}
		if t := guildHouse.service(s.Type); t != nil && int(s.Level) < len(t.Keep) {
			keep += uint32(t.Keep[s.Level])
		}
	}
	g.KeepEtin = keep
}

// bankSlots is GetGuildHouseBankAllowSlotCount: PermitSlot at the bank NPC
// level (the first entry without it).
func (g *guild) bankSlots() int {
	level := g.npcLevel(guildNpcBank)
	if level > 5 {
		level = 5
	}
	return int(guildHouse.BankSlots[level])
}

func (g *guild) row() db.GuildRow {
	return db.GuildRow{ID: g.ID, Name: g.Name, MasterName: g.Master, Country: g.Country, Points: g.Points, Rank: g.Rank,
		Etin: g.Etin, Remark: g.Remark, BuyHouse: g.BuyHouse, UseHouse: g.UseHouse, EtinReturn: g.EtinReturn, KeepEtin: g.KeepEtin}
}

// npcListPacket is CGuild::SendNpcLvList (0x0D24); nil without NPCs.
func (g *guild) npcListPacket() []byte {
	var entries []guildNpcSlot
	var slots []uint8
	for i, sl := range g.npcs {
		if sl.Type != 0 {
			entries, slots = append(entries, sl), append(slots, uint8(i))
		}
	}
	if len(entries) == 0 {
		return nil
	}
	w := packet.New(packet.GuildNpcList)
	w.U8(uint8(len(entries)))
	for i, sl := range entries {
		w.U16(sl.Type)
		w.U8(uint8(sl.Level))
		w.U8(slots[i])
	}
	return w.MustFinish()
}

func buildGuildNpcResult(result uint8, npcType uint16, level, slot uint8) []byte {
	w := packet.New(packet.GuildNpcUpgrade)
	w.U8(result)
	w.U16(npcType)
	w.U8(level)
	w.U8(slot)
	return w.MustFinish()
}

// houseServicesPacket quotes the amounts World uses for the guild house UI.
// A nil guild receives the initial service levels and default upkeep.
func (g *guild) houseServicesPacket() []byte {
	var id uint32
	upkeep, active := guildHouse.DefaultKeep, uint8(0)
	if g != nil {
		id = g.ID
		if g.BuyHouse != 0 {
			upkeep = g.KeepEtin
		}
		if g.houseEnabled() {
			active = 1
		}
	}
	w := packet.New(packet.Opcode(proto.GXGuildHouseServices))
	w.U32(id)
	w.U32(guildHouse.BuyMoney)
	w.U32(upkeep)
	w.U8(active)
	w.U8(uint8(len(guildHouse.Services)))
	for _, table := range guildHouse.Services {
		var level uint16
		if g != nil {
			level = min(g.npcLevel(table.Type), uint16(table.Max))
		}
		var nextCost uint16
		if level < uint16(table.Max) {
			nextCost = table.Cost[level+1]
		}
		w.U16(table.Type)
		w.U8(uint8(level))
		w.U16(table.Keep[level])
		w.U16(nextCost)
	}
	return w.MustFinish()
}

// guildNpcUpgrade is 0x0D25 (CGuild::NpcUpgrade), reached for guild ranks 1
// and 2 only. Called with guildMu held.
func (s *Session) guildNpcUpgrade(g *guild, npcType uint16, level uint8) {
	refuse := func(result uint8) { s.send(buildGuildNpcResult(result, npcType, level, 0), "0x0D25 guild NPC refused") }
	table := guildHouse.service(npcType)
	if !g.houseEnabled() || table == nil || level == 0 || level > table.Max {
		refuse(guildNpcRefused)
		return
	}
	available := g.Etin - min(g.Etin, g.KeepEtin)
	if available == 0 {
		refuse(guildNpcNoEtin)
		return
	}
	if table.Cost[level] == 0 {
		refuse(guildNpcRefused)
		return
	}
	cost := table.Cost[level]
	if available < uint32(cost) {
		refuse(guildNpcNoEtin)
		return
	}
	slot := -1
	for i, sl := range g.npcs {
		if sl.Type == npcType {
			slot = i
			break
		}
	}
	if slot >= 0 {
		if uint16(level) != g.npcs[slot].Level+1 {
			refuse(guildNpcBadLevel)
			return
		}
	} else {
		if level != 1 {
			refuse(guildNpcRefused)
			return
		}
		for i, sl := range g.npcs {
			if sl.Type == 0 {
				slot = i
				break
			}
		}
		if slot < 0 {
			refuse(guildNpcRefused)
			return
		}
	}
	// The change is staged on a copy and published after it is stored.
	next := *g
	next.npcs[slot] = guildNpcSlot{Type: npcType, Level: uint16(level)}
	next.calculateKeepEtin()
	next.Etin -= uint32(cost)
	if err := db.CommitGuildHouse(next.row(), &db.GuildNpcRow{GuildID: g.ID, Type: npcType, Level: level, Slot: uint8(slot)}, nil); err != nil {
		s.logErr("save guild NPC upgrade", err)
		return
	}
	g.npcs, g.KeepEtin, g.Etin = next.npcs, next.KeepEtin, next.Etin
	g.send(buildGuildNpcResult(guildNpcOK, npcType, level, uint8(slot)), "0x0D25 guild NPC upgraded", nil)
	g.send(guildID32(packet.GuildEtin, g.Etin), "0x0D23 guild etin", nil)
	g.send(g.houseServicesPacket(), "0x7F2D guild house services", nil)
	if f := g.houseStatusPacket(); f != nil {
		g.send(f, "0x0D33 guild house bonuses", nil)
	}
}

func buildGuildHouseBuy(result uint8, money uint32) []byte {
	w := packet.New(packet.GuildHouseBuy)
	w.U8(result)
	w.U32(money)
	return w.MustFinish()
}

// guildHouseBuy is the 0x0D34 branch of CUser::PacketGuild. Called with
// guildMu held.
func (s *Session) guildHouseBuy(g *guild, rank uint8) {
	ch := s.char
	switch {
	case g == nil || rank != guildRankMaster:
		s.send(buildGuildHouseBuy(guildHouseNoRight, ch.Money), "0x0D34 guild house refused")
		return
	case g.BuyHouse != 0:
		s.send(buildGuildHouseBuy(guildHouseOwned, ch.Money), "0x0D34 guild house refused")
		return
	case ch.Money < guildHouse.BuyMoney:
		s.send(buildGuildHouseBuy(guildHouseNoMoney, ch.Money), "0x0D34 guild house refused")
		return
	}
	next := *g
	next.BuyHouse = 1
	next.KeepEtin = guildHouse.DefaultKeep
	committed := s.inventoryOperationWithCommit(func() {
		s.char.Money -= guildHouse.BuyMoney
		s.send(buildGuildHouseBuy(guildHouseBought, s.char.Money), "0x0D34 guild house bought")
	}, func(change db.CharacterChange) error {
		return db.CommitGuildHouse(next.row(), nil, &change)
	})
	if committed {
		g.BuyHouse, g.KeepEtin = next.BuyHouse, next.KeepEtin
		if f := g.houseStatusPacket(); f != nil {
			g.send(f, "0x0D33 guild house owned", nil)
		}
		g.send(g.houseServicesPacket(), "0x7F2D guild house services", nil)
	}
}

// guildEtinValue is the etin an item donates: Type 43 with Special 52..55.
func guildEtinValue(it *db.Item) (value uint32, ok bool) {
	if it.Type != guildEtinType {
		return 0, false
	}
	def := db.MustDefs().Item(it.Type, it.TypeID)
	if def == nil {
		return 0, true
	}
	switch def.Special {
	case guildEtinSpecial:
		return uint32(it.Count), true
	case guildEtinSpecial + 1:
		return uint32(it.Count) * 10, true
	case guildEtinSpecial + 2:
		return uint32(it.Count) * 100, true
	case guildEtinSpecial + 3:
		return uint32(it.Count) * 1000, true
	}
	return 0, true
}

// guildDonate is the 0x0D35 branch: every etin item in the bags is consumed,
// its value added to the guild's etin and the return count incremented.
// Called with guildMu held.
func (s *Session) guildDonate(g *guild) {
	next := *g
	var total uint32
	committed := s.inventoryOperationWithCommit(func() {
		var taken [][2]uint8
		for bag := uint8(1); bag <= 5; bag++ {
			for slot := uint8(0); slot < 24; slot++ {
				i := findItem(s.items, bag, slot)
				if i < 0 {
					continue
				}
				value, ok := guildEtinValue(&s.items[i])
				if !ok {
					continue
				}
				total += value
				s.items = append(s.items[:i], s.items[i+1:]...)
				taken = append(taken, [2]uint8{bag, slot})
			}
		}
		w := packet.New(packet.GuildDonate)
		w.U8(uint8(len(taken)))
		for _, t := range taken {
			w.U8(t[0])
			w.U8(t[1])
		}
		s.send(w.MustFinish(), "0x0D35 guild etin donated")
	}, func(change db.CharacterChange) error {
		if total > 0 {
			next.Etin += total
			next.EtinReturn++
		}
		return db.CommitGuildHouse(next.row(), nil, &change)
	})
	if committed && total > 0 {
		g.Etin, g.EtinReturn = next.Etin, next.EtinReturn
		g.send(guildID32(packet.GuildEtin, g.Etin), "0x0D23 guild etin", nil)
	}
}

// settleGuildHouse is the house part of CGuildMgr::UpdateGuildPointRank for a
// ranked guild: below the upkeep the house is suspended (UseHouse 2),
// otherwise the upkeep is paid from the etin. Called with guildMu held.
func (g *guild) settleGuildHouse() {
	if g.gmHouse {
		g.forceGMHouse()
		return
	}
	if g.BuyHouse != 1 {
		return
	}
	wasActive := g.houseEnabled()
	if g.Etin < g.KeepEtin {
		g.UseHouse = 2
		if wasActive {
			g.send(g.houseServicesPacket(), "0x7F2D guild house services", nil)
		}
		return
	}
	g.Etin -= g.KeepEtin
	g.UseHouse = 1
	g.send(guildID32(packet.GuildEtin, g.Etin), "0x0D23 guild etin", nil)
	if !wasActive {
		g.send(g.houseServicesPacket(), "0x7F2D guild house services", nil)
	}
}

// inGuildHouseZone is CMap::IsGuildHouseMap for the character's zone.
func (s *Session) inGuildHouseZone() bool {
	return s.char != nil && mapCreateType(s.char.MapID) == guildHouseCreateType
}

// houseNpcRefusal answers a guild house service the guild cannot use with
// 0x0D28 {result, detail}, as PacketShop and PacketMaininterface do.
func (s *Session) houseNpcRefusal(result, detail uint8) {
	s.send(buildGuildBankResult(result, detail), "0x0D28 guild house service refused")
}

// shopBuyRate is the price factor of a shop type (1 weapon, 2 defense, 3
// general, 4 potion) at the guild's level of that service, or the refusal
// result and detail. Ranking no longer restricts any service, and NPC ids
// no longer matter: the discount belongs to the kind of shop.
func (g *guild) shopBuyRate(shopType uint8) (rate float64, result, detail uint8) {
	if shopType < 1 || shopType > 4 {
		return 1, 0, 0
	}
	level := min(g.npcLevel(uint16(shopType)<<8|1), 3)
	rate = guildHouse.ShopFactor[shopType][level]
	return rate, 0, 0
}

// gateRate is the teleport fee factor of the guild's gatekeeper level. The
// upgrades only change this fee; every gatekeeper stays usable.
func (g *guild) gateRate() float64 {
	return guildHouse.GateRate[min(g.npcLevel(guildNpcGate), 3)]
}

// roundHalfAway is the x87 ROUND of the original price products.
func houseRound(v float64) uint32 { return uint32(math.Floor(v + 0.5)) }

// smithIndex is the blacksmith NPC level clamped to the six-entry tables.
func (g *guild) smithIndex() int { return int(min(g.npcLevel(guildNpcSmith), 5)) }

// houseMix returns the gem-linking cost factor (RapiceMixDecreRate) and the
// success bonus (RapiceMixPercentRate, active house only) for the guild
// smith inside a house map; (1, 0) elsewhere.
func (s *Session) houseMix() (rate float64, bonus int) {
	p := s.operationPlayer()
	if !s.inGuildHouseZone() || p == nil {
		return 1, 0
	}
	guildMu.Lock()
	defer guildMu.Unlock()
	g := p.guild.guild
	if g == nil {
		return 1, 0
	}
	i := g.smithIndex()
	rate = guildHouse.MixRate[i]
	if g.houseEnabled() {
		bonus = int(guildHouse.MixPercent[i])
	}
	return rate, bonus
}

// houseShopRate resolves the guild's shop factor under guildMu, released on
// every path. A nil guild means the buyer has none.
func (s *Session) houseShopRate(shopType uint8) (g *guild, rate float64, result, detail uint8) {
	p := s.operationPlayer()
	if p == nil {
		return nil, 0, 0, 0
	}
	guildMu.Lock()
	defer guildMu.Unlock()
	if g = p.guild.guild; g == nil {
		return nil, 0, 0, 0
	}
	rate, result, detail = g.shopBuyRate(shopType)
	return g, rate, result, detail
}

// houseStatusPacket is the GX-only 0x0D33 notice (the EP5.4 client has no reader
// for it), sent to a guild's members only while the house is bought: the keep
// etin (bit 31 marks a GM guild) and, as whole percentages, the discounts the
// guild's current service levels give. The client shows them in tooltips and
// greys its Buy House option.
func (g *guild) houseStatusPacket() []byte {
	if g.BuyHouse == 0 {
		return nil
	}
	v := max(g.KeepEtin, 1) & 0x7fffffff
	if g.gmHouse {
		v |= 1 << 31
	}
	off := func(factor float64) uint8 { return uint8(min(100, max(0, houseRound((1-factor)*100)))) }
	w := packet.New(packet.GuildHouseKeepEtin)
	w.U32(v)
	for shop := uint8(1); shop <= 4; shop++ {
		level := min(g.npcLevel(uint16(shop)<<8|1), 3)
		w.U8(off(guildHouse.ShopFactor[shop][level]))
	}
	w.U8(off(guildHouse.GateRate[min(g.npcLevel(guildNpcGate), 3)]))
	smith := min(g.npcLevel(guildNpcSmith), 5)
	w.U8(uint8(min(100, max(0, guildHouse.MixPercent[smith]))))
	w.U8(off(guildHouse.MixRate[smith]))
	return w.MustFinish()
}
