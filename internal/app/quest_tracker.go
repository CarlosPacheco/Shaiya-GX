package app

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type questTrackerView struct {
	open, inventoryReady             bool
	expanded                         map[uint16]bool
	cachedState                      *game.State
	questRevision, inventoryRevision uint64
	refreshAt                        time.Time
	entries                          []ui.QuestTrackerEntry
}

func (a *App) showQuestTracker(id uint16) {
	if a.questTrackerUI == nil || a.questTracker.open {
		return
	}
	a.questTracker.open = true
	a.questTracker.expanded = map[uint16]bool{id: true}
	a.refreshQuestTracker()
	a.questTrackerUI.Reveal(id)
}

func (a *App) closeQuestTracker() {
	a.questTracker = questTrackerView{}
	if a.questTrackerUI != nil {
		a.questTrackerUI.Reset()
	}
}

func (a *App) refreshQuestTracker() {
	v, s := &a.questTracker, a.state
	now := time.Now()
	if v.cachedState == s && v.questRevision == s.QuestMapRevision && v.inventoryRevision == s.QuestInventoryRevision &&
		v.inventoryReady == s.InventoryReady && (v.refreshAt.IsZero() || now.Before(v.refreshAt)) {
		return
	}
	v.cachedState, v.questRevision, v.inventoryRevision = s, s.QuestMapRevision, s.QuestInventoryRevision
	v.inventoryReady = s.InventoryReady
	v.refreshAt = time.Time{}
	v.entries = v.entries[:0]
	for _, id := range a.questIDs() {
		if id == 0 {
			continue
		}
		entry := ui.QuestTrackerEntry{ID: id, Title: a.questName(id), Expanded: v.expanded[id]}
		if int(id) <= len(a.questDefs) {
			def, q := a.questDefs[id-1], s.Quests[id]
			entry.Ready = a.questObjectivesReady(def, q)
			if entry.Expanded {
				entry.Detail = a.questObjective(def, q)
				if hint := a.questTimeHint(def, q); hint != "" {
					entry.Detail = hint + "\n" + entry.Detail
					v.refreshAt = now.Add(time.Second)
				}
			}
		}
		v.entries = append(v.entries, entry)
	}
	a.questTrackerUI.SetEntries(a.font, v.entries)
}

func (a *App) updateQuestTracker(in *platform.Input, w, h int) bool {
	q := a.questTrackerUI
	if q == nil || !a.questTracker.open {
		return false
	}
	a.refreshQuestTracker()
	id, close, captured := q.Update(in, w, h)
	if close {
		a.closeQuestTracker()
		a.playUISound("ui_window.ogg")
		return true
	}
	if id != 0 {
		v := &a.questTracker
		if v.expanded[id] {
			delete(v.expanded, id)
		} else {
			v.expanded[id] = true
		}
		v.cachedState = nil
	}
	x, y := q.Origin(w, h)
	return captured || panelPointer(in, x, y, ui.QuestTrackerWidth, ui.QuestTrackerHeight)
}

func (a *App) drawQuestTracker(w, h int) {
	q := a.questTrackerUI
	if q == nil || !a.questTracker.open {
		return
	}
	a.refreshQuestTracker()
	x, y := q.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.QuestTrackerWidth, ui.QuestTrackerHeight)
	q.Draw(a.sprites, a.font, a.white, w, h)
}
