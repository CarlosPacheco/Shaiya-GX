package db

import (
	"fmt"
	"math"
)

type TeleportRuneDestination struct {
	MapID   uint16
	X, Y, Z float32
}

func TeleportRunes() (map[int16][2]TeleportRuneDestination, error) {
	pool, err := Pool(GameDefs)
	if err != nil {
		return nil, err
	}
	ctx, cancel := longCtx()
	defer cancel()
	rows, err := pool.QueryContext(ctx, `SELECT "Index",
		LightMapID, LightX, LightY, LightZ, FuryMapID, FuryX, FuryY, FuryZ
		FROM PS_GameDefs_TeleportRunes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runes := make(map[int16][2]TeleportRuneDestination)
	for rows.Next() {
		var index int16
		var dest [2]TeleportRuneDestination
		if err := rows.Scan(&index, &dest[0].MapID, &dest[0].X, &dest[0].Y, &dest[0].Z,
			&dest[1].MapID, &dest[1].X, &dest[1].Y, &dest[1].Z); err != nil {
			return nil, err
		}
		for _, d := range dest {
			for _, v := range [...]float32{d.X, d.Y, d.Z} {
				if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
					return nil, fmt.Errorf("teleport rune %d has a non-finite coordinate", index)
				}
			}
		}
		runes[index] = dest
	}
	return runes, rows.Err()
}
