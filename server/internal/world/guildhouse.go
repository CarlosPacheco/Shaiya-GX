package world

import (
	"path/filepath"
	"strconv"
	"strings"

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
	guildNpcOK        = 0
	guildNpcNoEtin    = 1
	guildNpcBadLevel  = 2
	guildNpcRefused   = 3
	guildNpcRankLimit = 4
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
	Cost [7]uint16
	Keep [6]uint16
}

// guildHouseSettings is everything InitGuildHouseIniFile loads, starting from
// the executable's own defaults.
type guildHouseSettings struct {
	BuyMoney, DefaultKeep uint32
	Services              [guildNpcServices]guildServiceTable // gate, weapon, defense, general, smith, potion, bank
	ShopRate, PotionRate  [4]float64                          // g_GuildNpcShopDecreRate, g_GuildNpcPotionShopDecreRate
	GateRate              [6]float64                          // g_GuildNpcGateKeeperDecreRate (MoveDecreRate)
	RepairRate            [6]float64                          // g_GuildNpcSmithRepairDecreRate
	MixRate               [6]float64                          // g_GuildNpcSmithRapiceMixDecreRate
	MixPercent            [6]int32                            // g_GuildNpcSmithRapiceMixPercentRate
	BankSlots             [6]uint16                           // g_GuildNpcBankMaxSlot (PermitSlot)
	// NpcIDs is g_GuildNpcIdShopOnLevel: [country][gate, weapon, defense,
	// general][level 0-3][3 NPC type ids].
	NpcIDs [2][4][4][3]uint16
}

