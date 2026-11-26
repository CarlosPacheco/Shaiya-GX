package proto

// Guild house packets (0x0D23-0x0D25, 0x0D33-0x0D35, 0x0D38); layouts from
// the EP5.4 readers FUN_005977d0, FUN_00597940, FUN_005978f0, FUN_00597780 and
// the writers FUN_0059c730/760/790/7c0.

// GXGuildHouseServices follows a GuildEtin request and updates on house changes.
const GXGuildHouseServices Opcode = 0x7F2D

type GuildHouseServices struct {
	GuildID uint32
	BuyGold uint32
	Upkeep  uint32
	Active  bool
	Rows    []GuildHouseService
}

type GuildHouseService struct {
	Type     uint16
	Level    uint8
	Upkeep   uint16
	NextCost uint16 // zero when the next level is unavailable
}

func DecodeGuildHouseServices(p []byte) (GuildHouseServices, error) {
	r := NewReader(p)
	v := GuildHouseServices{GuildID: r.U32(), BuyGold: r.U32(), Upkeep: r.U32(), Active: r.Bool()}
	n := int(r.U8())
	if n > r.Remaining()/7 {
		return GuildHouseServices{}, ErrShort
	}
	v.Rows = make([]GuildHouseService, n)
	for i := range v.Rows {
		v.Rows[i] = GuildHouseService{Type: r.U16(), Level: r.U8(), Upkeep: r.U16(), NextCost: r.U16()}
	}
	return v, r.Err()
}

func (v *GuildHouseServices) Service(npcType uint16) (GuildHouseService, bool) {
	for _, row := range v.Rows {
		if row.Type == npcType {
			return row, true
		}
	}
	return GuildHouseService{}, false
}

// GuildNpc is one guild house NPC slot of 0x0D24 or the result of 0x0D25.
type GuildNpc struct {
	Type  uint16
	Level uint8
	Slot  uint8
}

// GuildNpcUpgradeResult is the 0x0D25 reply.
type GuildNpcUpgradeResult struct {
	Result uint8
	Npc    GuildNpc
}

// GuildHouseBuyResult is the 0x0D34 reply: 0 bought, 1 owned, 2 no money, 3 no right.
type GuildHouseBuyResult struct {
	Result uint8
	Money  uint32
}

func DecodeGuildNpcList(p []byte) ([]GuildNpc, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/4 {
		return nil, ErrShort
	}
	out := make([]GuildNpc, n)
	for i := range out {
		out[i] = GuildNpc{Type: r.U16(), Level: r.U8(), Slot: r.U8()}
	}
	return out, r.Err()
}

func DecodeGuildNpcUpgrade(p []byte) (GuildNpcUpgradeResult, error) {
	r := NewReader(p)
	v := GuildNpcUpgradeResult{Result: r.U8()}
	v.Npc = GuildNpc{Type: r.U16(), Level: r.U8(), Slot: r.U8()}
	return v, r.Err()
}

func DecodeGuildHouseBuy(p []byte) (GuildHouseBuyResult, error) {
	r := NewReader(p)
	v := GuildHouseBuyResult{Result: r.U8(), Money: r.U32()}
	return v, r.Err()
}

// DecodeGuildDonation returns the (bag, slot) pairs the server consumed.
func DecodeGuildDonation(p []byte) ([][2]uint8, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/2 {
		return nil, ErrShort
	}
	out := make([][2]uint8, n)
	for i := range out {
		out[i] = [2]uint8{r.U8(), r.U8()}
	}
	return out, r.Err()
}

func GuildNpcUpgradePacket(npcType uint16, level uint8) []byte {
	w := NewWriter(GuildNpcUpgrade)
	w.U16(npcType)
	w.U8(level)
	return w.Bytes()
}

// GuildHouseBuyPacket and GuildDonatePacket carry the unused u32 the original sends.
func GuildHouseBuyPacket() []byte { w := NewWriter(GuildHouseBuy); w.U32(0); return w.Bytes() }
func GuildDonatePacket() []byte   { w := NewWriter(GuildDonate); w.U32(0); return w.Bytes() }
