package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const (
	chatHelp = iota + 1
	chatFilters
)

var chatFilterBoxes = [8][2]int{{29, 55}, {129, 55}, {29, 76}, {129, 76}, {29, 141}, {129, 141}, {29, 162}, {129, 162}}
var chatHelpBoxes = [6][2]int{{29, 187}, {29, 208}, {29, 228}, {129, 187}, {129, 208}, {129, 228}}
var chatHelpChannels = [5]uint8{0, 5, 3, 2, 1}

func chatFilter(style uint8) uint8 {
	switch style {
	case 0, 1, 5:
		return 1 << 0
	case 3, 4, 6, 7, 17:
		return 1 << 1
	case 2:
		return 1 << 2
	case 8, 24, 25, 26, 27, 28, 29, 33:
		return 1 << 3
	case 9, 30:
		return 1 << 4
	case 10:
		return 1 << 5
	case 12:
		return 1 << 6
	case 13, 31:
		return 1 << 7
	}
	return 0
}

func (c *ChatPanel) OptionsOpen() bool { return c.option != 0 }
func (c *ChatPanel) PopupOpen() bool   { return c.OptionsOpen() || c.ModeOpen || c.recipientsOpen }
func (c *ChatPanel) optionBounds(h int) (x, y, w, ht int) {
	x = 380
	if c.ViewportWidth > 0 {
		x = min(x, max(0, c.ViewportWidth-206))
	}
	ht = 256
	if c.option == chatFilters {
		ht = 196
	}
	return x, max(0, h-296), 206, ht
}
func (c *ChatPanel) optionHit(x, y, h int) bool {
	if !c.OptionsOpen() {
		return false
	}
	ox, oy, w, ht := c.optionBounds(h)
	return x >= ox && x < ox+w && y >= oy && y < oy+ht
}

const (
	chatOptionWidth  = 22
	chatOptionHeight = 19
)

var chatOptionOrder = [...]int{chatFilters, chatHelp}

func (c *ChatPanel) optionButtonsShown() bool {
	return c.paneAlpha(1, chatNow()) > 0
}
func (c *ChatPanel) optionY(h, i int) int {
	y, height, _ := c.paneGeometry(h, 1)
	return y + height - (len(chatOptionOrder)-i)*chatOptionHeight
}
func (c *ChatPanel) optionButton(x, y, h int) int {
	if !c.optionButtonsShown() || x < 0 || x >= chatOptionWidth {
		return 0
	}
	for i := range 2 {
		top := c.optionY(h, i)
		if y >= top && y < top+chatOptionHeight {
			return chatOptionOrder[i]
		}
	}
	return 0
}

func (c *ChatPanel) optionChoice(x, y, h int) int {
	ox, oy, _, _ := c.optionBounds(h)
	boxes := chatHelpBoxes[:]
	if c.option == chatFilters {
		boxes = chatFilterBoxes[:]
	}
	for i, b := range boxes {
		if x >= ox+b[0] && x < ox+b[0]+16 && y >= oy+b[1] && y < oy+b[1]+16 {
			return i
		}
	}
	return -1
}
func (c *ChatPanel) updateOptions(in *platform.Input, h int) bool {
	if in.HasKey(platform.KeyEscape) && c.OptionsOpen() {
		c.option, c.optionPress = 0, -1
		return true
	}
	button := c.optionButton(in.MouseX, in.MouseY, h)
	if button != 0 {
		if in.Pressed {
			c.optionPress = 8 + button
		}
		if in.Clicked && c.optionPress == 8+button {
			if c.option == button {
				c.option = 0
			} else {
				c.option = button
			}
			c.ModeOpen = false
		}
		if in.Clicked {
			c.optionPress = -1
		}
		return in.Pressed || in.Clicked || in.MouseDown || in.Wheel != 0
	}
	if !c.OptionsOpen() {
		if in.Clicked {
			c.optionPress = -1
		}
		return false
	}
	if !c.optionHit(in.MouseX, in.MouseY, h) {
		if in.Pressed {
			c.option, c.optionPress = 0, -1
			return true
		}
		if in.Clicked && c.optionPress >= 0 {
			c.optionPress = -1
			return true
		}
		return false
	}
	i := c.optionChoice(in.MouseX, in.MouseY, h)
	if in.Pressed {
		c.optionPress = i
	}
	if in.Clicked {
		if i >= 0 && i == c.optionPress {
			switch {
			case c.option == chatFilters:
				c.Filters ^= 1 << i
				c.OptionsChanged = true
			case i == 5:
				c.Transparent = !c.Transparent
				c.OptionsChanged = true
			default:
				c.Channel, c.ChannelChanged, c.FocusRequested = chatHelpChannels[i], true, true
				c.RecipientFocus = c.Channel == 5
				c.option = 0
			}
		}
		c.optionPress = -1
	}
	return in.Pressed || in.Clicked || in.MouseDown || in.Wheel != 0 || in.RightPressed || in.RightDown
}

