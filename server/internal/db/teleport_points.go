package db

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

type TeleportPoint struct {
	Saved   bool
	Map     uint16
	X, Y, Z float32
}

func TeleportPoints(charID uint32) (points [proto.TeleportPointCount]TeleportPoint, err error) {
	pool, err := Pool(GameData)
	if err != nil {
		return points, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, "SELECT Slot, Map, X, Y, Z FROM GX_TeleportPoints WHERE CharID=?", charID)
	if err != nil {
		return points, err
	}
	defer rows.Close()
	for rows.Next() {
		var slot int
		var p TeleportPoint
		if err := rows.Scan(&slot, &p.Map, &p.X, &p.Y, &p.Z); err != nil {
			return points, err
		}
		if slot >= 0 && slot < len(points) {
			p.Saved = true
			points[slot] = p
		}
	}
	return points, rows.Err()
}

func SaveTeleportPoint(charID uint32, slot uint8, p TeleportPoint) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	result, err := pool.ExecContext(c, `INSERT INTO GX_TeleportPoints (CharID, Slot, Map, X, Y, Z)
		SELECT CharID, ?, ?, ?, ?, ? FROM PS_GameData_Chars WHERE CharID=? AND Del=0
		ON CONFLICT(CharID, Slot) DO UPDATE SET Map=excluded.Map, X=excluded.X, Y=excluded.Y, Z=excluded.Z`,
		slot, p.Map, p.X, p.Y, p.Z, charID)
	if err != nil {
		return err
	}
	return requireOne(result)
}
