package world

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type gmError struct {
	reply proto.GMReply
}

func (e gmError) Error() string {
	return text.Defaults.Format(int(e.reply.ID), e.reply.Values)
}

func gmReplyData(id uint16, args ...any) proto.GMReply {
	reply := proto.GMReply{ID: id}
	if len(args) != 0 {
		reply.Values = make(map[string]string, len(args))
		for i, arg := range args {
			key := "v"
			if i > 0 {
				key += strconv.Itoa(i + 1)
			}
			reply.Values[key] = fmt.Sprint(arg)
		}
	}
	return reply
}

func newGMError(id uint16, args ...any) error {
	return gmError{reply: gmReplyData(id, args...)}
}

func (s *Session) gmMessage(id uint16, args ...any) {
	s.sendGMMessage(gmReplyData(id, args...))
}

func (s *Session) sendGMMessage(reply proto.GMReply) {
	s.gmReply(text.Defaults.Format(int(reply.ID), reply.Values), reply)
}

func (s *Session) gmFailure(err error) {
	var localized gmError
	if errors.As(err, &localized) {
		s.sendGMMessage(localized.reply)
		return
	}
	s.gmMessage(12378)
}
