package app

import (
	"fmt"
	"log"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

const worldPacketsPerFrame = 128
const worldPacketTimeBudget = 2 * time.Millisecond

// Preserve wire order and leave capacity for input and rendering during bursts.
func (a *App) pumpWorld() {
	world := a.world
	if world == nil {
		return
	}
	deadline := time.Now().Add(worldPacketTimeBudget)
	for i := 0; i < worldPacketsPerFrame && a.world == world; i++ {
		select {
		case p, ok := <-world.Packets:
			if !ok {
				a.backToLogin("Disconnected from the server.")
				return
			}
			a.applyWorldPacket(p)
			if time.Now().After(deadline) {
				return
			}
		default:
			return
		}
	}
}

func (a *App) applyWorldPacket(p []byte) {
	if proto.Op(p) == proto.GXPlayerFeedback {
		if id, kind, err := proto.DecodePlayerFeedback(p); err == nil && a.state != nil && (id != a.state.SelfID || kind == proto.PlayerLevelFeedback) {
			a.queuePlayerFeedback(id, kind)
		}
		return
	}
	if proto.Op(p) == proto.GXFirecracker {
		a.confirmedFirecracker(p)
		return
	}
	if proto.Op(p) == proto.GXAuction {
		a.applyAuction(p)
		return
	}
	if proto.Op(p) == proto.GXForgeQuote {
		a.confirmedForgeQuote(p)
		return
	}
	if a.state == nil {
		return
	}
	wasRaid := a.state.Party.Raid
	previousForm := a.state.PlayerForms[a.state.SelfID]
	var previousAppearance scene.Appearance
	if proto.Op(p) == proto.ItemBagToBag || proto.Op(p) == proto.ChangeAppearance || proto.Op(p) == proto.EnchantAdd || proto.Op(p) == proto.CloakAdd || proto.Op(p) == proto.CloakRemove {
		if slot, ok := a.selfSlot(); ok {
			previousAppearance = a.appearance(&slot)
		}
	}
	previousMoney := a.state.Details.Money
	previousPhase := a.state.Phase
	var previousLevel uint16
	if proto.Op(p) == proto.LevelUpSelf {
		if ch, ok := a.selfSlot(); ok {
			previousLevel = ch.Level
		}
	}
	ev, err := a.state.Apply(p)
	if err != nil {
		log.Printf("packet: %v", err)
		return
	}
	if previousPhase == game.PhaseLoading && a.state.Phase == game.PhaseInWorld {
		a.rememberCharacterSlot()
	}
	if ev.Op == proto.GXOxygen {
		a.oxygenAt = a.clock
		return
	}
	a.confirmedPlayerFeedback(p, previousLevel)
	if ev.Op == proto.GXEvolutionBar {
		a.confirmEvolutionBar()
	}
	if ev.Op == proto.GXFormAppearance && a.wv != nil && previousForm != a.state.PlayerForms[a.state.SelfID] &&
		(previousForm.Kind == proto.FormDisguise || a.state.PlayerForms[a.state.SelfID].Kind == proto.FormDisguise) {
		a.refreshSelf(a.wv)
	}
	if ev.Op == proto.ChatRaid {
		a.showRaidNotice(a.state.RaidNotice)
		return
	}
	if ev.ChatQuiet {
		return
	}
	a.syncUnionState(wasRaid)
	a.drainPartyNotices()
	if ev.Op == proto.Logout && a.state.Phase == game.PhaseCharSelect {
		a.playerFeedback = a.playerFeedback[:0]
		if a.menuPending && a.menuIntent == ui.MenuExit {
			a.Quit = true
		}
		a.menuOpen, a.menuPending = false, false
		a.releaseWorld()
	}
	if ev.Op == proto.AccountFaction && a.factionUI != nil {
		a.factionUI.Busy = false
	}
	a.applyGuildFeedback(p, ev.Op)
	if a.state == nil {
		return
	}
	a.confirmedExchange(p)
	if a.state == nil {
		return
	}
	a.confirmedGemLink(p)
	if a.state == nil {
		return
	}
	a.confirmedGemRemove(p)
	if a.state == nil {
		return
	}
	a.confirmedEnchant(p)
	if a.state == nil {
		return
	}
	a.confirmedCompose(p)
	if a.state == nil {
		return
	}
	a.confirmedGatekeeper(p)
	if a.state == nil {
		return
	}
	a.confirmedVehicle(p)
	if a.state == nil {
		return
	}
	if ev.Op >= proto.MyShopBegin && ev.Op <= proto.MyShopStock {
		a.confirmedPersonalMarket(ev.Op)
	}
	if ev.Op == proto.GXTeleportStone {
		a.confirmedTeleportStone(p)
	}
	if ev.Op == proto.GXSummonRune || ev.Op == proto.UserLeave {
		a.confirmedSummonRune(p)
	}
	a.confirmedItems(ev.Op)
	a.confirmedSystemChat(p, previousMoney)
	a.confirmedPickup(p)
	a.confirmedMall(p)
	a.confirmedAccount(p)
	a.confirmedQuest(p)
	if a.inventory.selected {
		if _, ok := a.state.Inventory[a.inventory.source]; !ok {
			a.inventory.selected = false
		}
	}
	a.applyWorldPresentation(p, ev.Op)
	if ev.Op == proto.SetStatPoint || ev.Op == proto.StatsReset {
		a.status.pending = false
		if a.statusUI != nil {
			a.statusUI.Clear()
		}
	}
	if ev.Op == proto.LearnNewSkill {
		a.skills.pending = false
		result := a.state.LastLearn
		if result.Result == 0 {
			a.loadSkills()
			cost := uint16(a.skillDefs[sdata.SkillKey{ID: result.ID, Level: result.Level}].Points)
			a.state.Details.SkillPoint -= min(a.state.Details.SkillPoint, cost)
		}
	}
	a.applyInventoryFeedback(p, ev.Op, previousAppearance)
	a.houseEvents()
	if ev.Op == proto.GuildBankResult && a.bank.guildPending {
		a.bank.guildPending = false
		a.inventory.pending = false
	}
	a.confirmedPosition(p)
	if proto.Op(p) == proto.CharacterSkillBar && a.screen == screenWorld {
		a.reconcileLocalActions()
	}
	if ev.MapLoadAck && a.world != nil {
		a.world.Send(proto.MapLoadedPacket()) // acknowledge initial entry and every map transfer
	}
}

func (a *App) applyGuildFeedback(p []byte, op proto.Opcode) {
	if op == proto.GuildInvite {
		a.guild.expires = a.clock + 30
	}
	switch op {
	case proto.GuildListBegin, proto.GuildList, proto.GuildListEnd, proto.GuildNew, proto.GuildDisbanded:
		a.guild.directoryDirty = true
	case proto.GuildOnlineList, proto.GuildOfflineList, proto.GuildMemberAdded, proto.GuildApplicantList, proto.GuildApplicant, proto.GuildApplicantRemoved, proto.GuildStatus, proto.GuildJoinResult, proto.GuildCreate, proto.GuildLeave, proto.GuildExpel:
		a.guild.rosterDirty = true
	}
	a.confirmedDuel(p)
	a.confirmedGuildDuel(p)
	a.confirmedGuildRanking(p)
	if op == proto.GuildRemark {
		a.guild.remarkPending = false
		if a.state.Guild.RemarkResult == 1 {
			a.guild.directoryDirty = true
			a.questMessage("Guild notice updated.")
		} else {
			a.questMessage("Guild notice was refused.")
		}
	}
}

func (a *App) applyWorldPresentation(p []byte, op proto.Opcode) {
	if op == proto.SkillCastStarted && a.state.Self.Mounted && a.state.LastCast.Source == a.state.SelfID {
		return
	}
	a.confirmedAnimation(p)
	a.confirmedGesture(op)
	a.confirmedSkills(op)
	a.confirmedPvPReward(op)
	a.confirmedSkillFeedback(op)
	if op == proto.SkillDetectInfo {
		v := a.state.LastDetectInfo
		a.questMessage(fmt.Sprintf("Detection: level %d, defense element %s.", v.Level, elementName(uint8(v.DefenseElement), a.msg)))
	}
	a.confirmedBuffs(op)
	a.confirmedSocial(op)
	a.confirmedEffects(p)
	a.confirmedSound(p)
	a.confirmedBlessing(op)
	a.confirmedChatBubble(op)
}

func (a *App) applyInventoryFeedback(p []byte, op proto.Opcode, previousAppearance scene.Appearance) {
	if op == proto.UseItem && proto.NewReader(p).U32() == a.state.SelfID {
		a.inventory.pending = false
		if !a.compose.pending {
			a.inventory.message = "Item used."
			if a.state.LastItemUse.Type == 0 {
				a.inventory.message = "This item cannot be used."
			}
		}
	}
	if op == proto.ShopBuy || op == proto.ShopSell {
		a.shop.pending = false
		a.inventory.pending = false
		if op == proto.ShopBuy && a.state.LastShop.Result == 0 {
			r := a.state.LastShop
			key := [2]uint8{r.Bag, r.Slot}
			it := a.state.Inventory[key]
			if def := a.itemsItem(it.Type, it.TypeID); def != nil {
				it.Quality = def.Quality
				a.state.Inventory[key] = it
			}
		}
	}
	if op == proto.ItemBagToBag {
		if a.bank.guildPending {
			items, err := proto.DecodeItemMove(p)
			if err == nil && [2]uint8{items[0].Bag, items[0].Slot} == a.bank.sourceMove && [2]uint8{items[1].Bag, items[1].Slot} == a.bank.destMove {
				a.inventory.pending = false
				a.bank.guildPending = false
			}
		} else {
			a.inventory.pending = false
		}
		if slot, ok := a.selfSlot(); ok && a.appearance(&slot) != previousAppearance && a.wv != nil {
			a.refreshSelf(a.wv)
		}
	}
	if (op == proto.ChangeAppearance || op == proto.EnchantAdd || op == proto.CloakAdd || op == proto.CloakRemove) && a.wv != nil {
		if slot, ok := a.selfSlot(); ok && a.appearance(&slot) != previousAppearance {
			a.refreshSelf(a.wv)
		}
	}

}
