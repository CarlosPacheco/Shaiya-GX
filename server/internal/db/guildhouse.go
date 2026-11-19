package db

import (
	"context"
	"database/sql"
)

// GuildNpcRow is one usp_Read_GuildNpcLv_R row: a guild house NPC and its
// level in the guild's slot (Number).
type GuildNpcRow struct {
	GuildID uint32
	Type    uint16
	Level   uint8
	Slot    uint8
}

// LoadGuildNpcLevels reads every live guild NPC level (usp_Read_GuildNpcLv_R).
func LoadGuildNpcLevels() ([]GuildNpcRow, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT GuildID, GNpcType, NpcLevel, Number FROM PS_GameData_GuildNpcLv WHERE Del = 0 ORDER BY GuildID, Number`)
	if err != nil {
		reportStorageFailure(err)
		return nil, err
	}
	defer rows.Close()
	var out []GuildNpcRow
	for rows.Next() {
		var g, t, l, n int64
		if err := rows.Scan(&g, &t, &l, &n); err != nil {
			return nil, err
		}
		out = append(out, GuildNpcRow{GuildID: uint32(g), Type: uint16(t), Level: uint8(l), Slot: uint8(n)})
	}
	return out, rows.Err()
}

// CommitGuildHouse stores a guild house change atomically: GuildDetails'
// house fields (the DB agent's 0x0B0F/0x0B10/0x0B12 writes), optionally the NPC
// level row (usp_Save_GuildNpcLv_Add_E / _Mod_E: the row of that type takes
// the level and slot) and the character's own change (money or the consumed
// etin items).
func CommitGuildHouse(g GuildRow, npc *GuildNpcRow, change *CharacterChange) error {
	p, e := Pool(GameData)
	if e != nil {
		return e
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, p, func(tx *sql.Tx) error {
		if change != nil {
			if e := commitGameplayTx(c, tx, []CharacterChange{*change}, nil); e != nil {
				return e
			}
		}
		if npc != nil {
			if _, e := tx.ExecContext(c, `DELETE FROM PS_GameData_GuildNpcLv WHERE GuildID=?1 AND (GNpcType=?2 OR Number=?3)`,
				int32(npc.GuildID), int32(npc.Type), int32(npc.Slot)); e != nil {
				return e
			}
			if _, e := tx.ExecContext(c, `INSERT INTO PS_GameData_GuildNpcLv(GuildID,GNpcType,NpcLevel,Number) VALUES(?1,?2,?3,?4)`,
				int32(npc.GuildID), int32(npc.Type), int32(npc.Level), int32(npc.Slot)); e != nil {
				return e
			}
		}
		return saveGuildHouseTx(c, tx, g)
	})
}

func saveGuildHouseTx(c context.Context, tx *sql.Tx, g GuildRow) error {
	_, e := tx.ExecContext(c, `INSERT INTO PS_GameData_GuildDetails(GuildID,UseHouse,BuyHouse,Rank,Etin,EtinReturnCnt,KeepEtin,Remark) VALUES(?1,?2,?3,?4,?5,?6,?7,?8)
   ON CONFLICT(GuildID) DO UPDATE SET UseHouse=excluded.UseHouse,BuyHouse=excluded.BuyHouse,Etin=excluded.Etin,EtinReturnCnt=excluded.EtinReturnCnt,KeepEtin=excluded.KeepEtin`,
		int32(g.ID), g.UseHouse, g.BuyHouse, g.Rank, int32(g.Etin), int32(g.EtinReturn), int32(g.KeepEtin), g.Remark)
	return e
}
