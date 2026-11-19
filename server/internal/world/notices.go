package world

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

var noticeCatalog []db.ServerNotice

func loadServerNotices(log *logx.Logger) error {
	notices, err := db.ServerNotices()
	if err != nil {
		return err
	}
	for _, n := range notices {
		if !validNoticeMessage(n.Message) {
			return fmt.Errorf("notice %d: invalid message", n.Index)
		}
		for i, value := range n.Fixed {
			if _, valid := parseNoticeHour(value); strings.TrimSpace(value) != "" && !valid {
				log.Printf("GX Server World ignored invalid FixedHour%d for notice %d", i+1, n.Index)
			}
		}
	}
	noticeCatalog = notices
	return nil
}

type noticeHour struct{ day, hour, minute int }

func parseNoticeHour(text string) (noticeHour, bool) {
	fields := strings.Split(strings.TrimSpace(text), "/")
	if len(fields) != 3 {
		return noticeHour{}, false
	}
	day, d := strconv.ParseUint(fields[0], 10, 8)
	hour, h := strconv.ParseUint(fields[1], 10, 8)
	minute, m := strconv.ParseUint(fields[2], 10, 8)
	return noticeHour{int(day), int(hour), int(minute)}, d == nil && h == nil && m == nil &&
		day <= 6 && hour <= 23 && minute <= 59
}

type scheduledNotice struct {
	message  string
	interval time.Duration
	next     time.Time
	fixed    map[noticeHour]int // Last calendar date sent per weekly slot.
}

func scheduleNotices(rows []db.ServerNotice, start time.Time) []scheduledNotice {
	notices := make([]scheduledNotice, 0, len(rows))
	for _, row := range rows {
		n := scheduledNotice{message: row.Message, interval: time.Duration(row.Frequency) * time.Minute}
		for _, value := range row.Fixed {
			if hour, ok := parseNoticeHour(value); ok {
				if n.fixed == nil {
					n.fixed = make(map[noticeHour]int)
				}
				n.fixed[hour] = 0
			}
		}
		if len(n.fixed) == 0 {
			if n.interval == 0 {
				continue
			}
			n.next = start.Add(n.interval)
		}
		notices = append(notices, n)
	}
	return notices
}

func (n *scheduledNotice) due(now time.Time) bool {
	if len(n.fixed) != 0 {
		hour := noticeHour{int(now.Weekday()), now.Hour(), now.Minute()}
		last, found := n.fixed[hour]
		if !found {
			return false
		}
		date := now.Year()*1000 + now.YearDay()
		if last == date {
			return false
		}
		n.fixed[hour] = date
		return true
	}
	if now.Before(n.next) {
		return false
	}
	n.next = n.next.Add((now.Sub(n.next)/n.interval + 1) * n.interval)
	return true
}

func validNoticeMessage(message string) bool {
	return strings.TrimSpace(message) != "" && len(encodeText(message)) <= maxChatMessageLen &&
		!strings.ContainsAny(message, "\x00\r\n")
}

func broadcastNotice(message string) {
	w := packet.New(packet.Opcode(proto.GXServerNotice))
	writeChatText(w, message)
	frame := w.MustFinish()
	for _, p := range AllPlayers() {
		p.Send(frame, "global server notice")
	}
}

func (s *Session) gmNotice(message string) error {
	if suffix, found := strings.CutPrefix(strings.ToLower(message), "index"); found && suffix != "" &&
		strings.Trim(suffix, "0123456789") == "" {
		index, err := strconv.ParseUint(suffix, 10, 32)
		if err != nil || index == 0 {
			return fmt.Errorf("invalid notice index")
		}
		for _, n := range noticeCatalog {
			if uint64(n.Index) == index {
				broadcastNotice(n.Message)
				return nil
			}
		}
		return fmt.Errorf("notice index %d not found", index)
	}
	if !validNoticeMessage(message) {
		return fmt.Errorf("usage: /notice message or /notice indexX")
	}
	broadcastNotice(message)
	return nil
}
