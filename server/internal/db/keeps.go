package db

import "database/sql"

// Keep mirrors the original Keeps row / DB-agent 0206 and 0207 record.
// Country packs runtime owner (low nibble) and last conqueror+1 (high nibble).
// ResetTime is the original packed calendar value, not a Unix timestamp.
type Keep struct {
	ID        uint32
	Country   uint8
	ResetTime uint32
}

func ReadKeeps() (map[uint32]Keep, error) {
	p, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := p.QueryContext(c, `SELECT KeepID,OwnCountry,ResetTime FROM PS_GameData_Keeps ORDER BY RowID`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uint32]Keep)
	for rows.Next() {
		var id, country, reset int64
		if err = rows.Scan(&id, &country, &reset); err != nil {
			return nil, err
		}
		k := Keep{uint32(id), uint8(country), uint32(reset)}
		if _, ok := out[k.ID]; !ok {
			out[k.ID] = k
		}
	}
	return out, rows.Err()
}

// SaveKeeps updates/creates only the supplied identities, atomically. Unrelated
// keeps and all account/character data survive a world restart or capture.
func SaveKeeps(keeps []Keep) error {
	p, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, p, func(tx *sql.Tx) error {
		for _, k := range keeps {
			r, e := tx.ExecContext(c, `UPDATE PS_GameData_Keeps SET OwnCountry=?,ResetTime=? WHERE KeepID=?`, k.Country, int32(k.ResetTime), k.ID)
			if e != nil {
				return e
			}
			n, e := r.RowsAffected()
			if e != nil {
				return e
			}
			if n == 0 {
				if _, e = tx.ExecContext(c, `INSERT INTO PS_GameData_Keeps(KeepID,OwnCountry,ResetTime) VALUES(?,?,?)`, k.ID, k.Country, int32(k.ResetTime)); e != nil {
					return e
				}
			}
		}
		return nil
	})
}
