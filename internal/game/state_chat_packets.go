package game

import (
	"fmt"

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
	case proto.ChatNormal, proto.ChatParty, proto.ChatShout, proto.ChatNormalAdmin:
		c, err := proto.DecodeChatByID(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.addChat(ChatLine{Op: op, SenderID: c.SenderID, Text: c.Text, Items: c.Items, Raid: op == proto.ChatParty && s.Party.Raid})
		ev.Text = fmt.Sprintf("chat %d: %s", c.SenderID, c.Text)
	case proto.ChatWorld, proto.ChatGuild, proto.ChatMap, proto.ChatWorldShout, proto.ChatWorldAdmin:
		c, err := proto.DecodeChatByName(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.addChat(ChatLine{Op: op, Sender: c.Sender, Text: c.Text, Items: c.Items})
		ev.Text = fmt.Sprintf("chat %s: %s", c.Sender, c.Text)
	case proto.ChatWhisper:
		c, err := proto.DecodeWhisper(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.addChat(ChatLine{Op: op, Sender: c.Sender, Text: c.Text, Items: c.Items, Direction: c.Direction})
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
