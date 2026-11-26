package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/gm"
	"slices"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// updateChat captures typing before world controls. Channel syntax is a
// delivery adaptation; encoding and opcodes use the verified 0x599a60 writer.
func (a *App) updateChat(in *platform.Input) bool {
	a.syncRaidChannel()
	if a.chatUI != nil && a.chatUI.RaidAllowed && !a.chatOpen && hasChar(in, '\'') {
		// The apostrophe key writes one raid notice. The channel is not kept;
		// picking Raid in the mode list keeps it.
		a.openChatComposer()
		a.raidOnce, a.raidReturn = true, a.chatUI.Channel
		a.chatUI.Channel = proto.ChannelRaid
		a.chatUI.RecipientFocus = false
		return true
	}
	if a.chatUI != nil {
		for _, shortcut := range [...]struct {
			key     platform.Key
			channel uint8
		}{
			{platform.KeyChatWhisper, 5}, {platform.KeyChatParty, proto.ChannelParty},
			{platform.KeyChatGuild, proto.ChannelGuild}, {platform.KeyChatTrade, proto.ChannelTrade},
			{platform.KeyChatArea, proto.ChannelShout},
		} {
			if in.HasKey(shortcut.key) {
				if !a.chatOpen {
					a.openChatComposer()
				}
				a.chatUI.Channel = shortcut.channel
				a.chatUI.RecipientFocus = shortcut.channel == 5 && a.chatUI.Recipient == ""
				return true
			}
		}
	}
	if !a.chatOpen {
		if !in.HasKey(platform.KeyEnter) {
			return false
		}
		a.openChatComposer()
		if a.chatUI != nil {
			a.chatUI.RecipientFocus = a.chatUI.Channel == 5 && a.chatUI.Recipient == ""
		}
		return true
	}
	if in.HasKey(platform.KeyEscape) {
		a.chatItems = nil
		a.chatOpen = false
		a.endRaidOnce()
		return true
	}
	if a.chatUI != nil && a.chatUI.Channel == 5 {
		if in.HasKey(platform.KeyTab) {
			a.cycleWhisperRecipient()
			return true
		}
		if a.chatUI.RecipientFocus {
			recipient := ui.TextField{Text: a.chatUI.Recipient, Max: 20}
			recipient.EditAt(in, &a.recipientCaret)
			a.chatUI.Recipient = strings.TrimSpace(recipient.Text)
			if in.HasKey(platform.KeyEnter) {
				a.chatUI.RecipientFocus = false
			}
			return true
		}
	}
	if len(a.chatItems) > 0 && a.chatInput.Text == "" && in.HasKey(platform.KeyBackspace) {
		a.chatItems = a.chatItems[:len(a.chatItems)-1]
	}
	a.chatInput.EditAt(in, &a.chatCaret)
	if in.HasKey(platform.KeyUp) || in.HasKey(platform.KeyDown) {
		if a.chatHistoryIndex == len(a.chatHistory) {
			a.chatDraft = a.chatInput.Text
		}
		direction := -1
		if in.HasKey(platform.KeyDown) {
			direction = 1
		}
		a.chatHistoryIndex = max(0, min(len(a.chatHistory), a.chatHistoryIndex+direction))
		a.chatInput.Text = a.chatDraft
		if a.chatHistoryIndex < len(a.chatHistory) {
			a.chatInput.Text = a.chatHistory[a.chatHistoryIndex]
		}
		a.chatCaret.End(a.chatInput.Text)
	}
	if in.HasKey(platform.KeyEnter) {
		text := strings.TrimSpace(a.chatInput.Text)
		var slots [][2]uint8
		for _, draft := range a.chatItems {
			if it, ok := a.state.Inventory[draft.slot]; ok && it == draft.item {
				slots = append(slots, draft.slot)
			} else {
				a.chatItems = nil
				a.questMessageID(12023)
				return true
			}
		}
		defer func() {
			if !a.chatOpen {
				a.chatItems = nil
				a.endRaidOnce()
			}
		}()
		if text != "" {
			a.chatHistory = append(a.chatHistory, text)
			if len(a.chatHistory) > 32 {
				a.chatHistory = a.chatHistory[len(a.chatHistory)-32:]
			}
		}
		if a.collisionViewCommand(text) || a.cameraLimitCommand(text) {
			a.chatOpen = false
			return true
		}
		if _, _, known := gm.Parse(text); known {
			if rewritten, ok := a.gmTargetText(text); !ok {
				a.chatOpen = false
				return true
			} else {
				text = rewritten
			}
			if a.world == nil {
				a.questMessageID(12024)
			} else if err := a.world.Send(proto.ChatPacket(proto.ChannelNormal, false, text)); err != nil {
				a.backToLoginID(12020)
			}
			a.chatOpen = false
			return true
		}
		selectedChannel, channelText, channelCommand := phoenixChannelCommand(text)
		if channelCommand {
			text = channelText
		}
		if !channelCommand && (a.townCommand(text) || a.gestureCommand(text) || a.vehicleCommand(text) || a.partyCommand(text) || a.mallCommand(text) || a.questCommand(text) || a.exchangeCommand(text) || a.duelCommand(text) || a.guildCommand(text) || a.guildDuelCommand(text)) {
			a.chatOpen = false
			return true
		}
		if packet, handled := whisperCommand(text, len(slots) > 0); handled && !channelCommand {
			if packet == nil {
				a.questMessageID(12025)
			} else if err := a.world.Send(proto.WithChatItemSlots(packet, slots)); err != nil {
				a.backToLoginID(12021)
			}
			a.chatOpen = false
			return true
		}
		if text != "" || len(slots) > 0 {
			channel := proto.ChannelNormal
			if a.chatUI != nil {
				channel = a.chatUI.Channel
			}
			if channelCommand {
				channel = selectedChannel
			}
			if !channelCommand && text != "" {
				switch text[0] {
				case '@':
					channel = proto.ChannelParty
					text = strings.TrimSpace(text[1:])
				case '#':
					channel = proto.ChannelGuild
					text = strings.TrimSpace(text[1:])
				case '$':
					channel = proto.ChannelTrade
					text = strings.TrimSpace(text[1:])
				}
			}
			if text == "" && len(slots) == 0 {
				a.chatOpen = false
				return true
			}
			packet := proto.ChatPacket(channel, false, chatBody(text, len(slots) > 0))
			if channel == 5 {
				recipient := strings.TrimSpace(a.chatUI.Recipient)
				if recipient == "" {
					recipient, text, _ = strings.Cut(text, " ")
				}
				packet = proto.WhisperPacket(recipient, chatBody(text, len(slots) > 0))
				if packet == nil {
					a.localizedNotice(1, 232, nil)
					return true
				}
			}
			if err := a.world.Send(proto.WithChatItemSlots(packet, slots)); err != nil {
				a.backToLoginID(12022)
			}
		}
		a.chatOpen = false
	}
	return true
}

