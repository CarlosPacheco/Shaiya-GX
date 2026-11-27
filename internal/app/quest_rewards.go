package app

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Quest reward rows, a Shaiya Core client feature absent from EP5.4: the
// take and turn-in pages list up to three rewards with their item icons, and
// a quest with a reward choice is answered by clicking the row instead of a
// blind prompt after turning in. The protocol is unchanged: the picked row
// is sent when the server asks for the choice (0x0907).

// questRewardCount is how many result rows a quest can grant: its choice
// count, or the single default result.
func questRewardCount(q sdata.Quest) int {
	return min(max(int(q.UserSelect), 1), len(q.Results))
}

func (a *App) setQuestRewards(q sdata.Quest, selectable bool) {
	v := a.npcQuestUI
	n := questRewardCount(q)
	v.Rewards = v.Rewards[:0]
	for i := range n {
		r := q.Results[i]
		row := ui.NPCQuestReward{Detail: a.msg.Format(12383, map[string]string{"v": fmt.Sprint(uint64(r.Experience)), "v2": fmt.Sprint(r.Money)})}
		if r.ItemType != 0 {
			row.Item = a.msg.Format(12299, map[string]string{"v": fmt.Sprint(r.ItemType), "v2": fmt.Sprint(r.ItemID)})
			if item := a.itemsItem(r.ItemType, r.ItemID); item != nil {
				if item.Name != "" {
					row.Item = item.Name
				}
				row.Icon = true
			}
		}
		v.Rewards = append(v.Rewards, row)
	}
	v.RewardSelectable = selectable && n > 1
	if !v.RewardSelectable || v.RewardSelected >= n {
		v.RewardSelected = -1
	}
	v.DrawRewardIcon = func(index, x, y int) {
		if index < n {
			r := q.Results[index]
			if item := a.itemsItem(r.ItemType, r.ItemID); item != nil {
				a.drawItemIcon(item, x, y, 32, [4]float32{1, 1, 1, 1})
			}
		}
	}
	a.questRewardQuest = q
}

// prepareQuestRewards fills the rows for the conversation's selected quest.
func (a *App) prepareQuestRewards(v *npcQuestView) {
	ui := a.npcQuestUI
	if v.selected == 0 || int(v.selected) > len(a.questDefs) || v.result != nil {
		ui.Rewards, ui.RewardSelectable = ui.Rewards[:0], false
		return
	}
	a.setQuestRewards(a.questDefs[v.selected-1], v.ending)
}

// questRewardTooltip shows the item tooltip of the hovered reward icon.
func (a *App) questRewardTooltip(w, h int) {
	i := a.npcQuestUI.RewardHover()
	q := a.questRewardQuest
	if i < 0 || i >= questRewardCount(q) {
		return
	}
	r := q.Results[i]
	if item := a.itemsItem(r.ItemType, r.ItemID); item != nil {
		a.drawItemDetails(item, nil, 1, 0, 0, w, h, "")
	}
}
