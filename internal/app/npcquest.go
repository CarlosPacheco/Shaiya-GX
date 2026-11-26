package app

import (
	"fmt"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/npcquest"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type npcQuestEntry struct {
	id     uint16
	ending bool
}
type npcQuestView struct {
	npc      uint32
	selected uint16
	ending   bool
	followUp bool
	offset   int
	result   *proto.QuestEndResult
}

// Reward values mirror Rapture's awardQuest; the result packet remains the
// authoritative record of what was actually granted.
func (a *App) questRewardLabel(q sdata.Quest, i int) string {
	r := q.Results[i]
	experience := uint64(r.Experience)
	reward := fmt.Sprintf("%d XP, %d gold", experience, r.Money)
	if r.ItemType == 0 {
		return fmt.Sprintf("%d: %s", i+1, reward)
	}
	name := a.msg.Format(12299, map[string]string{"v": fmt.Sprint(r.ItemType), "v2": fmt.Sprint(r.ItemID)})
	if a.arc != nil || a.items != nil {
		if item := a.itemsItem(r.ItemType, r.ItemID); item != nil && item.Name != "" {
			name = item.Name
		}
	}
	return fmt.Sprintf("%d: %s\n%s", i+1, name, reward)
}

func (a *App) questResultText(q sdata.Quest, result *proto.QuestEndResult) string {
	description := questText(q.Summary)
	if variant := int(result.Variant); variant < len(q.Completion) {
		if text := questText(q.Completion[variant]); text != "" {
			description = text
		}
	}
	if description != "" {
		description += "\n\n"
	}
	description += fmt.Sprintf("Received: %d XP, %d gold", result.Experience, result.Money)
	if result.Item[2] != 0 {
		name := a.msg.Format(12299, map[string]string{"v": fmt.Sprint(result.Item[2]), "v2": fmt.Sprint(result.Item[3])})
		if a.arc != nil || a.items != nil {
			if item := a.itemsItem(result.Item[2], result.Item[3]); item != nil && item.Name != "" {
				name = item.Name
			}
		}
		description += "\n" + name
	}
	return description
}

func questOfferText(q sdata.Quest) string {
	var parts []string
	if welcome := questText(q.Descriptions[1]); welcome != "" {
		parts = append(parts, welcome)
	}
	take := questText(q.Descriptions[0])
	if take == "" {
		take = questText(q.Summary)
	}
	if take != "" {
		parts = append(parts, take)
	}
	if q.CG != 0 || q.OG != 0 || q.IG != 0 {
		parts = append(parts, fmt.Sprintf("Minimum character attributes: CG %d, OG %d, IG %d.", q.CG, q.OG, q.IG))
	}
	return strings.Join(parts, "\n\n")
}

// questStartVisible mirrors Rapture's eligibility checks for presentation.
// Missing snapshots are treated as unknown; the server makes the decision.
func (a *App) questStartVisible(q sdata.Quest, e *game.Entity) bool {
	if q.StartType > 2 {
		return false
	}
	if (q.StartType != 0 || q.StartNPCType != 0) && (q.StartNPCType != e.NpcType || q.StartNPCID != e.TypeID) {
		return false
	}
	return a.questRequirementsVisible(q)
}

