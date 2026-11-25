package ui

import (
	"fmt"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestQuestTrackerScrollAndStableClick(t *testing.T) {
	q := new(QuestTracker)
	f := new(Font)
	var entries []QuestTrackerEntry
	for id := uint16(1); id <= 12; id++ {
		entries = append(entries, QuestTrackerEntry{ID: id, Title: fmt.Sprint("Quest ", id)})
	}
	q.SetEntries(f, entries)
	q.Reveal(10)
	if q.offset != 6 || q.rows[q.offset].id != 7 {
		t.Fatalf("quest 10 not revealed: offset %d", q.offset)
	}
	x, y := q.Origin(1024, 768)
	x, y = x+50, y+trackerListY+10
	q.Update(&platform.Input{MouseX: x, MouseY: y, Wheel: 1}, 1024, 768)
	if q.offset != 5 {
		t.Fatal("wheel did not scroll the quest list")
	}

	q.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
	q.SetEntries(f, entries[1:])
	if id, _, _ := q.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768); id != 0 {
		t.Fatalf("release activated a different row after a quest disappeared: %d", id)
	}
	q.Reset()
	if q.offset != 0 || len(q.rows) != 0 {
		t.Fatal("closing left stale scroll or quest rows")
	}
}
