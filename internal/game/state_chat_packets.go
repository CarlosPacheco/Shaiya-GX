package game

import (
	"fmt"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func (s *State) applyChatPacket(p []byte, ev Event) (Event, error) {
	op := ev.Op
	switch ev.Op {
	case proto.MobSpeechText, proto.MobSpeechIndex:
		c, err := proto.DecodeMobSpeech(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.LastMobSpeech = c
		if op == proto.MobSpeechText {
			s.addChat(ChatLine{Op: op, SenderID: c.MobID, Text: c.Text, Style: 1})
		}
		ev.Text = fmt.Sprintf("mob speech %d: %d", c.MobID, c.MessageID)
	case proto.ChatRaid:
		c, err := proto.DecodeChatByID(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.RaidNotice, s.RaidNoticeSerial = c.Text, s.RaidNoticeSerial+1
		ev.ChatQuiet = true
	case proto.ChatNormal, proto.ChatParty, proto.ChatShout, proto.ChatNormalAdmin:
		c, err := proto.DecodeChatByID(p)
		if err != nil {
			return ev.packetError(err)
		}
		ev.ChatQuiet = s.receivePlayerChat(ChatLine{Op: op, SenderID: c.SenderID, Text: c.Text, Items: c.Items, Raid: op == proto.ChatParty && s.Party.Raid}, time.Now())
		if !ev.ChatQuiet {
			ev.Text = fmt.Sprintf("chat %d: %s", c.SenderID, s.Chat[len(s.Chat)-1].Text)
		}
	case proto.ChatWorld, proto.ChatGuild, proto.ChatMap, proto.ChatWorldShout, proto.ChatWorldAdmin:
		c, err := proto.DecodeChatByName(p)
		if err != nil {
			return ev.packetError(err)
		}
		ev.ChatQuiet = s.receivePlayerChat(ChatLine{Op: op, Sender: c.Sender, Text: c.Text, Items: c.Items}, time.Now())
		if !ev.ChatQuiet {
			line := s.Chat[len(s.Chat)-1]
			ev.Text = fmt.Sprintf("chat %s: %s", line.Sender, line.Text)
		}
	case proto.ChatWhisper:
		c, err := proto.DecodeWhisper(p)
		if err != nil {
			return ev.packetError(err)
		}
		ev.ChatQuiet = s.receivePlayerChat(ChatLine{Op: op, Sender: c.Sender, Text: c.Text, Items: c.Items, Direction: c.Direction}, time.Now())
	case proto.ChatError:
		r := proto.NewReader(p)
		code := r.U8()
		if err := r.Err(); err != nil {
			return ev.packetError(err)
		}
		message := fmt.Sprintf("Chat error %d.", code)
		if code == 1 {
			message = "Whisper recipient not found or unavailable."
		}
		id := 0
		if code == 1 {
			id = 347
		}
		s.addChat(ChatLine{Op: op, Text: message, MessageID: id, Style: 1})
	}
	return ev, nil
}