func (c *ChatPanel) drawOptionButtons(s *render.Sprites, f *Font, blank *render.Texture, h int) {
	for i := range 2 {
		y := float32(c.optionY(h, i))
		if c.icons != nil {
			u := float32(i*32) / 128
			// The lower atlas row contains the language icons. Both mouse
			// states use the authored menu icons from the upper row.
			s.Draw(c.icons, 0, y, chatOptionWidth, chatOptionHeight, u, 3.0/64, u+22.0/128, 22.0/64, white)
		} else {
			DrawTipFrame(s, blank, 0, y, chatOptionWidth, chatOptionHeight, 1, TipFill, TipBorder)
			f.Draw(s, []string{"F", "?"}[i], 4, int(y)+2, white, 12)
		}
	}
}

func (c *ChatPanel) drawOptions(s *render.Sprites, f *Font, blank *render.Texture, h int) {
	if !c.OptionsOpen() {
		return
	}
	x, y, w, ht := c.optionBounds(h)
	tex := c.help
	boxes := chatHelpBoxes[:]
	labels := []string{"Common", "Whisper", "Party", "Guild", "Trade", "Alpha"}
	if c.option == chatFilters {
		tex, boxes = c.filter, chatFilterBoxes[:]
		labels = []string{"Damage", "Acquire", "Death", "Notice", "Common", "Party", "Guild", "Trade"}
	}
	if tex != nil {
		s.Draw(tex, float32(x), float32(y), float32(w), float32(ht), 0, 0, float32(w)/256, float32(ht)/256, white)
	} else {
		DrawTipFrame(s, blank, float32(x), float32(y), float32(w), float32(ht), 1, TipFill, TipBorder)
		if c.option == chatFilters {
			f.Draw(s, "Message filters", x+16, y+25, white, 180)
			f.Draw(s, "Conversation filters", x+16, y+112, white, 180)
		} else {
			f.Draw(s, "Help – Chat", x+65, y+12, white, 140)
			for i, line := range []string{"! name message", "@ party message", "$ trade message"} {
				f.Draw(s, line, x+18, y+45+i*27, white, 180)
			}
			f.Draw(s, "Chat Mode", x+65, y+163, white, 140)
		}
	}
	for i, b := range boxes {
		checked := true
		switch {
		case c.option == chatFilters:
			checked = c.Filters&(1<<i) == 0
		case i == 5:
			checked = !c.Transparent
		default:
			checked = chatHelpChannels[i] == c.Channel
		}
		bx, by := x+b[0], y+b[1]
		if tex == nil {
			DrawTipFrame(s, blank, float32(bx), float32(by), 16, 16, 1, TipFill, TipBorder)
			f.Draw(s, labels[i], bx+20, by, white, 78)
		}
		if checked {
			if c.check != nil {
				s.Draw(c.check, float32(bx), float32(by), 16, 16, 0, 0, 1, 1, white)
			} else {
				f.Draw(s, "✓", bx+2, by, white, 14)
			}
		}
	}
}

const (
	chatScrollX     = 4
	chatThumbHeight = 13
)

// Leave room for both auxiliary buttons inside the lower history.
func (c *ChatPanel) scrollSpan(i, h int) int {
	if i == 1 {
		h -= len(chatOptionOrder) * chatOptionHeight
	}
	return h - 8
}

func (c *ChatPanel) scrollGeometry(i, y, h, limit, offset int) (top, travel, thumb int) {
	top = y + 4 + scrollbarArrowHitH
	travel = max(0, c.scrollSpan(i, h)-2*scrollbarArrowHitH-chatThumbHeight)
	thumb = top + travel
	if limit > 0 {
		thumb -= (travel*offset + limit/2) / limit
	}
	return
}
func (c *ChatPanel) drawScroll(s *render.Sprites, i, y, h, limit int, alpha float32) {
	p := &c.panes[i]
	_, travel, thumb := c.scrollGeometry(i, y, h, limit, p.offset)
	tint := [4]float32{1, 1, 1, alpha}
	for j := range p.arrows {
		b := &p.arrows[j]
		// At the minimum row settings the fixed native arrow hit boxes
		// cannot both fit. Keep wheel scrolling available in that case.
		if b.Tex != nil && c.scrollSpan(i, h) >= 2*scrollbarArrowHitH {
			uv := b.UV[b.state]
			if b.Disabled {
				uv = b.UV[StateNormal]
			}
			s.Draw(b.Tex, float32(b.X()), float32(b.Y()), float32(b.DrawW), float32(b.DrawH), uv.U0, uv.V0, uv.U1, uv.V1, tint)
		}
	}
	if c.scroll != nil && travel > 0 {
		// Extend the flat body above the atlas bevel. There is one
		// bottom edge, so it reads as a single handle, even at rest.
		const body = chatThumbHeight - scrollbarThumbH
		const u = 48.0 / 128
		s.Draw(c.scroll, chatScrollX, float32(thumb), scrollbarHitW, body, u, 0, u+scrollbarHitW/128.0, 1.0/64, tint)
		drawScrollbarThumb(s, c.scroll, chatScrollX, thumb+body, tint)
	}
}