func (a *App) openChatComposer() {
	a.chatOpen = true
	a.chatItems = nil
	a.chatInput = ui.TextField{Max: 120}
	a.chatCaret, a.chatRecipientIndex = ui.ChatCaret{}, -1
	if a.chatUI != nil {
		a.recipientCaret.End(a.chatUI.Recipient)
	}
	a.chatHistoryIndex, a.chatDraft = len(a.chatHistory), ""
}

func (a *App) recentWhisperRecipients() []string {
	if a.state == nil {
		return nil
	}
	var names []string
	for i := len(a.state.Chat) - 1; i >= 0; i-- {
		line := a.state.Chat[i]
		if line.Op == proto.ChatWhisper && line.Sender != "" && !line.ChatProtected &&
			!slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, line.Sender) }) {
			names = append(names, line.Sender)
		}
	}
	return names
}

func (a *App) cycleWhisperRecipient() {
	names := a.recentWhisperRecipients()
	if len(names) > 0 {
		a.chatRecipientIndex = (a.chatRecipientIndex + 1) % len(names)
		a.chatUI.Recipient = names[a.chatRecipientIndex]
		a.recipientCaret.End(a.chatUI.Recipient)
		a.chatUI.RecipientFocus = false
	}
}

func chatBody(text string, linked bool) string {
	text = strings.TrimSpace(text)
	if text == "" && linked {
		return " "
	}
	return text
}

