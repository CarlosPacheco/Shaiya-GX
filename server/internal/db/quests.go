package db

import (
	"context"
	"database/sql"
	"fmt"
)

// CharQuest is an active quest row: Delay is minutes left (timed quests)
// or minutes elapsed (untimed), as the original saves it.
type CharQuest struct {
	QuestID uint16
	Delay   int16
	Counts  [3]uint8
}

// FinishedQuest is a finished quest row (Success 1 completed, 0 failed).
type FinishedQuest struct {
	QuestID uint16
	Success uint8
}

// CharacterQuests runs usp_Read_Char_Quest_R.
func CharacterQuests(charID uint32) ([]CharQuest, error) {
	pool, err := ReadPool()
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT QuestID, [Delay], Count1, Count2, Count3 FROM PS_GameData_CharQuests WHERE CharID=@charid AND Finish=0 AND Del=0`, sql.Named("charid", int32(charID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_Char_Quest_R: %w", err)
	}
	defer rows.Close()
	var out []CharQuest
	for rows.Next() {
		var q CharQuest
		var id int64
		if err := rows.Scan(&id, &q.Delay, &q.Counts[0], &q.Counts[1], &q.Counts[2]); err != nil {
			return nil, err
		}
		q.QuestID = clampU16(id)
		out = append(out, q)
	}
	return out, rows.Err()
}

// FinishedQuests runs usp_Read_Char_FinishedQuest_R.
func FinishedQuests(charID uint32) ([]FinishedQuest, error) {
	pool, err := ReadPool()
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT QuestID,Success FROM PS_GameData_CharQuests WHERE CharID=@charid AND Finish=1`, sql.Named("charid", int32(charID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_Char_FinishedQuest_R: %w", err)
	}
	defer rows.Close()
	var out []FinishedQuest
	for rows.Next() {
		var q FinishedQuest
		var id int64
		var success sql.NullInt64
		if err := rows.Scan(&id, &success); err != nil {
			return nil, err
		}
		q.QuestID, q.Success = clampU16(id), clampU8(success.Int64)
		out = append(out, q)
	}
	return out, rows.Err()
}

// AddCharacterQuest runs usp_Save_Char_Quest_Add_E.
func AddCharacterQuest(charID uint32, q CharQuest) error {
	return execSQL(GameData, `INSERT INTO PS_GameData_CharQuests(CharID,QuestID,[Delay],Count1,Count2,Count3) VALUES(@charid,@questid,@delay,@count1,@count2,@count3)`, sql.Named("charid", int32(charID)),
		sql.Named("questid", int16(q.QuestID)), sql.Named("delay", q.Delay), sql.Named("count1", q.Counts[0]),
		sql.Named("count2", q.Counts[1]), sql.Named("count3", q.Counts[2]))
}

// CommitQuestStart commits the inventory cost, starting grants and quest row
// together. A duplicate start or changed inventory rolls back the whole plan.
func CommitQuestStart(charID uint32, q CharQuest, before, after []Item) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error {
		if !sameInventory(before, after) {
			actual, err := characterItems(c, tx, charID)
			if err != nil {
				return err
			}
			if !sameInventory(actual, before) {
				return fmt.Errorf("quest start inventory changed")
			}
			change := CharacterChange{After: Character{ID: charID}, ItemsBefore: before, ItemsAfter: after}
			if err := writeInventoryChange(c, tx, change); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(c, `INSERT INTO PS_GameData_CharQuests(CharID,QuestID,Delay,Count1,Count2,Count3) VALUES(?,?,?,?,?,?)`,
			charID, int16(q.QuestID), q.Delay, q.Counts[0], q.Counts[1], q.Counts[2])
		return err
	})
}

// SaveCharacterQuest runs usp_Save_Char_Quest_Mod_E.
func SaveCharacterQuest(charID uint32, q CharQuest) error {
	return execSQL(GameData, `UPDATE PS_GameData_CharQuests SET [Delay]=@delay,Count1=@count1,Count2=@count2,Count3=@count3 WHERE CharID=@charid AND QuestID=@questid`, sql.Named("charid", int32(charID)),
		sql.Named("questid", int16(q.QuestID)), sql.Named("delay", q.Delay), sql.Named("count1", q.Counts[0]),
		sql.Named("count2", q.Counts[1]), sql.Named("count3", q.Counts[2]))
}

// FinishCharacterQuest runs usp_Save_Char_FinishedQuest_E on the active row.
func FinishCharacterQuest(charID uint32, questID uint16, success uint8) error {
	return execSQL(GameData, `UPDATE PS_GameData_CharQuests SET Success=@success,Finish=1 WHERE CharID=@charid AND QuestID=@questid`, sql.Named("charid", int32(charID)),
		sql.Named("questid", int16(questID)), sql.Named("success", success), sql.Named("insert", false))
}

// DeleteCharacterQuest runs usp_Save_Char_Quest_Del_E (repeatable quests).
func DeleteCharacterQuest(charID uint32, questID uint16) error {
	return execSQL(GameData, `DELETE FROM PS_GameData_CharQuests WHERE CharID=@charid AND QuestID=@questid`, sql.Named("charid", int32(charID)),
		sql.Named("questid", int16(questID)))
}

// SaveCharacterQuests atomically snapshots active counters. A stale/finished
// quest cannot be updated after another operation completed it.
func SaveCharacterQuests(charID uint32, quests []CharQuest) error {
	if len(quests) == 0 {
		return nil
	}
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error { return saveQuestsTx(c, tx, charID, quests) })
}

func saveQuestsTx(c context.Context, tx *sql.Tx, charID uint32, quests []CharQuest) error {
	stmt, err := tx.PrepareContext(c, `UPDATE PS_GameData_CharQuests SET Delay=?,Count1=?,Count2=?,Count3=? WHERE CharID=? AND QuestID=? AND Finish=0 AND Del=0`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, q := range quests {
		result, err := stmt.ExecContext(c, q.Delay, q.Counts[0], q.Counts[1], q.Counts[2], charID, q.QuestID)
		if err != nil {
			return err
		}
		if err = requireOne(result); err != nil {
			return err
		}
	}
	return nil
}
