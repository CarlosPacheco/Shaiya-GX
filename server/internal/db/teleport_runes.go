package db

import "github.com/Spelunkern/Shaiya-GX/internal/teleportrunes"

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
	rules, err := teleportrunes.Read(ctx, pool)
	if err != nil {
		return nil, err
	}
	runes := make(map[int16][2]TeleportRuneDestination, len(rules))
	for _, r := range rules {
		a := r.Light
		b := r.Fury
		runes[r.Index] = [2]TeleportRuneDestination{
			{a.MapID, float32(a.X), float32(a.Y), float32(a.Z)},
			{b.MapID, float32(b.X), float32(b.Y), float32(b.Z)},
		}
	}
	return runes, nil
}
