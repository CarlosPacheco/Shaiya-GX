package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
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

func (a *App) offerItemQuest(position [2]uint8, item sdata.ItemKey, startType uint8) bool {
	if a.questConfirm == nil || a.questConfirm.Visible || a.state == nil || a.state.Self.Dead || position[0] < 1 || position[0] > 5 {
		return false
	}
	if err := a.loadQuests(); err != nil {
		return false
	}
	id := a.itemQuestAt(item, startType)
	if id == 0 {
		return false
	}
	q := a.questDefs[id-1]
	text := questOfferText(q)
	a.questItemOffer = itemQuestOffer{id: id, position: position}
	a.questConfirm.OpenFull(q.Name+"\n\n"+text, true, a.font)
	return true
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
