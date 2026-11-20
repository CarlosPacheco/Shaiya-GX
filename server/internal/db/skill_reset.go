package db

import (
	"database/sql"
	"time"
)

func ResetSkillCooldowns(charID uint32) (map[uint16]time.Time, error) {
	pool, err := ReadPool()
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, "SELECT SkillID,ExpiresAt FROM GX_ResetSkillCooldowns WHERE CharID=? AND ExpiresAt>?", charID, time.Now().UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deadlines := make(map[uint16]time.Time)
	for rows.Next() {
		var id uint16
		var expires int64
		if err := rows.Scan(&id, &expires); err != nil {
			return nil, err
		}
		deadlines[id] = time.UnixMilli(expires)
	}
	return deadlines, rows.Err()
}

func CommitSkillReset(change CharacterChange, deadlines map[uint16]time.Time) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error {
		if err := commitGameplayTx(c, tx, []CharacterChange{change}, nil); err != nil {
			return err
		}
		id := change.After.ID
		now := time.Now()
		if _, err := tx.ExecContext(c, "DELETE FROM GX_ResetSkillCooldowns WHERE CharID=? AND ExpiresAt<=?", id, now.UnixMilli()); err != nil {
			return err
		}
		for skillID, expires := range deadlines {
			if !expires.After(now) {
				continue
			}
			if _, err := tx.ExecContext(c, `INSERT INTO GX_ResetSkillCooldowns VALUES(?,?,?)
				ON CONFLICT(CharID,SkillID) DO UPDATE SET ExpiresAt=max(ExpiresAt,excluded.ExpiresAt)`, id, skillID, expires.UnixMilli()); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(c, "DELETE FROM PS_GameData_CharQuickSlots WHERE CharID=? AND Bag=100", id); err != nil {
			return err
		}
		_, err := tx.ExecContext(c, "DELETE FROM PS_GameData_CharSkills WHERE CharID=?", id)
		return err
	})
}