func whisperCommand(text string, linked ...bool) ([]byte, bool) {
	hasItems := len(linked) > 0 && linked[0]
	if strings.HasPrefix(text, "!") {
		name, message, _ := strings.Cut(strings.TrimSpace(text[1:]), " ")
		return proto.WhisperPacket(name, chatBody(message, hasItems)), true
	}
	command, rest, _ := strings.Cut(strings.TrimSpace(text), " ")
	if !strings.EqualFold(command, "/w") && !strings.EqualFold(command, "/whisper") && !strings.EqualFold(command, "/t") && !strings.EqualFold(command, "/tell") {
		return nil, false
	}
	name, message, _ := strings.Cut(strings.TrimSpace(rest), " ")
	return proto.WhisperPacket(name, chatBody(message, hasItems)), true
}

func (a *App) drawChat(w, h int) {
	f := a.chatFont()
	a.chatUI.ViewportWidth = w
	a.chatUI.Caret, a.chatUI.RecipientCaret = a.chatCaret, a.recipientCaret

	for i, id := range ui.ChatChannelIDs {
		if label := a.tr(id); label != "" {
			a.chatUI.ChannelNames[i] = label
		}
	}
	a.chatUI.DrawItem = a.drawChatItem
	a.chatUI.LinkedItemCount = func(it proto.InventoryItem) uint16 {
		return a.itemCount(it.Type, it.TypeID, it.Count)
	}
	a.chatUI.DrawLinkedItem = func(it proto.InventoryItem, x, y, size int, alpha float32) {
		if def := a.itemsItem(it.Type, it.TypeID); def != nil {
			a.drawItemIcon(def, x, y, size, [4]float32{1, 1, 1, alpha}, &it)
			if a.lastMouseX >= x && a.lastMouseX < x+size && a.lastMouseY >= y && a.lastMouseY < y+size {
				a.drawItemDetails(def, &it, it.Count, x, y, w, h, "")
			}
		}
	}
	a.chatUI.DraftItems = proto.ChatItems{}
	if a.chatOpen {
		for i, draft := range a.chatItems {
			a.chatUI.DraftItems[i] = draft.item
		}
	}
	a.chatUI.SetEntries(f, a.formattedChat())
	shown := ""
	if a.chatOpen {
		shown = a.chatInput.Text
	}
	a.chatUI.Draw(a.sprites, f, a.white, h, a.chatOpen, shown)
}

func (a *App) formattedChat() []ui.ChatEntry {
	filter := a.chatFilter != nil && a.clientSettings().BannedWords
	if filter == a.chatFilterApplied && slices.Equal(a.chatCache, a.state.Chat) {
		return a.chatEntries
	}
	a.chatFilterApplied = filter
	a.chatUI.Recent = a.recentWhisperRecipients()
	clear(a.chatCache)
	a.chatCache = append(a.chatCache[:0], a.state.Chat...)
	clear(a.chatEntries)
	a.chatEntries = a.chatEntries[:0]
	for _, line := range a.state.Chat {
		if filter && line.Op != 0 && line.Op != proto.ChatError && line.Op != proto.MobSpeechText && line.Direction != 1 {
			line.Text = a.chatFilter.Censor(line.Text)
		}
		a.chatEntries = append(a.chatEntries, chatEntry(a.msg, line))
	}
	return a.chatEntries
}

func chatStyle(c game.ChatLine) (style uint8, messageID int) {
	switch c.Op {
	case 0:
		return c.Style, 0
	case proto.ChatError:
		return c.Style, c.MessageID
	case proto.ChatParty:
		return 10, 353
	case proto.ChatWhisper:
		return 11, 0
	case proto.ChatGuild:
		return 12, 370
	case proto.ChatWorld:
		return 13, 355
	case proto.ChatShout, proto.ChatWorldShout:
		return 14, 0
	case proto.ChatMap:
		return 31, 482
	case proto.ChatNormalAdmin:
		return 16, 0
	case proto.ChatWorldAdmin:
		return 15, 356
	default:
		return 9, 0
	}
}

