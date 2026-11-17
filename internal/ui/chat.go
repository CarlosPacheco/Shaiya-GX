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
	lines    []string
	runs     [][]chatRun
	offset   int
	arrows   [2]Button
	dragging bool
	active   time.Time // last new line or interaction; drives auto-hide
}

// Optional auto-hide reveals histories while typing or for chatHold after a
// new line or interaction, then fades over chatFade. Persistent chat remains
// the default.
const (
	chatHold = 10 * time.Second
	chatFade = time.Second
)

var chatNow = time.Now

func (c *ChatPanel) paneAlpha(i int, now time.Time) float32 {
	if !c.AutoHide || c.Typing || c.ModeOpen || c.resizing || c.panes[i].dragging {
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

func (c *ChatPanel) inputShown() bool { return !c.AutoHide || c.Typing || c.ModeOpen }

// ShowHistory reveals both history panes and restarts their hide timer.
func (c *ChatPanel) ShowHistory() {
	now := chatNow()
	for i := range c.panes {
		c.panes[i].active = now
	}
}

// ChatPanel follows Phoenix chat_hud.gd and ep4_chat.gd: flat persistent
// histories and 16px rows, with the native 64px selector and 292px input strip.
type ChatPanel struct {
	LinkedItemCount                  func(proto.InventoryItem) uint16
	DrawLinkedItem                   func(item proto.InventoryItem, x, y, size int, alpha float32)
	DraftItems                       proto.ChatItems
	DrawItem                         func(typ, id uint8, x, y, size int, alpha float32)
	FocusRequested                   bool
	Resized, ChannelChanged          bool
	scroll, input                    *render.Texture
	ChannelNames                     [6]string
	Recipient                        string
	entries                          []ChatEntry
	panes                            [2]chatPane
	modeButton                       Button
	Typing, ModeOpen                 bool
	AutoHide                         bool
	Rows                             [2]int
	resizePane, resizeY, resizeStart int
	resizing                         bool
	Channel                          uint8
	modeHeld                         int
}

func NewChatPanel(t Textures) (*ChatPanel, error) {
	ld := &loader{t: t}
	c := &ChatPanel{scroll: ld.get("scrollbar.png"), ChannelNames: chatChannelNames, input: ld.get("chat.png"), modeHeld: -1, Rows: [2]int{5, 5}}
	c.modeButton = Button{Tex: c.input, DrawW: 64, DrawH: 22, HitW: 64, HitH: 22}
	for state, row := range []int{0, 32, 64, 64, 96} {
		c.modeButton.UV[state] = UV{0, float32(row) / 256, 64.0 / 512, float32(row+22) / 256}
	}
	return c, ld.err
}

func ChatOrigin(h int) int { return max(0, h-348) }

func (c *ChatPanel) paneGeometry(h, index int) (y, height, capacity int) {
	rows := c.Rows[index]
	rows = max(0, min(20, rows))
	lowerRows := c.Rows[1]
	lowerRows = max(0, min(20, lowerRows))
	height = (rows + 2 + 2*index) * 16
	y = h - 84 - (lowerRows+4)*16
	if index == 0 {
		y -= 8 + height
	}
	return max(0, y), height, max(1, height/16)
}

func (c *ChatPanel) SetEntries(f *Font, entries []ChatEntry) {
	if slices.Equal(c.entries, entries) {
		return
	}
	overlap := min(len(c.entries), len(entries))
	for overlap > 0 && !slices.Equal(c.entries[len(c.entries)-overlap:], entries[:overlap]) {
		overlap--
	}
	added := [2]int{}
	c.entries = append(c.entries[:0], entries...)
	for i := range c.panes {
		c.panes[i].lines = c.panes[i].lines[:0]
		clear(c.panes[i].runs)
		c.panes[i].runs = c.panes[i].runs[:0]
	}
	for entryIndex, entry := range entries {
		index := 1
		if entry.System {
			index = 0
		}
		pane := &c.panes[1]
		if entry.System {
			pane = &c.panes[0]
		}
		links := entry.Items.Len()
		runs, rows := wrapChatContent(f, entry.Text, entry.Color, entry.Markup, 316, entry.ItemType != 0, entry.Items[:links], c.linkedItemSuffix)
		if entryIndex >= overlap {
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

func (c *ChatPanel) controls(h int) {
	for i := range c.panes {
		p := &c.panes[i]
		y, height, capacity := c.paneGeometry(h, i)
		limit := max(0, len(p.lines)-capacity)
		for arrow := range p.arrows {
			b := &p.arrows[arrow]
			b.Tex, b.OriginX, b.OriginY = c.scroll, 4, y+4+arrow*(height-24)
			b.DrawW, b.DrawH, b.HitW, b.HitH = 16, 16, 16, 16
			b.Disabled = limit == 0
			for state, col := range []int{0, 1, 1, 1, 2} {
				b.UV[state] = UV{float32(col) / 8, float32(arrow) / 2, float32(col+1) / 8, float32(arrow)/2 + 16.0/64}
			}
		}
	}
}

// Contains keeps world picking behind the visible chat panels from handling
// the same pointer gesture. Geometry follows the current resized histories.
func (c *ChatPanel) Contains(mx, my, h int) bool {
	if mx < 0 || mx >= 356 {
		return false
	}
	if c.inputShown() && my >= chatInputY(h) && my < chatInputY(h)+22 {
		return true
	}
	if mx >= 350 {
		return false
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
	if c.resizing {
		c.Rows[c.resizePane] = min(20, max(0, c.resizeStart+(c.resizeY-in.MouseY)/16))
		if in.Clicked || !in.MouseDown {
			c.resizing = false
			c.Resized = true
		}
		return true
	}
	now := chatNow()
	if in.Pressed && in.MouseX >= 25 && in.MouseX < 350 {
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
	if c.inputShown() && c.modeButton.Update(in) {
		c.ModeOpen = !c.ModeOpen
	}
	if c.ModeOpen {
		row := -1
		if in.MouseX >= 0 && in.MouseX < 120 && in.MouseY >= inputY-len(chatChannelOrder)*22 && in.MouseY < inputY {
			row = (in.MouseY - (inputY - len(chatChannelOrder)*22)) / 22
		}
		if in.Pressed {
			c.modeHeld = row
		}
		if in.Clicked && row >= 0 && c.modeHeld == row {
			c.Channel = chatChannelOrder[row]
			c.ChannelChanged = true
			c.FocusRequested = true
			c.Recipient = ""
			c.ModeOpen = false
		}
		if in.Pressed && row < 0 && !c.modeButton.hit(in) {
			c.ModeOpen = false
		}
		return true
	}
	if c.inputShown() && in.MouseX >= 0 && in.MouseX < 356 && in.MouseY >= inputY && in.MouseY < inputY+22 {
		c.FocusRequested = in.Pressed && in.MouseX >= 64
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
		over := in.MouseX >= 0 && in.MouseX < 350 && in.MouseY >= y && in.MouseY < y+height
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
		for arrow := range p.arrows {
			if in.Pressed {
				p.arrows[arrow].held = false
			}
			if p.arrows[arrow].Update(in) {
				p.offset += 1 - 2*arrow
			}
		}
		if in.Pressed && in.MouseX >= 4 && in.MouseX < 17 && in.MouseY >= y+20 && in.MouseY < y+height-20 {
			p.dragging = true
		}
		if p.dragging && in.MouseDown {
			track := max(1, height-48)
			p.offset = limit - limit*max(0, min(track, in.MouseY-y-20))/track
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
		s.Draw(blank, 0, float32(y), 350, float32(height), 0, 0, 1, 1, [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 154.0 / 255 * a})
		restoreClip := s.ClipRect(25, float32(y+1), 316, float32(height-2))
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
		// Phoenix's bordered scrollbar rail is always visible, alpha 0.82.
		s.Draw(blank, 4, float32(y+4), 16, float32(height-8), 0, 0, 1, 1, [4]float32{.31, .27, .19, .8 * .82 * a})
		s.Draw(blank, 5, float32(y+5), 14, float32(height-10), 0, 0, 1, 1, [4]float32{.037, .035, .031, .7 * .82 * a})
		if a == 1 {
			for arrow := range p.arrows {
				p.arrows[arrow].Draw(s)
			}
		}
	}
	if !c.inputShown() {
		return
	}
	y := chatInputY(h)
	c.modeButton.OriginY = y
	c.modeButton.Draw(s)
	label := c.ChannelNames[min(int(c.Channel), 5)]
	// Centre the font cell in the 22px row; a fixed +6 left the larger chat
	// font touching the bottom border.
	textY := chatEntryTextY(y, f.Height)
	f.Draw(s, label, 7, textY, white, 44)
	// Keep the measured field width while omitting the language icon by request.
	s.Draw(c.input, 64, float32(y), 2, 22, 0, .5, 2.0/512, 150.0/256, white)
	s.Draw(c.input, 66, float32(y), 288, 22, 2.0/512, .5, 270.0/512, 150.0/256, white)
	s.Draw(c.input, 354, float32(y), 2, 22, 290.0/512, .5, 292.0/512, 150.0/256, white)
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
	for len(shown) > 0 && f.Width(shown) > available {
		shown = textwire.DropFirst(shown)
	}
	restoreClip := s.ClipRect(69, float32(y+2), 264, 19)
	f.Draw(s, shown, 69, textY, white, available)
	if typing && chatNow().UnixMilli()%1000 < 500 {
		s.Draw(blank, float32(69+f.Width(shown)), float32(textY), 1, float32(f.Height), 0, 0, 1, 1, white)
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
var chatChannelOrder = [6]uint8{0, 3, 2, 5, 4, 1}

func (c *ChatPanel) capacity(index int) int { _, _, n := c.paneGeometry(768, index); return n }

// DrawOverlay keeps the Phoenix channel picker above movable HUD bars.
func (c *ChatPanel) DrawOverlay(s *render.Sprites, f *Font, blank *render.Texture, h int) {
	if !c.ModeOpen {
		return
	}
	top := chatInputY(h) - len(chatChannelOrder)*22
	s.Draw(blank, 0, float32(top), 120, float32(len(chatChannelOrder)*22), 0, 0, 1, 1, [4]float32{.114, .122, .141, .97})
	for i, channel := range chatChannelOrder {
		tint := white
		if channel == c.Channel {
			tint = RGBColor(0xffff00)
		}
		f.Draw(s, c.ChannelNames[channel], 8, top+i*22+3, tint, 104)
	}
}

// Native entry row is six pixels below the old Phoenix approximation.
func chatInputY(h int) int { return h - 76 }

func ChatInputHit(x, y, height int) bool {
	return x >= 66 && x < 356 && y >= chatInputY(height) && y < chatInputY(height)+22
}

// Newest lines occupy the bottom row even in a partially filled history.
// Keep a full font cell and its shadow inside the flat frame; native frames
// extend with a four-pixel border that is absent in the chosen flat style.
func (c *ChatPanel) lineY(h, index, row, end, fontHeight int) int {
	y, height, _ := c.paneGeometry(h, index)
	return y + height - max(1, fontHeight) - 1 - (end-1-row)*16
}
