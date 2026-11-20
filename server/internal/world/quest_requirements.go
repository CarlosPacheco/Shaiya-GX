package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/npcquest"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func (s *Session) questGroupReady(required [6]uint8) bool {
	if s.player == nil {
		return false
	}
	partyMu.Lock()
	defer partyMu.Unlock()
	group := partyOf(s.player)
	if group == nil {
		return false
	}
	var jobs [6]uint8
	for _, member := range group.members {
		class := member.Snapshot().Class
		if class < 6 {
			jobs[class]++
		}
	}
	return npcquest.PartyMeetsJobs(len(group.members), jobs, required)
}

// Additional configurable requirements are optional. HG and VG at the quest
// level still resolve alternative prerequisite quest IDs as in EP5.
func (s *Session) questStartRestriction(q *questDef) string {
	ch := s.char
	if ch.Cg < q.CG || ch.Og < q.OG || ch.Ig < q.IG {
		return "Your character does not meet this quest's CG, OG or IG requirement."
	}
	if q.RequireParty && !s.questGroupReady(q.PartyJobs) {
		return "This quest requires a party with the specified classes."
	}
	return ""
}

func (s *Session) questFeedback(message string) {
	s.send(buildChatByName(packet.ChatMap, "Quest", message), "quest requirements")
}

// Guarded quest items are charged only when the selected quest is accepted.
func questItemNeedsAtomicStart(typ, id uint8) bool {
	for _, row := range questItemLinks[[2]uint8{typ, id}].Start {
		d := findQuestDef(row)
		if d != nil && d.StartType == 4 && d.StartItemType == typ && d.StartItemID == id &&
			(d.RequireParty || d.CG != 0 || d.OG != 0 || d.IG != 0 || d.TickStart != 0) {
			return true
		}
	}
	return false
}

func (r questResult) hasRequirements() bool {
	return r.NeedMobCount != 0 || r.NeedItemCount != 0 ||
		r.NeedTime != 0 || r.NeedHG != 0 || r.NeedVG != 0 || r.NeedOG != 0
}

func (q *activeQuest) expired(now time.Time) bool {
	return !q.endAt.IsZero() && !now.Before(q.endAt)
}

func (s *Session) questObjectives(q *activeQuest) (totals [3]int, ready bool) {
	d := q.def
	switch d.EndType {
	case 0:
		return totals, d.EndNpcType != 0
	case 1:
		totals = questItemTotals(d, s.items)
		for i, f := range d.FarmItems {
			if totals[i] < int(f.Count) {
				return totals, false
			}
		}
		return totals, true
	case 2:
		return totals, q.counters[0] >= d.MobCounts[0] && q.counters[1] >= d.MobCounts[1]
	case 3:
		return totals, q.counters[0] >= d.PvpKills
	}
	return totals, false
}

func (s *Session) questRewardEligible(q *activeQuest, r questResult, totals [3]int, now time.Time) bool {
	switch q.def.EndType {
	case 1:
		if r.NeedItemCount != 0 && (r.NeedItemID == 0 || int(r.NeedItemID) > len(totals) || totals[r.NeedItemID-1] < int(r.NeedItemCount)) {
			return false
		}
	case 2:
		if r.NeedMobCount != 0 && (r.NeedMobID == 0 || r.NeedMobID > 2 || q.counters[r.NeedMobID-1] < r.NeedMobCount) {
			return false
		}
	}
	ch := s.char
	return (r.NeedTime == 0 || !now.Before(q.startAt.Add(time.Duration(r.NeedTime)*time.Second))) &&
		(r.NeedHG == 0 || int(ch.Hg) >= int(r.NeedHG)) &&
		(r.NeedVG == 0 || ch.Vg >= r.NeedVG) &&
		ch.Og >= r.NeedOG
}

func (q *activeQuest) retryAutomatically() bool {
	if q.success != 0 || q.def.EndNpcType != 0 {
		return false
	}
	switch q.def.EndType {
	case 1:
		if q.def.MinMinutes != 0 || q.def.TickKeep != 0 {
			return true
		}
		for _, r := range q.def.Results {
			if r.NeedTime != 0 || r.NeedHG != 0 || r.NeedVG != 0 || r.NeedOG != 0 {
				return true
			}
		}
	case 2, 3:
		return true
	}
	return false
}

func (d *questDef) usesElapsedSeconds() bool {
	if d.TickKeep != 0 {
		return true
	}
	for _, r := range d.Results {
		if r.NeedTime != 0 {
			return true
		}
	}
	return false
}