func chatPresentation(messages text.SysMsg, c game.ChatLine, id int) (name, template string) {
	name = c.Sender
	if name == "" && c.SenderID != 0 {
		name = messages.Format(12296, map[string]string{"v": fmt.Sprint(c.SenderID)})
	}
	if id != 0 && messages.Get(id) != "" {
		template = messages.Lookup(id)
	}
	return name, template
}

func chatEntry(messages text.SysMsg, c game.ChatLine) ui.ChatEntry {
	style, id := chatStyle(c)
	body := c.Text
	if c.Op != 0 {
		name, template := chatPresentation(messages, c, id)
		switch {
		case template != "" && c.GMReply != nil:
			body = text.FormatMessage(template, c.GMReply.Values)
		case template != "":
			body = text.FormatMessage(template, map[string]string{"p": name, "b": c.Text})
		case c.Op == proto.ChatWhisper:
			direction := "from"
			if c.Direction == 1 {
				direction = "to"
			}
			body = "Whisper " + direction + " " + name + ": " + c.Text
		case c.Op != proto.ChatError:
			body = name + " : " + c.Text
		}
	}
	rgb := ui.ChatRGB(style)
	return ui.ChatEntry{Items: c.Items, ItemType: c.ItemType, ItemID: c.ItemID, Text: body, Repetitions: c.Repetitions, System: c.SystemPane(), Color: ui.RGBColor(rgb), Style: style, Markup: c.Op == 0}
}

// chatWorldInput keeps pointer routing and automatic travel active while the
// composer owns keyboard input.
func chatWorldInput(in platform.Input) platform.Input {
	in.Chars, in.Keys, in.Actions = nil, nil, nil
	in.Down, in.Released, in.KeypadChars = "", "", ""
	in.Space, in.Shift, in.ActionHeld = false, false, 0
	return in
}

func (a *App) updateChatPointer(in *platform.Input, w, h int) bool {
	if a.chatUI == nil {
		return false
	}
	a.chatUI.Typing = a.chatOpen
	a.chatUI.ViewportWidth = w
	captured := a.chatUI.Update(in, h)
	if a.chatUI.FocusRequested && !a.chatOpen {
		a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
	}
	if a.chatUI.RecipientRequested {
		a.chatUI.RecipientFocus = true
		a.recipientCaret.End(a.chatUI.Recipient)
	}
	if (a.chatUI.Resized || a.chatUI.OptionsChanged) && a.gameMenu != nil {
		a.gameMenu.SetChatRows(a.chatUI.Rows)
		a.gameMenu.SetChatOptions(a.chatUI.Filters, a.chatUI.Transparent)
		a.saveClientSettings()
	}
	return captured
}

func (a *App) chatFont() *ui.Font {
	if a.chatTextFont != nil {
		return a.chatTextFont
	}
	if a.tooltipFont != nil {
		return a.tooltipFont
	}
	return a.font
}

// Phoenix aliases take precedence over delivery-only diagnostic commands.
func phoenixChannelCommand(value string) (uint8, string, bool) {
	command, body, _ := strings.Cut(value, " ")
	channels := map[string]uint8{"/s": 0, "/say": 0, "/p": 3, "/party": 3, "/g": 2, "/guild": 2, "/1": 4, "/area": 4, "/shout": 4, "/2": 1, "/trade": 1}
	if strings.EqualFold(command, "/party") && isPartyInviteName(body) {
		return 0, "", false // /party <name> invites; /party with text is still party chat
	}
	channel, ok := channels[strings.ToLower(command)]
	return channel, strings.TrimSpace(body), ok
}

