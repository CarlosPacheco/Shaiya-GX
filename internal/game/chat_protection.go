package game

import (
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/chatguard"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

const (
	chatSenderLimit  = 512
	chatRepeatWindow = 15 * time.Second
	chatRateWindow   = 10 * time.Second
)

type chatAudience uint8

const (
	chatPublic chatAudience = iota
	chatParty
	chatGuild
	chatWhisper
)

type chatSender struct {
	name string
	id   uint32
}

type chatRepeat struct {
	id       uint64
	key      string
	last     time.Time
	items    proto.ChatItems
	audience chatAudience
}

type chatRate struct {
	start time.Time
	count uint8
}

type chatActivity struct {
	last    time.Time
	rate    [4]chatRate
	repeats [4]chatRepeat
	next    int
}

type chatProtection struct {
	senders map[chatSender]*chatActivity
	serial  uint64
}

func (g *chatProtection) activity(sender chatSender, now time.Time) *chatActivity {
	if activity := g.senders[sender]; activity != nil {
		activity.last = now
		return activity
	}
	if g.senders == nil {
		g.senders = make(map[chatSender]*chatActivity)
	} else if len(g.senders) >= chatSenderLimit {
		var oldest chatSender
		var last time.Time
		for key, entry := range g.senders {
			if last.IsZero() || entry.last.Before(last) {
				oldest, last = key, entry.last
			}
		}
		delete(g.senders, oldest)
	}
	activity := &chatActivity{last: now}
	g.senders[sender] = activity
	return activity
}

func (s *State) ownChat(l ChatLine) bool {
	if l.Op == proto.ChatWhisper {
		return l.Direction == 1
	}
	if s.SelfID == 0 {
		return false
	}
	if l.SenderID != 0 {
		return l.SenderID == s.SelfID
	}
	for _, slot := range s.Slots {
		if slot.ID == s.SelfID {
			return strings.EqualFold(slot.Name, l.Sender)
		}
	}
	return false
}

func (s *State) verifiedStaffChat(line ChatLine, own bool) bool {
	if line.Op == proto.ChatNormalAdmin || line.Op == proto.ChatWorldAdmin || own && s.Self.IsGM {
		return true
	}
	actor := s.Entities[line.SenderID]
	return actor != nil && actor.Kind == KindUser && actor.IsGM
}

func (l ChatLine) audience() chatAudience {
	switch l.Op {
	case proto.ChatParty:
		return chatParty
	case proto.ChatGuild:
		return chatGuild
	case proto.ChatWhisper:
		return chatWhisper
	default:
		return chatPublic
	}
}

// receivePlayerChat returns true when ancillary presentation must stay quiet.
// This is the only receive path for player chat, before history is appended.
func (s *State) receivePlayerChat(line ChatLine, now time.Time) bool {
	s.resolveChatSender(&line)
	if s.blockedChat(line) {
		return true
	}
	own := s.ownChat(line)
	sender := chatSender{name: strings.ToLower(line.Sender)}
	if sender.name == "" {
		sender.id = line.SenderID
	}
	line.Sender = chatguard.CleanText(line.Sender)
	checked := chatguard.Filter(line.Text, s.verifiedStaffChat(line, own))
	line.Text, line.ChatProtected = checked.Text, checked.Quiet
	if checked.Impersonation {
		line.Text = "[Possible staff impersonation. Mute this player in Guild › Block.]"
		line.Items = proto.ChatItems{}
	}
	if strings.TrimSpace(line.Text) == "" && line.Items.Len() == 0 {
		return true
	}
	if own {
		s.appendChat(line)
		return checked.Quiet
	}

	activity := s.chatGuard.activity(sender, now)
	audience := line.audience()
	rate := &activity.rate[audience]
	if rate.start.IsZero() || now.Sub(rate.start) >= chatRateWindow {
		*rate = chatRate{start: now}
	}
	limit := uint8(6)
	if audience == chatParty || audience == chatGuild {
		limit = 12
	}
	// Count new messages separately. Repetitions update a single visible line
	// without a fresh alert, even across public channels.
	key := chatguard.SpamKey(line.Text)
	for i := range activity.repeats {
		recent := &activity.repeats[i]
		if recent.id == 0 || now.Sub(recent.last) > chatRepeatWindow || recent.audience != audience || recent.items != line.Items || !chatguard.Similar(recent.key, key) {
			continue
		}
		for j := len(s.Chat) - 1; j >= 0; j-- {
			if shown := &s.Chat[j]; shown.chatSerial == recent.id {
				shown.Repetitions = min(9999, max(1, shown.Repetitions)+1)
				recent.last = now
				return true
			}
		}
		recent.id = 0
	}
	if rate.count >= limit {
		if rate.count == limit {
			rate.count++
			line.Text = "[Rapid messages hidden. Mute this player in Guild › Block.]"
			line.Items = proto.ChatItems{}
			line.ChatProtected = true
			s.appendChat(line)
		}
		return true
	}
	rate.count++
	s.chatGuard.serial++
	line.chatSerial = s.chatGuard.serial
	activity.repeats[activity.next] = chatRepeat{id: line.chatSerial, key: key, last: now, items: line.Items, audience: audience}
	activity.next = (activity.next + 1) % len(activity.repeats)
	s.appendChat(line)
	return checked.Quiet
}