func (a *App) questRequirementsVisible(q sdata.Quest) bool {
	if q.RequireParty {
		var jobs [6]uint8
		for _, member := range a.state.Party.Members {
			if member.Class < 6 {
				jobs[member.Class]++
			}
		}
		if !npcquest.PartyMeetsJobs(len(a.state.Party.Members), jobs, q.PartyJobs) {
			return false
		}
	}
	if slot, ok := a.selfSlot(); ok {
		if slot.Level < q.MinLevel || slot.Level > q.MaxLevel || slot.Class >= 6 || q.Jobs[slot.Class] == 0 {
			return false
		}
		if (slot.Gender == 0 && !q.Male) || (slot.Gender == 1 && !q.Female) {
			return false
		}
		allowed := true
		switch q.Faction {
		case 0, 1:
			allowed = slot.Race == q.Faction
		case 2:
			allowed = slot.Race <= 1
		case 3:
			allowed = slot.Race == 3
		case 4:
			allowed = slot.Race == 2
		case 5:
			allowed = slot.Race == 2 || slot.Race == 3
		}
		if !allowed {
			return false
		}
	}
	if a.state.FinishedQuests != nil {
		ids := [3]uint16{q.PrevQuest, q.HG, uint16(q.VG)}
		if ids != [3]uint16{} {
			met := false
			for _, id := range ids {
				if id != 0 && a.state.FinishedQuests[id] != 0 {
					met = true
					break
				}
			}
			if !met {
				return false
			}
		}
	}
	if q.StartType == 2 && a.state.InventoryReady {
		found := false
		for key, item := range a.state.Inventory {
			if key[0] >= 1 && key[0] <= 5 && item.Type == q.StartItemType && item.TypeID == q.StartItemID && item.Count > 0 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (a *App) serviceQuestOrigin(w, h int, e *game.Entity) (int, int) {
	if e.NpcType == 2 && a.gateUI != nil {
		x, y := a.gateUI.Origin(w, h)
		return x + 145, y + 300
	}
	if e.NpcType == 3 && a.blacksmithUI != nil {
		x, y := a.blacksmithUI.Origin(w, h)
		return x + 145, y + 300
	}
	if e.NpcType == 6 && a.bankUI != nil {
		x, y := a.bankUI.Origin(w, h)
		return x + 145, y + 75
	}
	x, y := a.marketUI.Origin(w, h)
	return x + 145, y + 75
}
func (a *App) npcQuestEntries(e *game.Entity) []npcQuestEntry {
	def := a.npcDefinition(sdata.NPCKey{Type: e.NpcType, ID: e.TypeID})
	out := make([]npcQuestEntry, 0, len(def.StartQuests)+len(def.EndQuests))
	for _, id := range def.EndQuests {
		if _, ok := a.state.Quests[id]; ok {
			out = append(out, npcQuestEntry{id, true})
		}
	}
	for _, id := range def.StartQuests {
		if a.npcQuestCanOffer(id, e) {
			out = append(out, npcQuestEntry{id, false})
		}
	}
	return out
}
func (a *App) updateNPCQuest(in *platform.Input, w, h int, e *game.Entity) bool {
	v := &a.npcQuest
	a.npcQuestUI.Detail = v.selected != 0
	if v.npc != e.ID {
		*v = npcQuestView{npc: e.ID}
		a.npcQuestUI.ResetDescriptionScroll()
		if err := a.loadQuests(); err != nil {
			a.questMessageID(12150)
		}
	}
	entries := a.npcQuestEntries(e)
	if v.selected != 0 && v.result == nil {
		found := false
		for _, entry := range entries {
			if entry.id == v.selected && entry.ending == v.ending {
				found = true
				break
			}
		}
		if v.followUp && !v.ending && a.npcQuestCanOffer(v.selected, e) {
			found = true
		}
		if !found {
			// The request was accepted (or no longer applies): the conversation ends.
			a.closeNPC()
			return true
		}
	}
	if v.selected == 0 {
		a.closeNPC()
		return true
	}
	choices := 0
	if v.result == nil && v.selected != 0 && a.state.QuestChoosing && a.state.QuestChoice == v.selected && int(v.selected) <= len(a.questDefs) {
		choices = min(int(a.questDefs[v.selected-1].UserSelect), 3)
	}
	a.npcQuestUI.HasService = false
	a.npcQuestUI.ActionDisabled = v.ending && v.result == nil && v.selected > 0 && int(v.selected) <= len(a.questDefs) && !questMapReady(a.questDefs[v.selected-1], a.state.Quests[v.selected], a.state)
	action := a.npcQuestUI.Update(in, w, h, len(entries), v.offset, v.selected != 0, v.ending, choices)
	v.offset = action.Offset
	if action.Back {
		in.NotifyWindowClosed()
	}
	if action.Close || action.Back || in.CloseOnEscape() {
		if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
			a.backToLoginID(12011)
		} else {
			a.shop.npcID = 0
		}
		return true
	}
	if action.Row >= 0 && choices > 0 {
		if err := a.world.Send(proto.QuestChoosePacket(v.selected, uint8(action.Row))); err != nil {
			a.backToLoginID(12148)
		}
		return true
	}
	if action.Row >= 0 {
		entry := entries[action.Row]
		v.selected, v.ending, v.followUp = entry.id, entry.ending, false
		a.npcQuestUI.ResetDescriptionScroll()
	}
	if action.Accept && v.result != nil {
		if next := a.npcQuestFollowUp(v.result, e); next != 0 {
			*v = npcQuestView{npc: e.ID, selected: next, followUp: true}
			a.npcQuestUI.ResetDescriptionScroll()
			return true
		}
		a.closeNPC()
		return true
	}
	if action.Accept && v.selected != 0 {
		packet := proto.QuestStartPacket(e.ID, v.selected)
		if v.ending {
			packet = proto.QuestEndPacket(e.ID, v.selected)
		}
		if err := a.world.Send(packet); err != nil {
			a.backToLoginID(12149)
		}
		return true
	}
	x, y := a.npcQuestUI.Origin(w, h)
	height := 391
	if v.selected != 0 {
		height = 512
	}
	return panelPointer(in, x, y, ui.NPCQuestWidth, height)
}

func (a *App) npcQuestFollowUp(result *proto.QuestEndResult, e *game.Entity) uint16 {
	if result.Success == 0 || result.ID == 0 || int(result.ID) > len(a.questDefs) || result.Variant >= 3 {
		return 0
	}
	next := a.questDefs[result.ID-1].Results[result.Variant].NextQuest
	if a.npcQuestCanOffer(next, e) {
		return next
	}
	return 0
}

func (a *App) npcQuestCanOffer(id uint16, e *game.Entity) bool {
	if id == 0 || int(id) > len(a.questDefs) {
		return false
	}
	_, active := a.state.Quests[id]
	_, finished := a.state.FinishedQuests[id]
	return !active && !finished && a.questStartVisible(a.questDefs[id-1], e)
}
func (a *App) drawNPCQuest(w, h int, e *game.Entity) {
	v := &a.npcQuest
	a.npcQuestUI.Detail = v.selected != 0
	def := a.npcDefinition(sdata.NPCKey{Type: e.NpcType, ID: e.TypeID})
	entries := a.npcQuestEntries(e)
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = a.questName(entry.id)
	}
	title, description := "", ""
	var choices []string
	a.npcQuestUI.RewardText = ""
	if v.selected > 0 && int(v.selected) <= len(a.questDefs) {
		q := a.questDefinition(v.selected)
		title = q.Name
		description = questText(q.Summary)
		if !v.ending {
			description = questOfferText(q)
		}
		if v.result != nil {
			description = a.questResultText(q, v.result)
		} else if v.ending {
			active := a.state.Quests[v.selected]
			text := questText(q.Descriptions[2])
			if text == "" {
				text = questText(q.Descriptions[3])
			}
			if questMapReady(q, active, a.state) {
				text = questText(q.Completion[0])
			}
			if text != "" {
				description = text
			}
			description += "\n\n" + a.questObjective(q, active)
		}
		if v.result == nil && v.ending {
			a.npcQuestUI.RewardText = "Reward\n" + a.questRewardLabel(q, 0)
			if q.ResultType == 1 {
				description += questRewardRequirementsText(q, 0)
			}
		}
		if a.state.QuestChoosing && a.state.QuestChoice == v.selected {
			for i := 0; i < min(int(q.UserSelect), 3); i++ {
				choices = append(choices, a.questRewardLabel(q, i))
				description += questRewardRequirementsText(q, i)
			}
		}
	}
	a.npcQuestUI.Draw(a.sprites, a.font, w, h, def.Name, questText(def.Welcome), a.npcPortrait(def), names, v.offset, v.selected != 0, v.ending, title, description, choices)
}

func questRewardRequirementsText(q sdata.Quest, i int) string {
	r := q.Results[i]
	var requirements []string
	if r.NeedTime != 0 {
		requirements = append(requirements, fmt.Sprintf("%d seconds since acceptance", r.NeedTime))
	}
	for _, entry := range []struct {
		name  string
		value int
	}{{"HG", int(r.NeedHG)}, {"VG", int(r.NeedVG)}, {"OG", int(r.NeedOG)}} {
		if entry.value != 0 {
			requirements = append(requirements, fmt.Sprintf("%s ≥ %d", entry.name, entry.value))
		}
	}
	if q.EndType == 2 && r.NeedMobCount != 0 {
		requirements = append(requirements, fmt.Sprintf("%d kills for objective %d", r.NeedMobCount, r.NeedMobID))
	}
	if q.EndType == 1 && r.NeedItemCount != 0 {
		requirements = append(requirements, fmt.Sprintf("%d items for objective %d", r.NeedItemCount, r.NeedItemID))
	}
	if len(requirements) == 0 {
		return ""
	}
	return fmt.Sprintf("\n\nReward %d requires %s.", i+1, strings.Join(requirements, ", "))
}
