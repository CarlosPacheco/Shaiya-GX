package world

import (
	"encoding/binary"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"

	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// Chat follows CUser::PacketChatE 0x00470490. Every channel takes a length
// byte (the text including its terminator): 0 or 1 is ignored, above 128
// the connection is closed (0x004d43a0 with reason 9).
const (
	chatNormalRadius = 30  // PSendViewChat
	chatShoutRadius  = 100 // PSendViewRange
	chatShoutDelay   = 30 * time.Second
	chatMaxLength    = 0x80
)

// chatLength validates the length byte at p[at]: ok false drops the
// message, and closes the connection when it is too long.
func (s *Session) chatLength(p []byte, at int) bool {
	if len(p) <= at {
		return false
	}
	n := int(p[at])
	if n > chatMaxLength {
		s.log.Printf("GX Server World chat length %d over %d, closing (original reason 9)", n, chatMaxLength)
		if s.conn != nil {
			s.conn.Close()
		}
		return false
	}
	return n > 1 && len(p) >= at+1+n
}

func (s *Session) handleChat(op packet.Opcode, p []byte) {
	if s.char == nil || !s.chatLength(p, 2) {
		return
	}
	msg, err := readChatMessage(p)
	if err != nil {
		return
	}
	if s.handleGMCommand(msg) {
		return
	}
	if isTownCommand(msg) {
		s.executeTown()
		return
	}
	items, ok := s.chatItemSnapshots(p, 2)
	if ok {
		s.handlePublicChat(op, msg, items)
	}
}

// handlePublicChat delivers the normal, trade, shout and area channels.
func (s *Session) handlePublicChat(op packet.Opcode, message string, items proto.ChatItems) {
	if (op == packet.ChatNormalAdmin || op == packet.ChatWorldAdmin) && !s.isAdmin {
		s.log.Printf("GX Server World rejected admin chat from non-admin account_id=%d", s.accountID)
		return
	}
	ch := s.char
	var frame []byte
	var to []*ConnectedPlayer
	switch op {
	case packet.ChatNormal, packet.ChatNormalAdmin:
		// 0x1101: char id + text within 30 units.
		frame, to = buildChatByID(packet.ChatNormal, ch.ID, message), PlayersNear(ch.MapID, ch.PosX, ch.PosZ, chatNormalRadius)
	case packet.ChatWorld, packet.ChatWorldAdmin:
		// 0x1103 trade: name + text to the whole country (PSendAllCountry).
		frame, to = buildChatByName(packet.ChatWorld, ch.Name, message), AllPlayers()
	case packet.ChatShout:
		// 0x1107: char id + text within 100 units, once every 30 s.
		if time.Now().Before(s.shoutReadyAt) {
			return
		}
		s.shoutReadyAt = time.Now().Add(chatShoutDelay)
		frame, to = buildChatByID(packet.ChatShout, ch.ID, message), PlayersNear(ch.MapID, ch.PosX, ch.PosZ, chatShoutRadius)
	case packet.ChatWorldShout:
		// 0x1108 needs the one-shot permission an item grants (CUser
		// +0x5840); no item grants it yet.
		return
	default:
		// 0x1111 area: name + text to the zone.
		frame, to = buildChatByName(packet.ChatMap, ch.Name, message), PlayersOnMap(ch.MapID)
	}
	recipients := make([]*ConnectedPlayer, 0, len(to))
	for _, p := range to {
		if p.IsAdmin || p.Faction() == s.faction {
			recipients = append(recipients, p)
		}
	}
	broadcast(recipients, chatWithItems(frame, items), "chat", s.logger)
}

// handleWhisper (0x1102): target name[21] (20 characters), length, text.
// The target (by name, same country) gets direction 0 with the sender's
// name; the sender gets direction 1 with the target's name; otherwise
// 0x1106 result 1.
func (s *Session) handleWhisper(p []byte) {
	if s.char == nil || len(p) < 0x18 || !s.chatLength(p, 0x17) {
		return
	}
	target := fixedText(p, 2, 20)
	msg, err := readChatMessage(append([]byte{0, 0}, p[0x17:]...))
	if err != nil {
		return
	}
	t := FindPlayerByName(target)
	if t == nil || t.Faction() != s.faction || db.SocialBlocked(t.CharacterID, s.char.ID) {
		s.send(buildChatError(1), "0x1106 whisper target not found")
		return
	}
	items, ok := s.chatItemSnapshots(p, 0x17)
	if !ok {
		return
	}
	t.Send(chatWithItems(buildWhisper(0, s.char.Name, msg), items), "0x1102 whisper in")
	s.send(chatWithItems(buildWhisper(1, t.CharacterName, msg), items), "0x1102 whisper out")
}

func buildWhisper(direction uint8, name, message string) []byte {
	w := packet.New(packet.ChatWhisper)
	w.U8(direction)
	w.FixedString(name, 21)
	writeChatText(w, message)
	return w.MustFinish()
}

// Build display data from server-owned inventory, never from chat text.
func (s *Session) chatItemSnapshots(p []byte, textOffset int) (items proto.ChatItems, ok bool) {
	slots, ok := proto.ChatItemSlots(p, textOffset)
	if !ok {
		return items, false
	}
	for n, slot := range slots {
		if !isValidItemPosition(slot[0], slot[1]) {
			return proto.ChatItems{}, false
		}
		i := findItem(s.items, slot[0], slot[1])
		if i < 0 || s.items[i].Count == 0 {
			return proto.ChatItems{}, false
		}
		it := s.items[i]
		items[n] = proto.InventoryItem{Type: it.Type, TypeID: it.TypeID, Quality: it.Quality, Count: it.Count, Gems: it.Gems, Craft: it.CraftName}
	}
	return items, true
}

func chatWithItems(frame []byte, items proto.ChatItems) []byte {
	if items[0].Type == 0 {
		return frame
	}
	frame = proto.WithChatItems(frame, items)
	binary.LittleEndian.PutUint16(frame, uint16(len(frame)))
	return frame
}
