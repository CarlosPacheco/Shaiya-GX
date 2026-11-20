package db

import (
	"context"
	"database/sql"
)

// SaveCharacterExit commits the final character state and offline timers
// together. Other sessions can share the durable commit.
func SaveCharacterExit(ch Character, buffs []AppliedBuff, skills []CharacterSkill, quests []CharQuest) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return groupWrite(c, pool, func(c context.Context, tx *sql.Tx) error {
		result, err := writeCharacter(c, tx, &ch)
		if err != nil {
			return err
		}
		if err := requireOne(result); err != nil {
			return err
		}
		if err := saveBuffsTx(c, tx, ch.ID, buffs); err != nil {
			return err
		}
		if len(skills) != 0 {
			if err := saveSkillDelaysTx(c, tx, ch.ID, skills); err != nil {
				return err
			}
		}
		if len(quests) == 0 {
			return nil
		}
		return saveQuestsTx(c, tx, ch.ID, quests)
	})
}
