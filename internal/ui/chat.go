package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/textwire"
	"slices"
	"strconv"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type ChatEntry struct {
	Items            proto.ChatItems
	ItemType, ItemID uint8
	Text             string
	System           bool
	Color            [4]float32
	Style            uint8
	Markup           bool
}

type chatPane struct {
	lines       []string
	runs        [][]chatRun
	offset      int
	arrows      [2]Button
	arrowRepeat [2]time.Time
	dragging    bool
	grab        int
	active      time.Time // last new line or interaction; drives auto-hide
}

// Optional auto-hide reveals histories while typing or for chatHold after a
// new line or interaction, then fades over chatFade. Persistent chat remains
// the default.
const (
	chatWidth       = 350
	chatPaneGap     = 2
	chatComposerGap = 2
	chatBottom      = 84
	chatHold        = 10 * time.Second
	chatFade        = time.Second
)

var chatNow = time.Now

func (c *ChatPanel) paneAlpha(i int, now time.Time) float32 {
	if !c.AutoHide || c.Typing || c.ModeOpen || c.OptionsOpen() || c.resizing || c.panes[i].dragging {
		return 1
	}
	idle := now.Sub(c.panes[i].active)
	if c.panes[i].active.IsZero() || idle >= chatHold+chatFade {
		return 0
	}
	if idle <= chatHold {
		return 1
	}
	return 1 - float32(idle-chatHold)/float32(chatFade)
}

func (c *ChatPanel) inputShown() bool {
	return c.Typing
}

// ShowHistory reveals both history panes and restarts their hide timer.
func (c *ChatPanel) ShowHistory() {
	now := chatNow()
	for i := range c.panes {
		c.panes[i].active = now
	}
}

// ChatPanel lays out both native 16px histories and the bottom-anchored composer.
type ChatPanel struct {
	LinkedItemCount                         func(proto.InventoryItem) uint16
	DrawLinkedItem                          func(item proto.InventoryItem, x, y, size int, alpha float32)
	DraftItems                              proto.ChatItems
	DrawItem                                func(typ, id uint8, x, y, size int, alpha float32)
	FocusRequested                          bool
	Resized, ChannelChanged, OptionsChanged bool
	RecipientFocus                          bool
	Recent                                  []string
	recipientsOpen                          bool
	recipientHeld                           int
	RecipientRequested                      bool
	Caret, RecipientCaret                   ChatCaret
	ViewportWidth                           int
	Filters                                 uint8
	Transparent                             bool
	appliedFilters                          uint8
	mouseX, mouseY                          int
	option                                  int
	optionPress                             int
	help, filter, icons, check              *render.Texture
	scroll, input                           *render.Texture
	ChannelNames                            [6]string
	Recipient                               string
	entries                                 []ChatEntry
	panes                                   [2]chatPane
	modeButton                              Button
	Typing, ModeOpen                        bool
	AutoHide                                bool
	Rows                                    [2]int
	resizePane, resizeY, resizeStart        int
	resizing                                bool
	Channel                                 uint8
	modeHeld                                int
}

func NewChatPanel(t Textures) (*ChatPanel, error) {
	ld := &loader{t: t}
	c := &ChatPanel{scroll: ld.get("scrollbar.png"), ChannelNames: chatChannelNames, input: ld.get("chat.png"), modeHeld: -1, optionPress: -1, Caret: ChatCaret{Position: -1}, Rows: [2]int{5, 5}}
	if optional, ok := t.(interface{ OptionalInterface(string) *render.Texture }); ok {
		c.help = optional.OptionalInterface("chat_config1.png")
		c.filter = optional.OptionalInterface("chat_config2.png")
		c.icons = optional.OptionalInterface("chat_icon.png")
		c.check = optional.OptionalInterface("chat_check.png")
	}
	c.modeButton = Button{Tex: c.input, DrawW: 64, DrawH: 22, HitW: 64, HitH: 22}
	for state, row := range []int{0, 32, 64, 64, 96} {
		c.modeButton.UV[state] = UV{0, float32(row) / 256, 64.0 / 512, float32(row+22) / 256}
	}
	return c, ld.err
}

func ChatOrigin(h int) int { return max(0, h-348) }