const chatRecipientX = 25

func (c *ChatPanel) recipientHit(x, y, h int) bool {
	return c.Typing && c.Channel == 5 && x >= chatRecipientX && x < chatRecipientX+195 && y >= chatInputY(h)-22 && y < chatInputY(h)
}
func (c *ChatPanel) drawRecipient(s *render.Sprites, f *Font, blank *render.Texture, h int) {
	if !c.Typing || c.Channel != 5 {
		return
	}
	y := chatInputY(h) - 22
	s.Draw(c.input, chatRecipientX, float32(y), 194, 22, 0, .5, 194.0/512, 150.0/256, white)
	f.Draw(s, c.ChannelNames[5], chatRecipientX+7, chatEntryTextY(y, f.Height), white, 54)
	textY := chatEntryTextY(y, f.Height)
	runes := []rune(c.Recipient)
	if c.RecipientFocus {
		drawChatSelection(s, f, blank, runes, 0, c.RecipientCaret, chatRecipientX+66, textY, 105)
	}
	f.Draw(s, c.Recipient, chatRecipientX+66, textY, white, 105)
	if len(c.Recent) > 0 {
		f.Draw(s, "▾", chatRecipientX+179, textY, white, 14)
	}
	if c.RecipientFocus && chatNow().UnixMilli()%1000 < 500 {
		cursor := min(len(runes), max(0, c.RecipientCaret.Position))
		s.Draw(blank, float32(chatRecipientX+66+min(105, f.Width(string(runes[:cursor])))), float32(textY), 1, float32(f.Height), 0, 0, 1, 1, white)
	}
}

func (c *ChatPanel) backgroundHovered(h int) bool {
	upper, _, _ := c.paneGeometry(h, 0)
	lower, height, _ := c.paneGeometry(h, 1)
	return c.mouseX >= 0 && c.mouseX < 350 && c.mouseY >= upper && c.mouseY < lower+height+20
}

func (c *ChatPanel) recentGeometry(h int) (x, y, count int) {
	count = min(10, len(c.Recent))
	return chatRecipientX + 64, chatInputY(h) - 22 - count*15, count
}

func (c *ChatPanel) recentHit(mx, my, h int) int {
	x, y, count := c.recentGeometry(h)
	if mx < x || mx >= x+131 || my < y || my >= y+count*15 {
		return -1
	}
	return (my - y) / 15
}

func (c *ChatPanel) updateRecipient(in *platform.Input, h int) bool {
	if !c.Typing || c.Channel != 5 {
		c.recipientsOpen = false
		return false
	}
	if c.recipientsOpen && in.HasKey(platform.KeyEscape) {
		c.recipientsOpen = false
		return true
	}
	if c.recipientsOpen {
		i := c.recentHit(in.MouseX, in.MouseY, h)
		if i >= 0 {
			if in.Pressed {
				c.recipientHeld = i
			}
			if in.Clicked && c.recipientHeld == i {
				c.Recipient, c.RecipientFocus = c.Recent[i], false
				c.recipientsOpen = false
			}
			return true
		}
		if in.Pressed {
			c.recipientsOpen = false
		}
	}
	if !c.recipientHit(in.MouseX, in.MouseY, h) {
		return false
	}
	if in.MouseX >= chatRecipientX+176 && len(c.Recent) > 0 {
		if in.Pressed {
			c.recipientHeld = -2
		}
		if in.Clicked && c.recipientHeld == -2 {
			c.recipientsOpen = !c.recipientsOpen
			c.recipientHeld = -1
		}
	} else {
		c.RecipientRequested = in.Pressed
	}
	return true
}
func (c *ChatPanel) drawRecentRecipients(s *render.Sprites, f *Font, blank *render.Texture, h int) {
	if !c.recipientsOpen || !c.Typing || c.Channel != 5 {
		return
	}
	x, y, count := c.recentGeometry(h)
	if count == 0 {
		return
	}
	DrawTipFrame(s, blank, float32(x), float32(y), 131, float32(count*15), 1, TipFill, TipBorder)
	active := c.recentHit(c.mouseX, c.mouseY, h)
	for i, name := range c.Recent[:count] {
		color := white
		if i == active {
			color = RGBColor(0xffff00)
		}
		f.Draw(s, name, x+5, y+i*15, color, 122)
	}
}
