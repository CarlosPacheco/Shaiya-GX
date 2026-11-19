package sdata

import (
	"encoding/binary"
	"fmt"
)

// GuildHouseService is one of the 36 records of data/Npc/GuildHouse.SData
// (Game.exe FUN_004cc920): a guild house NPC type at one level.
type GuildHouseService struct {
	Discount    uint8  // percentage shown for the level (shop discount, gate fee)
	Level       uint8  // 0-5
	Flag5       uint8  // unused by the loader's defaults (always 0)
	Flag6       uint8  // unused by the loader's defaults (always 0)
	RankLimit   uint8  // guild rank needed (0x1e = any ranked guild)
	Icon        uint16 // NPC icon index
	TextID      uint16 // sysmsg id of the NPC name (8011 gatekeeper ...)
	UpgradeEtin uint16
	KeepEtin    uint16
	Type        uint16 // 2 gate, 0x101 weapon, 0x201 defense, 0x301 general, 3 smith, 0x401 potion, 6 bank
}

// GuildHouseData is the whole file: price, default upkeep, the services and
// the NPC type ids per country (light, dark) and level for gatekeepers and
// the weapon, defense and general shops.
type GuildHouseData struct {
	Version     uint32
	BuyMoney    uint32
	DefaultKeep uint32
	Services    [36]GuildHouseService
	// NpcIDs[kind][country][level-1]: gate, weapon, defense, general.
	NpcIDs [4][2][3]uint32
}

const guildHouseSize = 12 + 36*15 + 4*6*4

// ParseGuildHouse reads the plain (unencrypted) GuildHouse.SData.
func ParseGuildHouse(b []byte) (*GuildHouseData, error) {
	if len(b) != guildHouseSize {
		return nil, fmt.Errorf("GuildHouse.SData: %d bytes, want %d", len(b), guildHouseSize)
	}
	d := &GuildHouseData{Version: binary.LittleEndian.Uint32(b), BuyMoney: binary.LittleEndian.Uint32(b[4:]), DefaultKeep: binary.LittleEndian.Uint32(b[8:])}
	o := 12
	for i := range d.Services {
		r := b[o : o+15]
		d.Services[i] = GuildHouseService{Discount: r[0], Level: r[1], Flag5: r[2], Flag6: r[3], RankLimit: r[4],
			Icon: binary.LittleEndian.Uint16(r[5:]), TextID: binary.LittleEndian.Uint16(r[7:]),
			UpgradeEtin: binary.LittleEndian.Uint16(r[9:]), KeepEtin: binary.LittleEndian.Uint16(r[11:]), Type: binary.LittleEndian.Uint16(r[13:])}
		o += 15
	}
	// The loader reads the tables as DAT_0199bd90, bd48, bd60, bd78: the file
	// order is gate, weapon, defense, general (verified against the ini ids).
	for kind := range d.NpcIDs {
		for country := range d.NpcIDs[kind] {
			for level := range d.NpcIDs[kind][country] {
				d.NpcIDs[kind][country][level] = binary.LittleEndian.Uint32(b[o:])
				o += 4
			}
		}
	}
	return d, nil
}

// Service returns the record of an NPC type at a level.
func (d *GuildHouseData) Service(npcType uint16, level uint8) *GuildHouseService {
	for i := range d.Services {
		if d.Services[i].Type == npcType && d.Services[i].Level == level {
			return &d.Services[i]
		}
	}
	return nil
}