func (c *ChatPanel) paneGeometry(h, index int) (y, height, capacity int) {
	upper, lower := min(20, max(0, c.Rows[0])), min(20, max(0, c.Rows[1]))
	budget := max(0, (h-chatBottom-chatPaneGap)/16-6)
	if excess := max(0, upper+lower-budget); excess > 0 {
		cut := min(upper, (excess+1)/2)
		upper -= cut
		lower = max(0, lower-(excess-cut))
		upper = min(upper, max(0, budget-lower))
	}
	lowerHeight := (lower + 4) * 16
	upperHeight := (upper + 2) * 16
	y = max(0, h-chatBottom-lowerHeight)
	if index == 0 {
		return max(0, y-chatPaneGap-upperHeight), upperHeight, upper + 2
	}
	return y, lowerHeight, lower + 4
}

func (c *ChatPanel) SetEntries(f *Font, entries []ChatEntry) {
	if c.appliedFilters == c.Filters && slices.Equal(c.entries, entries) {
		return
	}
	var oldByPane, newByPane [2][]ChatEntry
	for _, entry := range c.entries {
		i := chatPaneIndex(entry)
		oldByPane[i] = append(oldByPane[i], entry)
	}
	for _, entry := range entries {
		i := chatPaneIndex(entry)
		newByPane[i] = append(newByPane[i], entry)
	}
	var overlap [2]int
	for i := range overlap {
		old, next := oldByPane[i], newByPane[i]
		n := min(len(old), len(next))
		for n > 0 && !slices.Equal(old[len(old)-n:], next[:n]) {
			n--
		}
		overlap[i] = n
	}
	c.appliedFilters = c.Filters
	added := [2]int{}
	c.entries = append(c.entries[:0], entries...)
	for i := range c.panes {
		c.panes[i].lines = c.panes[i].lines[:0]
		clear(c.panes[i].runs)
		c.panes[i].runs = c.panes[i].runs[:0]
	}
	var seen [2]int
	for _, entry := range entries {
		index := chatPaneIndex(entry)
		seen[index]++
		if chatFilter(entry.Style)&c.Filters != 0 {
			continue
		}
		pane := &c.panes[index]
		links := entry.Items.Len()
		runs, rows := wrapChatContent(f, entry.Text, entry.Color, entry.Markup, 316, entry.ItemType != 0, entry.Items[:links], c.linkedItemSuffix)
		if seen[index] > overlap[index] {
			added[index] += len(rows)
		}
		for row := range runs {
			for n := range runs[row] {
				if runs[row][n].icon && runs[row][n].item.Type == 0 {
					runs[row][n].itemType, runs[row][n].itemID = entry.ItemType, entry.ItemID
				}
			}
		}
		pane.runs = append(pane.runs, runs...)
		pane.lines = append(pane.lines, rows...)
	}
	if added[0]+added[1] > 0 {
		c.ShowHistory() // any new line brings up both panes together
	}
	for i := range c.panes {
		p := &c.panes[i]
		if p.offset > 0 {
			p.offset += added[i]
		}
		p.offset = min(p.offset, max(0, len(p.lines)-c.capacity(i)))
	}
}

func chatPaneIndex(entry ChatEntry) int {
	if entry.System {
		return 0
	}
	return 1
}

func (c *ChatPanel) controls(h int) {
	for i := range c.panes {
		p := &c.panes[i]
		y, height, capacity := c.paneGeometry(h, i)
		limit := max(0, len(p.lines)-capacity)
		for iArrow := range p.arrows {
			b := &p.arrows[iArrow]
			if b.DrawW == 0 {
				*b = scrollbarArrow(c.scroll, iArrow)
			}
			span := c.scrollSpan(i, height)
			b.OriginX, b.OriginY = chatScrollX, y+4+iArrow*(span-scrollbarArrowHitH)
			b.Disabled = limit == 0 || span < 2*scrollbarArrowHitH
		}
	}
}

// Contains keeps world picking behind the visible chat panels from handling
// the same pointer gesture. Geometry follows the current resized histories.
func (c *ChatPanel) Contains(mx, my, h int) bool {
	if c.recipientsOpen && c.recentHit(mx, my, h) >= 0 {
		return true
	}
	if c.optionHit(mx, my, h) || c.optionButton(mx, my, h) != 0 || c.recipientHit(mx, my, h) {
		return true
	}
	if mx < 0 || mx >= chatWidth {
		return false
	}
	if c.inputShown() && my >= chatInputY(h) && my < chatInputY(h)+22 {
		return true
	}
	now := chatNow()
	for i := range c.panes {
		if c.paneAlpha(i, now) == 0 {
			continue
		}
		y, height, _ := c.paneGeometry(h, i)
		if my >= y && my < y+height {
			return true
		}
	}
	return false
}

