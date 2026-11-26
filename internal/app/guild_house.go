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

// Service icons and strings in the original UI order.
var houseServiceDisplay = [...]struct {
	npcType             uint16
	icon, title, textID int
}{
	{0x002, 24, 0x1428, 8011},  // gatekeeper
	{0x101, 19, 0x1423, 8012},  // weapon shop
	{0x201, 53, 0x1424, 8013},  // armor shop
	{0x006, 248, 0x142d, 8015}, // warehouse
	{0x301, 247, 0x1425, 8014}, // general shop
	{0x003, 233, 0x1429, 8017}, // blacksmith
	{0x401, 192, 0x1426, 8018}, // potion shop
}

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
	confirmGuild          uint32
	confirmBox            *ui.Message
	npcSerial, bankSerial uint64
	houseSerial           uint64
	donateSerial          uint64
	donatedValue          uint32
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

// houseView builds what the window draws for the current page.
func (a *App) houseBuildView(kind ui.HouseKind, e *game.Entity) *ui.HouseView {
	h := &a.house
	g := &a.state.Guild
	services := g.HouseServices
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	v := &ui.HouseView{Portrait: a.npcPortrait(def), NpcName: def.Name, Selected: h.selected,
		CanManage: g.ID != 0 && g.Rank >= 1 && g.Rank <= 2, ListOffset: h.listOffset}
	if kind == ui.HouseGuildManager {
		v.Title = a.interfaceText(8009)
	} else {
		v.Title = a.interfaceText(8010)
		v.HouseOwned = g.HouseBought
		v.CanBuy = g.ID != 0 && g.Rank == 1 && !g.HouseBought && services != nil
	}
	if services != nil {
		available := g.Etin - min(g.Etin, services.Upkeep)
		v.Etin = strconv.FormatUint(uint64(g.Etin), 10)
		v.Available = strconv.FormatUint(uint64(available), 10)
		v.Total = strconv.FormatUint(uint64(services.Upkeep), 10)
		etinWord, levelWord := a.interfaceText(8021), a.interfaceText(5211)
		v.Rows = make([]ui.HouseRow, len(houseServiceDisplay))
		for i, display := range houseServiceDisplay {
			service, ok := services.Service(display.npcType)
			if !ok {
				continue
			}
			name := a.interfaceText(display.title)
			level, fee, feeLabel := service.Level, service.Upkeep, 8020
			row := ui.HouseRow{Show: true, Icon: display.icon, RankOK: true}
			if kind == ui.HouseGuildManager {
				row.ShowUpgrade = service.NextCost != 0
				if row.ShowUpgrade {
					level++
				}
				fee, feeLabel = service.NextCost, 8048
				row.UpgradeOn = v.CanManage && services.Active && row.ShowUpgrade && uint32(fee) <= available
			}
			row.Title = fmt.Sprintf("%s %s%d", name, levelWord, level)
			row.FeeText = fmt.Sprintf("%s %d %s", a.interfaceText(feeLabel), fee, etinWord)
			row.FeeOK = uint32(fee) <= available
			row.IconDim = !row.FeeOK
			v.Rows[i] = row
		}
	}
	if h.mode == ui.HouseModeInfo && h.selected >= 0 && h.selected < len(houseServiceDisplay) {
		v.Text = a.msg.Get(houseServiceDisplay[h.selected].textID)
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
		a.backToLoginID(12098)
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
		a.houseAskBuy()
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

func (a *App) houseAskBuy() {
	g, hv := &a.state.Guild, &a.house
	hv.confirm, hv.confirmGuild = houseConfirmBuy, g.ID
	text := a.msg.Format(12377, map[string]string{"v": ui.GoldText(g.HouseServices.BuyGold)})
	hv.confirmBox.Open(text, true)
}

func (a *App) houseAskUpgrade(row int) {
	g, hv := &a.state.Guild, &a.house
	npcType := houseServiceDisplay[row].npcType
	service, _ := g.HouseServices.Service(npcType)
	hv.upType, hv.upLevel, hv.confirm, hv.confirmGuild = npcType, service.Level+1, houseConfirmUpgrade, g.ID
	text := a.msg.Format(8049, map[string]string{"v": strconv.Itoa(int(service.NextCost))})
	hv.confirmBox.Open(text, true)
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
	g, hv := &a.state.Guild, &a.house
	var p []byte
	switch hv.confirm {
	case houseConfirmBuy:
		if g.ID == hv.confirmGuild && g.Rank == 1 && !g.HouseBought && g.HouseServices != nil {
			p = proto.GuildHouseBuyPacket()
		}
	case houseConfirmDonate:
		p = proto.GuildDonatePacket()
		hv.donatedValue = 0
		for key, it := range a.state.Inventory {
			if key[0] >= 1 && it.Type == 43 {
				hv.donatedValue += uint32(it.Count) * [5]uint32{0, 1, 10, 100, 1000}[min(it.TypeID, 4)]
			}
		}
	case houseConfirmUpgrade:
		if g.ID == hv.confirmGuild {
			p = proto.GuildNpcUpgradePacket(hv.upType, hv.upLevel)
		}
	}
	if p != nil && a.world != nil {
		if err := a.world.Send(p); err != nil {
			a.backToLoginID(12105)
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
			a.localizedNotice(0, id, nil)
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
