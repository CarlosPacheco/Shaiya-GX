package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func (s *Session) gmSetLevel(args []string) error {
	defs := db.MustDefs()
	values, err := gmNumbers(args, uint64(defs.LevelCap))
	if err != nil {
		return newGMError(12433, defs.LevelCap)
	}
	level := uint16(values[0])
	if level != s.char.Level {
		next := *s.char
		next.Level, next.Exp = level, defs.Exp[level-1]
		normalize(&next)
		stats := StatsFor(&next, s.equipment())
		next.CurrentHP, next.CurrentMP, next.CurrentSP = stats.CurrentHP, stats.CurrentMP, stats.CurrentSP
		if err := db.SaveCharacter(&next); err != nil {
			return err
		}
		*s.char = next
		s.syncPlayer()
		// Refresh the roster slot and experience instead of announcing a
		// gameplay level-up, which would grant a primary attribute client-side.
		s.send(buildCharacterListSlot(s.char, s.items), "GM set level")
		s.send(buildCharacterDetails(s.char, &stats), "GM set level experience")
		s.sendStatPackets()
		s.sendPlayerFeedback(proto.PlayerLevelFeedback)
		s.partyLevel()
		s.partyPools()
		if s.player != nil {
			s.guildLevelChanged()
		}
	}
	s.gmMessage(12432, level)
	return nil
}
