package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type itemQuestOffer struct {
	id       uint16
	position [2]uint8
}

func isQuestInventoryItem(typ uint8) bool { return typ >= 27 && typ <= 29 }

func (a *App) itemQuestAt(item sdata.ItemKey, startType uint8) uint16 {
	for _, id := range a.questItemLinks[item].Start {
		if id == 0 || int(id) > len(a.questDefs) {
			continue
		}
		q := a.questDefs[id-1]
		if q.StartType != startType || q.StartItemType != item.Type || q.StartItemID != item.TypeID {
			continue
		}
		if _, active := a.state.Quests[id]; active {
			continue
		}
		if _, finished := a.state.FinishedQuests[id]; finished {
			continue
		}
		if a.questRequirementsVisible(q) {
			return id
		}
	}
	return 0
}

// offerItemQuest is Game.exe 0x4d0120. A hidden item (start type 3) offers
// its quest as soon as it reaches the inventory, and a usable one (type 4)
// when it is used; the offer opens the quest conversation's take page
// (window state 0x70), not a message box. Requirement notices are shown only
// when the player uses the item (argument 1), never on pickup.
func (a *App) offerItemQuest(position [2]uint8, item sdata.ItemKey, startType uint8, manual bool) bool {
	if a.npcQuestUI == nil || a.questItemOffer.id != 0 || a.state == nil || a.state.Self.Dead || position[0] < 1 || position[0] > 5 {
		return false
	}
	if err := a.loadQuests(); err != nil {
		return false
	}
	id := a.itemQuestAt(item, startType)
	if id == 0 {
		if manual {
			a.itemQuestRefusal(item)
		}
		return false
	}
	if a.shop.npcID != 0 {
		a.closeNPC()
	}
	a.questItemOffer = itemQuestOffer{id: id, position: position}
	a.npcQuestUI.ResetDescriptionScroll()
	return true
}

// itemQuestRefusal shows the native notice for the first linked quest that
// cannot start: active 465, finished 466, prerequisite 464, country or sex
// 246, job 248, level 311.
func (a *App) itemQuestRefusal(item sdata.ItemKey) {
	for _, id := range a.questItemLinks[item].Start {
		if id == 0 || int(id) > len(a.questDefs) {
			continue
		}
		q := a.questDefs[id-1]
		if q.StartType < 3 || q.StartType > 4 || q.StartItemType != item.Type || q.StartItemID != item.TypeID {
			continue
		}
		message := 0
		slot, known := a.selfSlot()
		_, active := a.state.Quests[id]
		_, finished := a.state.FinishedQuests[id]
		switch {
		case active:
			message = 465
		case finished:
			message = 466
		case !a.questPrerequisitesMet(q):
			message = 464
		case known && (!raceAllowedForQuest(q.Faction, slot.Race) || slot.Gender == 0 && !q.Male || slot.Gender == 1 && !q.Female):
			message = 246
		case known && (slot.Class >= 6 || q.Jobs[slot.Class] == 0):
			message = 248
		case known && (slot.Level < q.MinLevel || slot.Level > q.MaxLevel):
			message = 311
		}
		if message != 0 && a.tr(message) != "" {
			a.questMessageID(message)
		}
		return
	}
}

func (a *App) updateItemQuest(in *platform.Input, w, h int) bool {
	offer := a.questItemOffer
	if offer.id == 0 || int(offer.id) > len(a.questDefs) {
		a.questItemOffer = itemQuestOffer{}
		return false
	}
	ui := a.npcQuestUI
	ui.HasService, ui.ActionDisabled = false, false
	a.setQuestRewards(a.questDefs[offer.id-1], false)
	action := ui.Update(in, w, h, 0, 0, true, false)
	if action.Back {
		in.NotifyWindowClosed()
	}
	if action.Close || action.Back || in.CloseOnEscape() {
		a.questItemOffer = itemQuestOffer{}
		return true
	}
	if action.Accept || in.HasKey(platform.KeyEnter) {
		a.questItemOffer = itemQuestOffer{}
		a.acceptItemQuest(offer)
		return true
	}
	x, y := ui.Origin(w, h)
	return panelPointer(in, x, y, npcQuestWindowWidth, 512)
}

func (a *App) drawItemQuest(w, h int) {
	offer := a.questItemOffer
	if offer.id == 0 || int(offer.id) > len(a.questDefs) || a.npcQuestUI == nil {
		return
	}
	q := a.questDefs[offer.id-1]
	a.setQuestRewards(q, false)
	// 0x4d0120 sets the window's face id (+0x37f8) to 55 and 0x4db2ba draws
	// data/interface/NpcFace/<id>.tga: the empty dark orb, not the item icon.
	face := a.assets.Texture("interface/NpcFace/55.png")
	a.npcQuestUI.Draw(a.sprites, a.font, w, h, "", "", face, nil, 0, true, false, q.Name, questOfferText(q))
	a.questRewardTooltip(w, h)
}

func (a *App) acceptItemQuest(offer itemQuestOffer) {
	if a.world == nil || a.state.Self.Dead || offer.id == 0 || int(offer.id) > len(a.questDefs) {
		return
	}
	q := a.questDefs[offer.id-1]
	pos := offer.position
	item := a.state.Inventory[pos]
	if item.Count == 0 || a.itemQuestAt(sdata.ItemKey{Type: item.Type, TypeID: item.TypeID}, q.StartType) != offer.id {
		return
	}
	if q.StartType == 4 {
		if err := a.world.Send(proto.UseItemPacket(pos[0], pos[1])); err != nil {
			a.backToLoginID(12173)
			return
		}
	}
	if err := a.world.Send(proto.QuestStartPacket(uint32(q.StartItemID), offer.id)); err != nil {
		a.backToLoginID(12149)
	}
}

const npcQuestWindowWidth = 256
