package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// CreateGuildGroup commits the group, membership, mirror and master's fee in
// one transaction. The world holds the master's state lock until publication.
func CreateGuildGroup(id uint32, name, masterUserID, remark string, country uint8, master Character, fee uint32, members []GuildCharRow) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error {
		if master.Money < fee {
			return fmt.Errorf("not enough guild creation funds")
		}
		var exists int
		if err := tx.QueryRowContext(c, `SELECT COUNT(*) FROM PS_GameData_Guilds WHERE GuildName=? AND Del=0`, name).Scan(&exists); err != nil {
			return err
		}
		if exists != 0 {
			return fmt.Errorf("guild name already exists")
		}
		seen := map[uint32]bool{}
		for _, member := range members {
			if seen[member.CharID] {
				return fmt.Errorf("duplicate guild founder")
			}
			seen[member.CharID] = true
			if err := tx.QueryRowContext(c, `SELECT COUNT(*) FROM PS_GameData_GuildChars WHERE CharID=? AND Del=0`, member.CharID).Scan(&exists); err != nil {
				return err
			}
			if exists != 0 {
				return fmt.Errorf("founder already belongs to a guild")
			}
		}
		if !seen[master.ID] {
			return fmt.Errorf("guild master not in founding group")
		}
		if _, err := tx.ExecContext(c, `INSERT INTO PS_GameData_Guilds(GuildID,GuildName,MasterUserID,MasterCharID,MasterName,Country,TotalCount,GuildPoint,CreateDate) VALUES(?,?,?,?,?,?,?,0,CURRENT_TIMESTAMP)`, id, strings.TrimSpace(name), masterUserID, master.ID, master.Name, country, len(members)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(c, `INSERT INTO PS_GameData_GuildDetails(GuildID,Remark) VALUES(?,?)`, id, strings.TrimSpace(remark)); err != nil {
			return err
		}
		for _, member := range members {
			if _, err := tx.ExecContext(c, `INSERT INTO PS_GameData_GuildChars(GuildID,CharID,GuildLevel,JoinDate) VALUES(?,?,?,CURRENT_TIMESTAMP)`, id, member.CharID, member.Rank); err != nil {
				return err
			}
		}
		result, err := tx.ExecContext(c, `UPDATE PS_GameData_Chars SET Money=Money-? WHERE CharID=? AND Money=? AND Money>=? AND Del=0`, fee, master.ID, master.Money, fee)
		if err != nil {
			return err
		}
		if err = requireOne(result); err != nil {
			return fmt.Errorf("guild master funds changed: %w", err)
		}
		return mirrorGuildTx(c, tx, id)
	})
}
