package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// CharacterSkill is one learned skill. Delay is the remaining cooldown in
// seconds as last saved (CharSkills.Delay).
type CharacterSkill struct {
	SkillID uint16
	Level   uint8
	Number  uint8
	Delay   int16
}

// CharacterSkills runs usp_Read_Char_Skills_R.
func CharacterSkills(charID uint32) ([]CharacterSkill, error) {
	pool, err := ReadPool()
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT SkillID,SkillLevel,Number,[Delay] FROM PS_GameData_CharSkills WHERE CharID=@charid AND Del=0 ORDER BY Number ASC`, sql.Named("charid", int32(charID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_Char_Skills_R: %w", err)
	}
	defer rows.Close()
	var skills []CharacterSkill
	for rows.Next() {
		var s CharacterSkill
		var id int64
		if err := rows.Scan(&id, &s.Level, &s.Number, &s.Delay); err != nil {
			return nil, err
		}
		s.SkillID = clampU16(id)
		skills = append(skills, s)
	}
	return skills, rows.Err()
}

// AddCharacterSkill runs usp_Save_Char_Skill_Add_E.
func AddCharacterSkill(charID uint32, s CharacterSkill) error {
	return execSQL(GameData, `INSERT INTO PS_GameData_CharSkills(CharID,SkillID,SkillLevel,Number,[Delay]) VALUES(@charid,@skillid,@skilllevel,@number,@delay)`, sql.Named("charid", int32(charID)),
		sql.Named("skillid", int16(s.SkillID)), sql.Named("skilllevel", s.Level),
		sql.Named("number", s.Number), sql.Named("delay", s.Delay))
}

// SaveCharacterSkill runs usp_Save_Char_Skill_Mod_E (level and remaining
// cooldown).
func SaveCharacterSkill(charID uint32, s CharacterSkill) error {
	return execSQL(GameData, `UPDATE PS_GameData_CharSkills SET SkillLevel=@skilllevel,[Delay]=@delay WHERE CharID=@charid AND SkillID=@skillid`, sql.Named("charid", int32(charID)),
		sql.Named("skillid", int16(s.SkillID)), sql.Named("skilllevel", s.Level), sql.Named("delay", s.Delay))
}

// QuickSlot is one skill-bar entry.
type QuickSlot struct {
	QuickBar, QuickSlot, Bag uint8
	Number                   uint16
}

// QuickSlots runs usp_Read_Char_QuickSlots_R.
func QuickSlots(charID uint32) ([]QuickSlot, error) {
	pool, err := ReadPool()
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT QuickBar,QuickSlot,Bag,Number FROM PS_GameData_CharQuickSlots WHERE CharID=@charid`, sql.Named("charid", int32(charID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_Char_QuickSlots_R: %w", err)
	}
	defer rows.Close()
	var slots []QuickSlot
	for rows.Next() {
		var q QuickSlot
		var number int64
		if err := rows.Scan(&q.QuickBar, &q.QuickSlot, &q.Bag, &number); err != nil {
			return nil, err
		}
		q.Number = clampU16(number)
		slots = append(slots, q)
	}
	return slots, rows.Err()
}

// SaveQuickSlots atomically applies the complete bar using two set operations.
// Sending the same bar does not rewrite its unchanged slots.
func SaveQuickSlots(charID uint32, slots []QuickSlot) error {
	if len(slots) == 0 {
		return execSQL(GameData, "DELETE FROM PS_GameData_CharQuickSlots WHERE CharID=?", charID)
	}
	entries := make([][3]uint16, 0, min(128, len(slots)))
	seen := make(map[uint16]bool, cap(entries))
	for _, q := range slots[:min(128, len(slots))] {
		key := uint16(q.QuickBar)*256 + uint16(q.QuickSlot)
		if seen[key] {
			return fmt.Errorf("duplicate quick slot")
		}
		seen[key] = true
		entries = append(entries, [3]uint16{key, uint16(q.Bag), q.Number})
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return groupWrite(c, pool, func(c context.Context, tx *sql.Tx) error {
		const keys = `SELECT CAST(value->>0 AS INTEGER) FROM json_each(?)`
		if _, err := tx.ExecContext(c, `DELETE FROM PS_GameData_CharQuickSlots WHERE CharID=? AND QuickBar*256+QuickSlot NOT IN (`+keys+`)`, charID, string(data)); err != nil {
			return err
		}
		_, err := tx.ExecContext(c, `INSERT INTO PS_GameData_CharQuickSlots(CharID,QuickBar,QuickSlot,Bag,Number)
			SELECT ?, (value->>0)/256, (value->>0)%256, value->>1, value->>2 FROM json_each(?) WHERE true
			ON CONFLICT(CharID,QuickBar,QuickSlot) DO UPDATE SET Bag=excluded.Bag,Number=excluded.Number
			WHERE Bag IS NOT excluded.Bag OR Number IS NOT excluded.Number`, charID, string(data))
		return err
	})
}

// AppliedBuff is a persisted active buff with its remaining seconds.
type AppliedBuff struct {
	SkillID     uint16
	Level       uint8
	LeftSeconds uint32
}

// AppliedBuffs runs usp_Read_Char_ApplySkills_R.
func AppliedBuffs(charID uint32) ([]AppliedBuff, error) {
	pool, err := ReadPool()
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT SkillID,SkillLevel,LeftResetTime FROM PS_GameData_CharApplySkills WHERE CharID=@charid`, sql.Named("charid", int32(charID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_Char_ApplySkills_R: %w", err)
	}
	defer rows.Close()
	var buffs []AppliedBuff
	for rows.Next() {
		var b AppliedBuff
		var id int64
		var left int32
		if err := rows.Scan(&id, &b.Level, &left); err != nil {
			return nil, err
		}
		b.SkillID, b.LeftSeconds = clampU16(id), uint32(max(left, 0))
		buffs = append(buffs, b)
	}
	return buffs, rows.Err()
}

// SaveAppliedBuffs replaces the persisted buffs
// (usp_Save_Char_ApplySkill_Del_E with DeleteAll, then
// usp_Save_Char_ApplySkill_Add_E2 per buff).
func SaveAppliedBuffs(charID uint32, buffs []AppliedBuff) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error { return saveBuffsTx(c, tx, charID, buffs) })
}

func saveBuffsTx(c context.Context, tx *sql.Tx, charID uint32, buffs []AppliedBuff) error {

	if _, err := tx.ExecContext(c, `DELETE FROM PS_GameData_CharApplySkills WHERE CharID=?`, charID); err != nil {
		return err
	}
	for _, b := range buffs {
		if _, err := tx.ExecContext(c, `INSERT INTO PS_GameData_CharApplySkills(CharID,Skillid,SkillLevel,LeftResetTime) VALUES(@charid,@skillid,@skilllevel,@leftresettime)`, sql.Named("charid", int32(charID)),
			sql.Named("skillid", int16(b.SkillID)), sql.Named("skilllevel", b.Level),
			sql.Named("leftresettime", int32(b.LeftSeconds))); err != nil {
			return err
		}
	}
	return nil
}

// CommitLearnSkill guards both the previous skill level and available points;
// learning never grants a skill without committing its cost.
func CommitLearnSkill(before, after Character, previous *CharacterSkill, learned CharacterSkill) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	if before.ID == 0 || after.ID != before.ID || after.SkillPoint > before.SkillPoint {
		return fmt.Errorf("invalid skill plan")
	}
	return withTx(c, pool, func(tx *sql.Tx) error {
		var actual CharacterSkill
		err := tx.QueryRowContext(c, `SELECT SkillID,SkillLevel,Number,Delay FROM PS_GameData_CharSkills WHERE CharID=? AND SkillID=? AND Del=0`, before.ID, learned.SkillID).Scan(&actual.SkillID, &actual.Level, &actual.Number, &actual.Delay)
		if previous == nil {
			if err != sql.ErrNoRows {
				return fmt.Errorf("skill already learned or unavailable: %v", err)
			}
		} else if err != nil || actual.Level != previous.Level || actual.Number != previous.Number {
			return fmt.Errorf("skill level changed")
		}
		if previous == nil {
			if _, err = tx.ExecContext(c, `INSERT INTO PS_GameData_CharSkills(CharID,SkillID,SkillLevel,Number,Delay) VALUES(?,?,?,?,?)`, before.ID, learned.SkillID, learned.Level, learned.Number, learned.Delay); err != nil {
				return err
			}
		} else {
			if _, err = tx.ExecContext(c, `UPDATE PS_GameData_CharSkills SET SkillLevel=?,Delay=? WHERE CharID=? AND SkillID=? AND Del=0`, learned.Level, learned.Delay, before.ID, learned.SkillID); err != nil {
				return err
			}
		}
		result, err := tx.ExecContext(c, `UPDATE PS_GameData_Chars SET SkillPoint=? WHERE CharID=? AND SkillPoint=? AND Del=0`, after.SkillPoint, before.ID, before.SkillPoint)
		if err != nil {
			return err
		}
		return requireOne(result)
	})
}

func SaveSkillDelays(charID uint32, skills []CharacterSkill) error {
	if len(skills) == 0 {
		return nil
	}
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error { return saveSkillDelaysTx(c, tx, charID, skills) })
}

func saveSkillDelaysTx(c context.Context, tx *sql.Tx, charID uint32, skills []CharacterSkill) error {
	stmt, err := tx.PrepareContext(c, `UPDATE PS_GameData_CharSkills SET Delay=? WHERE CharID=? AND SkillID=? AND Del=0`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, skill := range skills {
		result, err := stmt.ExecContext(c, skill.Delay, charID, skill.SkillID)
		if err != nil {
			return err
		}
		if err = requireOne(result); err != nil {
			return err
		}
	}
	return nil
}
