package app

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Guild house NPC windows. The Guild Manager is the Guild Master NPC (type 11)
// inside the house maps 51 and 52 (Game.exe 0x48e050 case 0xb: the map test
// 0x4cbca0 returns 1 for 0x33 and 0x34); the Etin Manager is the merchant NPC
// of type 100 (0x48e050: merchant 'd' opens case 0x73).
const etinManagerMerchant = 100

var houseMaps = map[uint16]bool{51: true, 52: true}

// Service display order of both windows (0x481450); index 5 is the unused
// apprentice slot.
var houseServiceTypes = [8]uint16{2, 0x101, 0x201, 6, 0x301, 0, 3, 0x401}
var houseServiceNames = [8]int{0x1428, 0x1423, 0x1424, 0x142d, 0x1425, 0, 0x1429, 0x1426}
var houseServiceMax = [8]uint8{3, 3, 3, 5, 3, 0, 5, 3}

const (
	houseConfirmNone = iota
	houseConfirmBuy
	houseConfirmDonate
	houseConfirmUpgrade
)

type houseView struct {
	npc                   uint32
	mode, selected        int
	listOffset            int
	confirm               int
	upType                uint16
	upLevel               uint8
	data                  *sdata.GuildHouseData
	dataTried             bool
	confirmBox            *ui.Message
	npcSerial, bankSerial uint64
	houseSerial           uint64
	donateSerial          uint64
	donatedValue          uint32
	requestedEtinFor      uint32
	availableCache        uint32
}

// houseKind reports which house window an NPC opens.
func (a *App) houseKind(e *game.Entity) (ui.HouseKind, bool) {
	if e == nil || a.guildHouseUI == nil || a.state == nil {
		return 0, false
	}
	if e.NpcType == 11 && houseMaps[a.state.MapID] {
		return ui.HouseGuildManager, true
	}
	if e.NpcType == 1 {
		if def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]; def.MerchantType == etinManagerMerchant {
			return ui.HouseEtinManager, true
		}
	}
	return 0, false
}

func (a *App) houseData() *sdata.GuildHouseData {
	h := &a.house
	if !h.dataTried && a.arc != nil {
		h.dataTried = true
		if b, err := a.arc.ReadFile("Npc/GuildHouse.SData"); err == nil {
			h.data, _ = sdata.ParseGuildHouse(b)
		}
	}
	return h.data
}

// houseServiceLevel is the guild's level of an NPC type (0xd24 / 0xd25).
func (a *App) houseServiceLevel(npcType uint16) uint8 {
	for _, n := range a.state.Guild.Npcs {
		if n.Type == npcType {
			return n.Level
		}
	}
	return 0
}

// houseRecord is FUN_004cbc20: the record of a type at a level, the level
// clamped to the type's maximum.
func houseRecord(d *sdata.GuildHouseData, npcType uint16, level uint8) *sdata.GuildHouseService {
	limit := uint8(3)
	if npcType == 6 || npcType == 3 {
		limit = 5
	}
	return d.Service(npcType, min(level, limit))
}

// houseView builds what the window draws for the current page.
func (a *App) houseBuildView(kind ui.HouseKind, e *game.Entity) *ui.HouseView {
	h := &a.house
	g := &a.state.Guild
	d := a.houseData()
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	v := &ui.HouseView{Portrait: a.npcPortrait(def), NpcName: def.Name, Selected: h.selected,
		CanManage: g.ID != 0 && g.Rank >= 1 && g.Rank <= 2, ListOffset: h.listOffset}
	if kind == ui.HouseGuildManager {
		v.Title = a.msg.Get(8009)
	} else {
		v.Title = a.msg.Get(8010)
		v.HouseOwned = g.HouseBought
		v.CanBuy = g.ID != 0 && g.Rank == 1 && !g.HouseBought
	}
	if d == nil {
		return v
	}
	etin := g.Etin
	keepDefault := d.DefaultKeep
	if keepDefault == 0 {
		keepDefault = 2000
	}
	total := keepDefault
	for i, t := range houseServiceTypes {
		if i == 5 {
			continue
		}
		if rec := houseRecord(d, t, a.houseServiceLevel(t)); rec != nil {
			total += uint32(rec.KeepEtin)
		}
	}
	available := uint32(0)
	if total < etin {
		available = etin - total
	}
	h.availableCache = available
	v.Etin, v.Available, v.Total = strconv.FormatUint(uint64(etin), 10), strconv.FormatUint(uint64(available), 10), strconv.FormatUint(uint64(total), 10)
	etinWord := a.msg.Get(8021)
	for i, t := range houseServiceTypes {
		if i == 5 {
			continue
		}
		level := a.houseServiceLevel(t)
		cur := houseRecord(d, t, level)
		next := houseRecord(d, t, level+1)
		if cur == nil || next == nil {
			v.Rows = append(v.Rows, ui.HouseRow{})
			continue
		}
		canLevel := level < houseServiceMax[i]
		row := ui.HouseRow{Show: true, Icon: int(cur.Icon), ShowUpgrade: canLevel}
		name := a.msg.Get(houseServiceNames[i])
		row.Title = fmt.Sprintf("%s %s%d", name, a.msg.Get(5211), level)
		if kind == ui.HouseEtinManager {
			feeOK := uint32(cur.KeepEtin) <= available
			row.FeeText = fmt.Sprintf("%s %d %s", a.msg.Get(8020), cur.KeepEtin, etinWord)
			row.RankOK, row.FeeOK, row.IconDim = true, feeOK, !feeOK
		} else {
			costOK := uint32(next.UpgradeEtin) <= available
			row.Title = fmt.Sprintf("%s %s%d", name, a.msg.Get(5211), next.Level)
			row.FeeText = fmt.Sprintf("%s %d %s", a.msg.Get(8048), next.UpgradeEtin, etinWord)
			row.RankOK, row.FeeOK, row.IconDim = true, costOK, !costOK
			row.UpgradeLook = costOK && canLevel
			row.UpgradeOn = uint32(next.KeepEtin) <= available && canLevel
		}
		v.Rows = append(v.Rows, row)
	}
	// The tooltip of the selected service fills the left text box (0x49de50).
	if h.mode == ui.HouseModeInfo {
		k := 0
		for i, t := range houseServiceTypes {
			if i == 5 {
				continue
			}
			if k == h.selected {
				if rec := houseRecord(d, t, a.houseServiceLevel(t)); rec != nil {
					v.Text = a.msg.Get(int(rec.TextID))
				}
				break
			}
			k++
		}
	}
	// 0x483210: the Etin manager's text box is filled with SysMsg 8031 at creation
	// and keeps it on every page except the service information.
	if v.Text == "" && kind == ui.HouseEtinManager && h.mode != ui.HouseModeInfo {
		v.Text = a.msg.Get(8031)
	}
	if kind == ui.HouseEtinManager && h.mode == ui.HouseModeList {
		v.List = a.houseGuildList()
	}
	return v
}