func (c *ChatPanel) Update(in *platform.Input, h int) bool {
	c.Resized = false
	c.ChannelChanged = false
	c.FocusRequested = false
	c.OptionsChanged = false
	c.RecipientRequested = false
	c.mouseX, c.mouseY = in.MouseX, in.MouseY
	if c.updateOptions(in, h) {
		return true
	}
	if c.updateRecipient(in, h) {
		return true
	}
	if c.resizing {
		c.Rows[c.resizePane] = min(20, max(0, c.resizeStart+(c.resizeY-in.MouseY)/16))
		if in.Clicked || !in.MouseDown {
			c.resizing = false
			c.Resized = true
		}
		return true
	}
	now := chatNow()
	if in.Pressed && in.MouseX >= 25 && in.MouseX < chatWidth {
		for pane := range c.panes {
			if c.paneAlpha(pane, now) == 0 {
				continue
			}
			y, _, _ := c.paneGeometry(h, pane)
			if in.MouseY >= y-4 && in.MouseY < y+5 {
				c.resizing = true
				c.resizePane, c.resizeY, c.resizeStart = pane, in.MouseY, c.Rows[pane]
				return true
			}
		}
	}
	c.controls(h)
	inputY := chatInputY(h)
	c.modeButton.OriginY = inputY
	if !c.inputShown() {
		c.ModeOpen, c.modeButton.held = false, false
	}
	if c.inputShown() && c.modeButton.Update(in) {
		c.ModeOpen = !c.ModeOpen
	}
	if c.ModeOpen {
		if in.HasKey(platform.KeyEscape) {
			c.ModeOpen = false
			return true
		}
		row := -1
		if in.MouseX >= 0 && in.MouseX < 120 && in.MouseY >= inputY-len(chatChannelOrder)*chatModePitch && in.MouseY < inputY {
			row = (in.MouseY - (inputY - len(chatChannelOrder)*chatModePitch)) / chatModePitch
		}
		if in.Pressed {
			c.modeHeld = row
		}
		if in.Clicked && row >= 0 && c.modeHeld == row {
			c.Channel = chatChannelOrder[row]
			c.ChannelChanged = true
			c.FocusRequested = true
			c.RecipientFocus = c.Channel == 5 && c.Recipient == ""
			c.ModeOpen = false
		}
		if in.Pressed && row < 0 && !c.modeButton.hit(in) {
			c.ModeOpen = false
		}
		return true
	}
	if c.inputShown() && in.MouseX >= 0 && in.MouseX < chatWidth && in.MouseY >= inputY && in.MouseY < inputY+22 {
		c.FocusRequested = in.Pressed && in.MouseX >= 64
		if c.FocusRequested {
			c.RecipientFocus = false
		}
		return true
	}

	captured := false
	for i := range c.panes {
		p := &c.panes[i]
		y, height, capacity := c.paneGeometry(h, i)
		limit := max(0, len(p.lines)-capacity)
		if c.paneAlpha(i, now) == 0 {
			p.dragging = false
			p.offset = max(0, min(p.offset, limit))
			continue
		}
		over := in.MouseX >= 0 && in.MouseX < chatWidth && in.MouseY >= y && in.MouseY < y+height
		if over {
			p.active = now // keep a visible pane up while the pointer is on it
		}
		if in.Pressed {
			p.dragging = false
		}
		if over {
			if in.Wheel > 0 {
				p.offset += 3
			} else if in.Wheel < 0 {
				p.offset -= 3
			}
			captured = captured || in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0
		}
		for i := range p.arrows {
			arrow := &p.arrows[i]
			if in.Pressed {
				arrow.held = false
			}
			arrow.Update(in)
			if arrow.held && arrow.hit(in) && in.MouseDown {
				if in.Pressed {
					p.offset += 1 - 2*i
					p.arrowRepeat[i] = now.Add(300 * time.Millisecond)
				} else if !now.Before(p.arrowRepeat[i]) {
					p.offset += 1 - 2*i
					p.arrowRepeat[i] = now.Add(60 * time.Millisecond)
				}
			} else if in.Pressed && in.Clicked && !arrow.Disabled && arrow.hit(in) {
				p.offset += 1 - 2*i
			}
		}
		top, travel, thumb := c.scrollGeometry(i, y, height, limit, p.offset)
		thumbHeight := chatThumbHeight
		if limit > 0 && travel > 0 && in.Pressed && in.MouseX >= chatScrollX && in.MouseX < chatScrollX+scrollbarHitW && in.MouseY >= top && in.MouseY < top+travel+thumbHeight {
			if in.MouseY >= thumb && in.MouseY < thumb+thumbHeight {
				p.grab, p.dragging = in.MouseY-thumb, true
			} else if in.MouseY < thumb {
				p.offset += capacity
			} else {
				p.offset -= capacity
			}
		}
		if p.dragging && travel > 0 && (in.MouseDown || in.Clicked) {
			p.offset = limit - (limit*max(0, min(travel, in.MouseY-top-p.grab))+travel/2)/travel
			captured = true
		}
		if p.dragging && (in.Clicked || !in.MouseDown) {
			p.dragging = false
			captured = true
		}
		p.offset = max(0, min(p.offset, limit))
	}
	return captured
}

