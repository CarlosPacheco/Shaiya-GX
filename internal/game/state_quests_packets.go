package game

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func (s *State) applyQuestsPacket(p []byte, ev Event) (Event, error) {
	switch ev.Op {
	case proto.QuestList:
		quests, err := proto.DecodeQuestList(p)
		if err != nil {
			return ev.packetError(err)
		}
		if s.Quests == nil {
			s.Quests = make(map[uint16]proto.ActiveQuest)
		}
		for _, q := range quests {
			s.Quests[q.ID] = q
		}
	case proto.QuestFinishedList:
		quests, err := proto.DecodeFinishedQuests(p)
		if err != nil {
			return ev.packetError(err)
		}
		if s.FinishedQuests == nil {
			s.FinishedQuests = make(map[uint16]uint8)
		}
		for _, q := range quests {
			s.FinishedQuests[q.ID] = q.Success
		}
	case proto.QuestStart:
		_, id, err := proto.DecodeQuestStart(p)
		if err != nil {
			return ev.packetError(err)
		}
		if s.Quests == nil {
			s.Quests = make(map[uint16]proto.ActiveQuest)
		}
		s.Quests[id] = proto.ActiveQuest{ID: id}
	case proto.QuestCounter:
		id, slot, value, err := proto.DecodeQuestCounter(p)
		if err != nil {
			return ev.packetError(err)
		}
		if slot >= 3 {
			return ev.packetError(fmt.Errorf("invalid quest counter slot %d", slot))
		}
		if q, ok := s.Quests[id]; ok {
			q.Counters[slot] = value
			s.Quests[id] = q
		}
	case proto.QuestChoose:
		id, err := proto.DecodeQuestChoice(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.QuestChoice, s.QuestChoosing = id, true
	case proto.QuestEnd:
		result, err := proto.DecodeQuestEnd(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.LastQuestEnd = result
		delete(s.Quests, result.ID)
		if s.QuestChoosing && s.QuestChoice == result.ID {
			s.QuestChoosing = false
		}
	}
	return ev, nil
}