// houseGuildList is the country's ranked guilds in ranking order (0x482880
// mode 0): rank and points only for the thirty ranked guilds.
func (a *App) houseGuildList() []ui.HouseListRow {
	entries := make([]proto.GuildInfo, 0, len(a.state.Guild.Directory))
	for _, info := range a.state.Guild.Directory {
		entries = append(entries, info)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Rank != entries[j].Rank {
			return entries[i].Rank < entries[j].Rank
		}
		return entries[i].ID < entries[j].ID
	})
	out := make([]ui.HouseListRow, len(entries))
	for i, g := range entries {
		rank, points := "-", "-"
		if g.Rank <= 30 {
			rank, points = strconv.Itoa(int(g.Rank)), strconv.FormatUint(uint64(g.Points), 10)
		}
		out[i] = ui.HouseListRow{Rank: rank, Name: g.Name, Master: g.Master, Points: points}
	}
	return out
}

func (a *App) openHouse(e *game.Entity, kind ui.HouseKind) {
	h := &a.house
	h.npc, h.selected, h.listOffset, h.confirm = e.ID, 0, 0, houseConfirmNone
	h.mode = ui.HouseModeInfo
	if kind == ui.HouseEtinManager {
		h.mode = ui.HouseModeList
	}
	if a.world != nil {
		_ = a.world.Send(proto.GuildEtinPacket()) // FUN_0059c790 at construction
	}
}

func (a *App) closeHouse() {
	if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
		a.backToLogin("Unable to close guild manager.")
		return
	}
	a.shop.npcID = 0
	a.house.npc, a.house.confirm = 0, houseConfirmNone
	if a.house.confirmBox != nil {
		a.house.confirmBox.Visible = false
	}
}

func (a *App) updateHouse(in *platform.Input, w, h int, e *game.Entity, kind ui.HouseKind) bool {
	hv := &a.house
	if hv.npc != e.ID {
		a.openHouse(e, kind)
	}
	if hv.confirmBox == nil {
		hv.confirmBox, _ = ui.NewMessage(a)
	}
	x, y := a.guildHouseUI.Origin(w, h)
	captured := panelPointer(in, x, y, ui.GuildHouseWidth, ui.GuildHouseHeight)
	if hv.confirmBox != nil && hv.confirmBox.Visible {
		switch hv.confirmBox.Update(in, w, h) {
		case ui.MessageOK:
			a.houseConfirmed()
			fallthrough
		case ui.MessageCancel:
			hv.confirmBox.Visible, hv.confirm = false, houseConfirmNone
		}
		return true
	}
	view := a.houseBuildView(kind, e)
	act := a.guildHouseUI.Update(in, w, h, kind, hv.mode, view)
	if act.Tab >= 0 {
		hv.mode, hv.selected = act.Tab, 0
	}
	if act.Select >= 0 {
		hv.selected = act.Select
	}
	if act.Scroll != 0 {
		hv.listOffset = max(0, min(hv.listOffset+act.Scroll, max(0, len(view.List)-12)))
	}
	switch {
	case act.Buy && view.CanBuy:
		hv.confirm = houseConfirmBuy
		hv.confirmBox.Open(a.msg.Get(8032), true)
	case act.Donate:
		if a.houseEtinInInventory() {
			hv.confirm = houseConfirmDonate
			hv.confirmBox.Open(a.msg.Get(8033), true)
		} else {
			hv.confirmBox.Open(a.msg.Get(8051), false)
		}
	case act.Upgrade >= 0:
		a.houseAskUpgrade(act.Upgrade)
	}
	if act.Close || in.HasKey(platform.KeyEscape) {
		a.closeHouse()
		return true
	}
	return captured
}

