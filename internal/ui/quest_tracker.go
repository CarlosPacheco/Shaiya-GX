package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const (
	QuestTrackerWidth     = 294
	QuestTrackerHeight    = 180
	QuestTrackerDragWidth = QuestTrackerWidth - 30
	trackerListY          = 34
	trackerRowHeight      = 23
	trackerVisibleRows    = 6
	trackerListRight      = QuestTrackerWidth - 22
	trackerTextLeft       = 30
	trackerTextWidth      = trackerListRight - trackerTextLeft - 6
)

type QuestTrackerEntry struct {
	ID              uint16
	Title, Detail   string
	Ready, Expanded bool
}

type trackerRow struct {
	id              uint16 // Only title rows are clickable.
	text            string
	first, expanded bool
	ready           bool
}

// QuestTracker owns layout and pointer state. Active quests belong to the game state.
type QuestTracker struct {
	Position WindowPosition
	close    Button
	scroll   panelScroll
	rows     []trackerRow
	offset   int
	hovered  uint16
	pressed  uint16
}

func NewQuestTracker(t Textures) (*QuestTracker, error) {
	ld := &loader{t: t}
	return &QuestTracker{
		close:  nativeClose(ld.get("talk1_close_button.png"), QuestTrackerWidth-27, 5),
		scroll: newPanelScroll(ld.get("scrollbar.png"), QuestTrackerWidth-18, trackerListY, trackerRowHeight*trackerVisibleRows),
	}, ld.err
}

func (q *QuestTracker) Origin(w, h int) (int, int) {
	return q.Position.Origin(w, h, QuestTrackerWidth, QuestTrackerHeight, w-QuestTrackerWidth-18, min(214, h-QuestTrackerHeight-85))
}

func (q *QuestTracker) SetEntries(f *Font, entries []QuestTrackerEntry) {
	q.rows = q.rows[:0]
	for _, entry := range entries {
		for i, line := range wrapLines(f, entry.Title, trackerTextWidth) {
			q.rows = append(q.rows, trackerRow{id: entry.ID, text: line, first: i == 0, expanded: entry.Expanded, ready: entry.Ready})
		}
		if entry.Expanded && entry.Detail != "" {
			for _, line := range wrapLines(f, entry.Detail, trackerTextWidth) {
				q.rows = append(q.rows, trackerRow{text: line})
			}
		}
	}
	q.offset = min(q.offset, max(0, len(q.rows)-trackerVisibleRows))
}

// Reveal makes a quest selected from a scrolled journal visible on first open.
func (q *QuestTracker) Reveal(id uint16) {
	for i, row := range q.rows {
		if row.id == id {
			q.offset = min(i, max(0, len(q.rows)-trackerVisibleRows))
			return
		}
	}
}

func (q *QuestTracker) Reset() {
	q.rows = q.rows[:0]
	q.offset, q.hovered, q.pressed = 0, 0, 0
	q.close.held, q.close.state = false, StateNormal
	q.scroll.dragging = false
	for i := range q.scroll.arrows {
		q.scroll.arrows[i].held = false
	}
}

func (q *QuestTracker) Update(in *platform.Input, w, h int) (toggle uint16, close, captured bool) {
	x, y := q.Origin(w, h)
	q.close.OriginX, q.close.OriginY = x, y
	close = q.close.Update(in)
	over := in.MouseX >= x && in.MouseX < x+QuestTrackerWidth &&
		in.MouseY >= y && in.MouseY < y+QuestTrackerHeight
	if over && in.Wheel != 0 {
		if in.Wheel > 0 {
			q.offset--
		} else {
			q.offset++
		}
	}
	captured = q.scroll.dragging
	q.offset = q.scroll.Update(in, x, y, len(q.rows), trackerVisibleRows, q.offset)
	captured = captured || q.scroll.dragging
	q.hovered = 0
	if in.MouseX >= x+8 && in.MouseX < x+trackerListRight &&
		in.MouseY >= y+trackerListY && in.MouseY < y+trackerListY+trackerVisibleRows*trackerRowHeight {
		i := q.offset + (in.MouseY-y-trackerListY)/trackerRowHeight
		if i < len(q.rows) {
			q.hovered = q.rows[i].id
		}
	}
	if in.Pressed {
		q.pressed = q.hovered
	}
	if in.Clicked && q.hovered == q.pressed {
		toggle = q.hovered
	}
	if in.Clicked || (!in.MouseDown && !in.Pressed) {
		q.pressed = 0
	}
	return toggle, close, captured
}

func (q *QuestTracker) Draw(s *render.Sprites, f *Font, flat *render.Texture, w, h int) {
	x, y := q.Origin(w, h)
	rect := func(x, y, w, h int, color [4]float32) {
		s.Draw(flat, float32(x), float32(y), float32(w), float32(h), 0, 0, 1, 1, color)
	}
	edge := [4]float32{.38, .32, .24, .97}
	ink := [4]float32{.85, .80, .71, 1}
	rect(x, y, QuestTrackerWidth, QuestTrackerHeight, edge)
	rect(x+1, y+1, QuestTrackerWidth-2, QuestTrackerHeight-2, [4]float32{.064, .055, .046, .94})
	rect(x+1, y+1, QuestTrackerWidth-2, trackerListY-5, [4]float32{.16, .131, .097, .98})
	f.Draw(s, "Quest Tracker", x+11, y+8, ink, QuestTrackerDragWidth-16)
	if len(q.rows) == 0 {
		f.Draw(s, "No active quests", x+11, y+trackerListY+5, ink, trackerTextWidth)
	}
	for i, row := range q.rows[q.offset:min(len(q.rows), q.offset+trackerVisibleRows)] {
		top := y + trackerListY + i*trackerRowHeight
		color := ink
		if row.id != 0 {
			if row.id == q.hovered {
				rect(x+8, top, trackerListRight-8, trackerRowHeight, [4]float32{.26, .21, .14, .88})
			}
			color = white
			if row.ready {
				color = [4]float32{.69, .88, .56, 1}
			}
		}
		if row.first {
			sign := "+"
			if row.expanded {
				sign = "-"
			}
			f.Draw(s, sign, x+12, top+4, color, 14)
		}
		f.Draw(s, row.text, x+trackerTextLeft, top+4, color, trackerTextWidth)
	}
	q.scroll.Draw(s, x, y, len(q.rows), trackerVisibleRows, q.offset)
	q.close.OriginX, q.close.OriginY = x, y
	q.close.Draw(s)
}