func (c *ChatPanel) Draw(s *render.Sprites, f *Font, blank *render.Texture, h int, typing bool, text string) {
	c.Typing = typing
	c.controls(h)
	now := chatNow()
	for i := range c.panes {
		p := &c.panes[i]
		if c.inputShown() {
			p.active = now // closing the input starts the hold timer
		}
		a := c.paneAlpha(i, now)
		if a == 0 {
			continue
		}
		y, height, capacity := c.paneGeometry(h, i)
		if !c.Transparent || c.backgroundHovered(h) {
			s.Draw(blank, 0, float32(y), chatWidth, float32(height), 0, 0, 1, 1, [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 154.0 / 255 * a})
		}
		restoreClip := s.ClipRect(25, float32(y+1), 316, float32(height-2))
		p.offset = min(p.offset, max(0, len(p.lines)-capacity))
		end := max(0, len(p.lines)-p.offset)
		start := max(0, end-capacity)
		for row := start; row < end; row++ {
			yText := c.lineY(h, i, row, end, f.Height)
			for _, run := range p.runs[row] {
				if run.icon {
					if run.item.Type != 0 && c.DrawLinkedItem != nil && yText >= y+1 && yText+f.Height <= y+height-1 {
						c.drawLinkedItem(s, f, run.item, run.suffix, 25+run.x, yText, a)
					} else if c.DrawItem != nil && run.itemType != 0 {
						c.DrawItem(run.itemType, run.itemID, 25+run.x, yText, f.Height, a)
					}
					continue
				}
				tint := run.color
				if tint[3] == 0 {
					tint = white
				}
				tint[3] *= a
				f.Draw(s, run.text, 26+run.x, yText+1, [4]float32{0, 0, 0, tint[3]}, 316-run.x)
				f.Draw(s, run.text, 25+run.x, yText, tint, 316-run.x)
			}
		}
		restoreClip()
		c.drawScroll(s, i, y, height, max(0, len(p.lines)-capacity), a)
	}
	if c.optionButtonsShown() {
		c.drawOptionButtons(s, f, blank, h)
	}
	if !c.inputShown() {
		return
	}
	c.drawRecipient(s, f, blank, h)
	y := chatInputY(h)
	c.modeButton.OriginY = y
	c.modeButton.Draw(s)
	label := c.ChannelNames[min(int(c.Channel), 5)]
	// Centre the font cell in the 22px row; a fixed +6 left the larger chat
	// font touching the bottom border.
	textY := chatEntryTextY(y, f.Height)
	f.Draw(s, label, 7, textY, white, 44)
	// Align the composed strip to the history width and omit the language icon.
	s.Draw(c.input, 64, float32(y), 2, 22, 0, .5, 2.0/512, 150.0/256, white)
	s.Draw(c.input, 66, float32(y), chatWidth-68, 22, 2.0/512, .5, 270.0/512, 150.0/256, white)
	s.Draw(c.input, chatWidth-2, float32(y), 2, 22, 290.0/512, .5, 292.0/512, 150.0/256, white)
	shown := text
	count := c.DraftItems.Len()
	var suffixes [proto.MaxChatItems]string
	firstIconX := 331
	for i, item := range c.DraftItems[:count] {
		suffixes[i] = c.linkedItemSuffix(item)
		firstIconX -= f.Height + f.Width(suffixes[i]) + 4
	}
	available := 262
	if count > 0 {
		available = max(1, firstIconX-73)
	}
	x := firstIconX
	for i, item := range c.DraftItems[:count] {
		c.drawLinkedItem(s, f, item, suffixes[i], x, textY, 1)
		x += f.Height + f.Width(suffixes[i]) + 4
	}
	runes := []rune(shown)
	cursor := len(runes)
	if c.Caret.Position >= 0 {
		cursor = min(cursor, c.Caret.Position)
	}
	leftOfCaret := string(runes[:cursor])
	for len(leftOfCaret) > 0 && f.Width(leftOfCaret) >= available {
		leftOfCaret = textwire.DropFirst(leftOfCaret)
	}
	shown = leftOfCaret + string(runes[cursor:])
	restoreClip := s.ClipRect(69, float32(y+2), 264, 19)
	if typing && !c.RecipientFocus {
		first := cursor - len([]rune(leftOfCaret))
		drawChatSelection(s, f, blank, runes, first, c.Caret, 69, textY, available)
	}
	f.Draw(s, shown, 69, textY, white, available)
	if typing && !c.RecipientFocus && chatNow().UnixMilli()%1000 < 500 {
		s.Draw(blank, float32(69+f.Width(leftOfCaret)), float32(textY), 1, float32(f.Height), 0, 0, 1, 1, white)
	}
	restoreClip()
}