func defaultGuildHouse() guildHouseSettings {
	return guildHouseSettings{
		BuyMoney: 50000000, DefaultKeep: 2000,
		Services: [guildNpcServices]guildServiceTable{
			{Type: guildNpcGate, Cost: [7]uint16{0, 1000, 1500, 2000}, Keep: [6]uint16{0, 200, 500, 800}},
			{Type: guildNpcWeapon, Cost: [7]uint16{0, 100, 200, 500}},
			{Type: guildNpcDefense, Cost: [7]uint16{0, 100, 200, 500}},
			{Type: guildNpcGeneral, Cost: [7]uint16{0, 50, 100, 200}},
			{Type: guildNpcSmith, Cost: [7]uint16{0, 500, 1000, 2000, 10000, 15000}, Keep: [6]uint16{0, 100, 200, 500, 800, 1000}},
			{Type: guildNpcPotion, Cost: [7]uint16{0, 2000, 5000, 10000}, Keep: [6]uint16{0, 50, 200, 500}},
			{Type: guildNpcBank, Cost: [7]uint16{0, 200, 500, 1000, 2000, 5000}, Keep: [6]uint16{0, 100, 200, 500, 800, 1000}},
		},
		ShopRate:   [4]float64{0.99, 0.98, 0.95, 0.9},
		PotionRate: [4]float64{0.9, 0.8, 0.7, 0.6},
		RepairRate: [6]float64{0.99, 0.95, 0.95, 0.9, 0.9, 0.9},
		MixRate:    [6]float64{1, 1, 0.95, 0.95, 0.95, 0.9},
		MixPercent: [6]int32{0, 0, 0, 0, 1, 2},
		BankSlots:  [6]uint16{40, 80, 120, 160, 200, 240},
		NpcIDs: [2][4][4][3]uint16{
			{{{}, {37}, {38}, {39}}, {{}, {196}, {197}, {198}}, {{}, {200}, {201}, {202}}, {{}, {200}, {201}, {202}}},
			{{{}, {41}, {42}, {43}}, {{}, {207}, {208}, {209}}, {{}, {211}, {212}, {213}}, {{}, {200}, {201}, {202}}},
		},
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

// fillU16 stores up to len(dst) tokens from the start of dst; missing tokens
// keep their value.
func fillU16(dst []uint16, value string) {
	for i, t := range iniTokens(value) {
		if i >= len(dst) {
			return
		}
		dst[i] = uint16(iniInt(t))
	}
}

func fillF64(dst []float64, value string) {
	for i, t := range iniTokens(value) {
		if i >= len(dst) {
			return
		}
		if v, err := strconv.ParseFloat(t, 64); err == nil {
			dst[i] = v
		} else {
			dst[i] = 0
		}
	}
}

func fillI32(dst []int32, value string) {
	for i, t := range iniTokens(value) {
		if i >= len(dst) {
			return
		}
		dst[i] = int32(iniInt(t))
	}
}

// parseGuildHouse applies GuildHouse.ini over the executable defaults. The
// original runs GetPrivateProfileString for each key and keeps the default
// when it is absent.
func parseGuildHouse(ini iniFile) guildHouseSettings {
	h := defaultGuildHouse()
	get := func(section, key string) string { return ini[section][key] }
	if v := get("DEFAULTSET", "HOUSEBUYMONEY"); v != "" {
		if n := iniInt(v); n > 0 {
			h.BuyMoney = uint32(n)
		}
	}
	if v := get("DEFAULTSET", "HOUSEKEEPETIN"); v != "" {
		h.DefaultKeep = uint32(iniInt(v))
	}
	table := func(service int, section string, costCount, keepCount int) {
		if v := get(section, "UPGRADEETIN"); v != "" {
			fillU16(h.Services[service].Cost[1:1+costCount], v)
		}
		if v := get(section, "KEEPETIN"); v != "" && keepCount > 0 {
			fillU16(h.Services[service].Keep[1:1+keepCount], v)
		}
	}
	table(0, "GATEKEEPER", 3, 3)
	table(1, "SHOPWEAPON", 3, 0)
	table(2, "SHOPDEFEN", 3, 0)
	table(3, "GENERAL", 3, 0)
	table(4, "SMITH", 5, 5)
	table(5, "SHOPPOSION", 3, 3)
	table(6, "BANK", 5, 5)
	if v := get("GATEKEEPER", "MOVEDECRERATE"); v != "" {
		fillF64(h.GateRate[:], v)
	}
	if v := get("SHOPDEFAULT", "BUYDECRERATE"); v != "" {
		fillF64(h.ShopRate[:], v)
	}
	if v := get("SHOPDEFAULT", "BUYPOTIONDECRERATE"); v != "" {
		fillF64(h.PotionRate[:], v)
	}
	if v := get("BANK", "PERMITSLOT"); v != "" {
		fillU16(h.BankSlots[:], v)
	}
	if v := get("SMITH", "REPAIRDECRERATE"); v != "" {
		fillF64(h.RepairRate[:], v)
	}
	if v := get("SMITH", "RAPICEMIXDECRERATE"); v != "" {
		fillF64(h.MixRate[:], v)
	}
	if v := get("SMITH", "RAPICEMIXPERCENTRATE"); v != "" {
		fillI32(h.MixPercent[:], v)
	}
	for country, section := range [2]string{"NPCLEVEL_LIGHT", "NPCLEVEL_DARK"} {
		for kind, name := range [4]string{"GATEKEEPER", "SHOPWEAPON", "SHOPDEFEN", "SHOPGENERAL"} {
			for level := 1; level <= 3; level++ {
				if v := get(section, name+strconv.Itoa(level)); v != "" {
					h.NpcIDs[country][kind][level][0] = uint16(iniInt(v))
				}
			}
		}
	}
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

// houseEnabled is CGuild::EnableGuildHouse: ranked, active and bought.
func (g *guild) houseEnabled() bool {
	return g.Rank < 31 && g.UseHouse == 1 && g.BuyHouse != 0
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

// guildNpcUpgrade is 0x0D25 (CGuild::NpcUpgrade), reached for guild ranks 1
// and 2 only. Called with guildMu held.
func (s *Session) guildNpcUpgrade(g *guild, npcType uint16, level uint8) {
	refuse := func(result uint8) { s.send(buildGuildNpcResult(result, npcType, level, 0), "0x0D25 guild NPC refused") }
	if !g.houseEnabled() || level == 0 || level > 5 {
		refuse(guildNpcRefused)
		return
	}
	switch npcType {
	case guildNpcGate, guildNpcWeapon, guildNpcDefense:
		if level > 3 {
			refuse(guildNpcRefused)
			return
		}
	case guildNpcSmith:
		if g.Rank > 14 {
			refuse(guildNpcRankLimit)
			return
		}
	case guildNpcPotion:
		if g.Rank > 6 {
			refuse(guildNpcRankLimit)
			return
		}
		if level > 3 {
			refuse(guildNpcRefused)
			return
		}
	case guildNpcBank, guildNpcGeneral:
	default:
		refuse(guildNpcRefused)
		return
	}
	available := int32(g.Etin - g.KeepEtin)
	if available < 1 {
		refuse(guildNpcNoEtin)
		return
	}
	table := guildHouse.service(npcType)
	if table == nil || table.Cost[level] == 0 {
		refuse(guildNpcRefused)
		return
	}
	cost := table.Cost[level]
	if available < int32(cost) {
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
	s.send(guildID32(packet.GuildEtin, g.Etin), "0x0D23 guild etin")
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
	if g.BuyHouse != 1 {
		return
	}
	if g.Etin < g.KeepEtin {
		g.UseHouse = 2
		return
	}
	g.Etin -= g.KeepEtin
	g.UseHouse = 1
	g.send(guildID32(packet.GuildEtin, g.Etin), "0x0D23 guild etin", nil)
}