// houseAskUpgrade opens the 0x1f71 confirmation for the service row.
func (a *App) houseAskUpgrade(row int) {
	d := a.houseData()
	hv := &a.house
	if d == nil {
		return
	}
	k := 0
	for i, t := range houseServiceTypes {
		if i == 5 {
			continue
		}
		if k == row {
			level := a.houseServiceLevel(t)
			if level >= houseServiceMax[i] {
				return
			}
			rec := houseRecord(d, t, level+1)
			if rec == nil {
				return
			}
			hv.upType, hv.upLevel, hv.confirm = t, level+1, houseConfirmUpgrade
			text := a.msg.Format(8049, map[string]string{"v": strconv.Itoa(int(rec.UpgradeEtin))})
			hv.confirmBox.Open(text, true)
			return
		}
		k++
	}
}

func (a *App) houseEtinInInventory() bool {
	for key, it := range a.state.Inventory {
		if key[0] >= 1 && it.Type == 43 {
			return true
		}
	}
	return false
}

func (a *App) houseConfirmed() {
	hv := &a.house
	var p []byte
	switch hv.confirm {
	case houseConfirmBuy:
		p = proto.GuildHouseBuyPacket()
	case houseConfirmDonate:
		p = proto.GuildDonatePacket()
		hv.donatedValue = 0
		for key, it := range a.state.Inventory {
			if key[0] >= 1 && it.Type == 43 {
				hv.donatedValue += uint32(it.Count) * [5]uint32{0, 1, 10, 100, 1000}[min(it.TypeID, 4)]
			}
		}
	case houseConfirmUpgrade:
		p = proto.GuildNpcUpgradePacket(hv.upType, hv.upLevel)
	}
	if p != nil && a.world != nil {
		if err := a.world.Send(p); err != nil {
			a.backToLogin("Unable to send guild request.")
		}
	}
}

func (a *App) drawHouse(w, h int, e *game.Entity, kind ui.HouseKind) {
	hv := &a.house
	a.guildHouseUI.Draw(a.sprites, a.font, a.white, w, h, kind, hv.mode, a.houseBuildView(kind, e))
	if hv.confirmBox != nil && hv.confirmBox.Visible {
		hv.confirmBox.Draw(a.sprites, a.font, w, h)
	}
}

// houseEvents shows the native messages of the guild house replies.
func (a *App) houseEvents() {
	g, hv := &a.state.Guild, &a.house
	if g.NpcSerial != hv.npcSerial {
		hv.npcSerial = g.NpcSerial
		switch g.NpcResult.Result {
		case 1:
			a.systemMessage(0, 8003, nil)
		case 2:
			a.systemMessage(0, 8004, nil)
		case 3:
			a.systemMessage(0, 8005, nil)
		case 4:
			a.systemMessage(0, 8006, nil)
		}
	}
	if g.BankResultSerial != hv.bankSerial {
		hv.bankSerial = g.BankResultSerial
		// 0x5979b0 -> 0x569890: every refusal prints its SysMsg whichever service
		// asked (bank, shop, gatekeeper, blacksmith); nothing else is awaited.
		if id := map[uint8]int{1: 8026, 2: 8027, 3: 8028, 4: 2038}[g.BankResult.Result]; id != 0 {
			fallback := map[uint8]string{1: "The guild must be ranked to use this service.", 2: "Insufficient guild permissions.", 3: "The guild's service NPC level is too low.", 4: "Your guild rank cannot do this."}[g.BankResult.Result]
			a.localizedNotice(0, id, nil, fallback)
		}
		a.shop.pending, a.inventory.pending, a.gate.pending = false, false, false
	}
	if g.HouseSerial != hv.houseSerial {
		hv.houseSerial = g.HouseSerial
		id := map[uint8]int{0: 8035, 1: 8029, 2: 303, 3: 2038, 4: 8030}[g.HouseResult]
		if id != 0 {
			a.systemMessage(0, id, nil)
		}
	}
	if g.DonateSerial != hv.donateSerial {
		hv.donateSerial = g.DonateSerial
		if g.Donated == 0 {
			a.systemMessage(0, 8051, nil)
		} else {
			a.systemMessage(0, 8052, map[string]string{"v": strconv.FormatUint(uint64(hv.donatedValue), 10)})
		}
	}
}

func (a *App) houseWindowOpen(e *game.Entity) bool {
	_, ok := a.houseKind(e)
	return ok
}

func (a *App) houseWindowOpenNow() bool {
	e := a.state.Entities[a.shop.npcID]
	return a.shop.npcID != 0 && a.houseWindowOpen(e)
}