func (c *ChatPanel) linkedItemSuffix(item proto.InventoryItem) string {
	if c.LinkedItemCount != nil {
		if count := c.LinkedItemCount(item); count > 0 {
			return " (" + strconv.Itoa(int(count)) + ")"
		}
	}
	return ""
}

func (c *ChatPanel) drawLinkedItem(s *render.Sprites, f *Font, item proto.InventoryItem, suffix string, x, y int, alpha float32) {
	if c.DrawLinkedItem != nil {
		c.DrawLinkedItem(item, x, y, f.Height, alpha)
	}
	if suffix != "" {
		f.DrawOutlined(s, suffix, x+f.Height, y, [4]float32{1, 1, 1, alpha})
	}
}

// chatEntryTextY centres a font cell in the 22px entry row.
func chatEntryTextY(rowY, fontHeight int) int {
	return rowY + max(0, 22-fontHeight)/2
}

var chatChannelNames = [6]string{"Normal", "Trade", "Guild", "Party", "Area", "Whisper"}
var chatChannelOrder = [6]uint8{0, 5, 3, 2, 1, 4}

const chatModePitch = 15

func (c *ChatPanel) capacity(index int) int { _, _, n := c.paneGeometry(768, index); return n }

// DrawOverlay keeps chat popups above movable HUD panels.
func (c *ChatPanel) DrawOverlay(s *render.Sprites, f *Font, blank *render.Texture, h int) {
	c.drawOptions(s, f, blank, h)
	c.drawRecentRecipients(s, f, blank, h)
	if !c.Typing || !c.ModeOpen {
		return
	}
	top := chatInputY(h) - len(chatChannelOrder)*chatModePitch
	s.Draw(blank, 0, float32(top), 120, float32(len(chatChannelOrder)*chatModePitch), 0, 0, 1, 1, [4]float32{.114, .122, .141, .97})
	for i, channel := range chatChannelOrder {
		tint := white
		if channel == c.Channel {
			tint = RGBColor(0xffff00)
		}
		f.Draw(s, c.ChannelNames[channel], 8, top+i*chatModePitch, tint, 104)
	}
}

// The closed composer has no hit area. On opening it lines up with both
// edges of the lower history and leaves a two-pixel seam.
func chatInputY(h int) int { return h - chatBottom + chatComposerGap }

// Newest lines occupy the bottom row even in a partially filled history.
// Keep a full font cell and its shadow inside the flat frame; native frames
// extend with a four-pixel border that is absent in the chosen flat style.
func (c *ChatPanel) lineY(h, index, row, end, fontHeight int) int {
	y, height, _ := c.paneGeometry(h, index)
	return y + height - max(1, fontHeight) - 1 - (end-1-row)*16
}