// isPartyInviteName reports whether the text after /party is a lone character
// name. Names have no spaces, so several words remain party chat.
func isPartyInviteName(body string) bool {
	fields := strings.Fields(body)
	return len(fields) == 1
}

func (a *App) drawChatItem(typ, id uint8, x, y, size int, alpha float32) {
	if a.assets == nil {
		return
	}
	if d := a.itemsItem(typ, id); d != nil {
		a.drawItemIcon(d, x, y, size, [4]float32{1, 1, 1, alpha})
	}
}

type chatItemDraft struct {
	slot [2]uint8
	item proto.InventoryItem
}

func (a *App) updateChatItemDrag(in *platform.Input, w, h int) bool {
	v := &a.inventory
	if a.chatUI == nil || a.inventoryUI == nil || v.pending || a.shop.pending || a.state.Self.Dead {
		return false
	}
	v.advanceItemDrag(in)
	if v.open && in.Clicked && v.dragging && v.selected && a.chatUI.Contains(in.MouseX, in.MouseY, h) {
		v.cancelItemPointer()
		it, ok := a.state.Inventory[v.source]
		if !ok {
			return true
		}
		if !a.chatOpen {
			a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
		}
		for _, draft := range a.chatItems {
			if draft.slot == v.source {
				return true
			}
		}
		if len(a.chatItems) < proto.MaxChatItems {
			a.chatItems = append(a.chatItems, chatItemDraft{v.source, it})
		}
		return true
	}
	if !a.chatOpen {
		return false
	}
	if v.open {
		if v.bag < 1 || v.bag > 5 {
			v.bag = 1
		}
		var closed bool
		v.bag, closed = a.inventoryUI.Update(in, w, h, v.bag)
		if closed {
			v.open = false
			v.cancelItemPointer()
			return true
		}
		if in.Pressed && !in.Clicked {
			x, y := a.inventoryUI.Origin(w, h)
			if key, hit := ui.InventoryHit(in.MouseX-x, in.MouseY-y, v.bag); hit {
				if it, ok := a.state.Inventory[key]; ok {
					a.playItemSound(it.Type)
					v.pressItem(key, in)
					return true
				}
			}
		}
		if (v.pressedItem || v.dragging) && (in.Clicked || !in.MouseDown) {
			v.cancelItemPointer()
			return true
		}
	}
	if a.bottom != nil && a.bottom.UpdateInventoryButton(in, w, h, v.open) {
		a.toggleInventory()
		return true
	}
	return false
}

// raidChatAllowed: only a raid leader or sub-leader has the Raid channel.
func (a *App) raidChatAllowed() bool {
	return a.state != nil && a.state.Party.Raid && a.unionRole() <= 1
}

// syncRaidChannel shows or hides the Raid entry as the role changes. Losing the
// role while Raid is selected drops back to the normal channel.
func (a *App) syncRaidChannel() {
	if a.chatUI == nil {
		return
	}
	a.chatUI.RaidAllowed = a.raidChatAllowed()
	if !a.chatUI.RaidAllowed && a.chatUI.Channel == proto.ChannelRaid {
		a.chatUI.Channel = a.raidReturn
		if a.raidReturn == proto.ChannelRaid {
			a.chatUI.Channel = proto.ChannelNormal
		}
		a.raidOnce = false
	}
}

// endRaidOnce restores the channel after the one-shot raid composer closes.
func (a *App) endRaidOnce() {
	if a.raidOnce && a.chatUI != nil && a.chatUI.Channel == proto.ChannelRaid {
		a.chatUI.Channel = a.raidReturn
	}
	a.raidOnce = false
}

// showRaidNotice prints a received 0x1112 notice: SysMsg 6606 "[Union] <b>" on
// screen in violet (style 0x20, 0x567980).
func (a *App) showRaidNotice(body string) {
	if a.chatFilter != nil && a.clientSettings().BannedWords {
		body = a.chatFilter.Censor(body)
	}
	if a.msg.Get(6606) != "" {
		a.systemMessage(32, 6606, map[string]string{"b": body})
	} else {
		a.addSystemText(32, 0, "[Union] "+body)
	}
}
