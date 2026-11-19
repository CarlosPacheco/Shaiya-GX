package itemboxes

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"slices"

	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
)

const (
	Table        = "PS_GameDefs_ItemBoxes"
	RewardsTable = "PS_GameDefs_ItemBoxRewards"
	Normal       = 1
	Independent  = 2
	Grade        = 1
	Direct       = 2
)

//go:embed migration.sql
var MigrationSQL string

type Reward struct {
	Item   int32
	Chance uint64
}

type Box struct {
	Index             int16
	Description       string
	BoxType, ItemType uint8
	Rewards           []Reward
}

func Clone(boxes []Box) []Box {
	cloned := slices.Clone(boxes)
	for i := range cloned {
		cloned[i].Rewards = slices.Clone(cloned[i].Rewards)
	}
	return cloned
}

func Equal(a, b []Box) bool {
	return slices.EqualFunc(a, b, Box.Equal)
}

func (b Box) Equal(o Box) bool {
	return b.Index == o.Index && b.Description == o.Description && b.BoxType == o.BoxType &&
		b.ItemType == o.ItemType && slices.Equal(b.Rewards, o.Rewards)
}

func (b Box) Total() uint64 {
	var total uint64
	for _, reward := range b.Rewards {
		total += reward.Chance
	}
	return total
}

func Validate(boxes []Box) error {
	var prev int16
	for _, b := range boxes {
		if b.Index <= prev || (b.BoxType != Normal && b.BoxType != Independent) || (b.ItemType != Grade && b.ItemType != Direct) || len(b.Rewards) == 0 {
			return fmt.Errorf("box %d requires a unique positive index, valid types and at least one reward", b.Index)
		}
		prev = b.Index
		limit := uint64(1000)
		if b.BoxType == Independent {
			limit = worlddrops.RollRange
		}
		for _, r := range b.Rewards {
			if r.Item <= 0 || (b.ItemType == Grade && r.Item > 1024) || r.Chance > limit {
				return fmt.Errorf("box %d has an invalid reward item or chance", b.Index)
			}
		}
		if b.BoxType == Normal && b.Total() == 0 {
			return fmt.Errorf("box %d needs at least one positive weight", b.Index)
		}
	}
	return nil
}

func Read(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) ([]Box, error) {
	rows, err := db.QueryContext(ctx, `SELECT b."Index",b.Description,b.BoxType,b.ItemType,r.Position,r.Item,r.Chance
		FROM `+Table+` b LEFT JOIN `+RewardsTable+` r ON r.BoxIndex=b."Index" ORDER BY b."Index",r.Position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var boxes []Box
	for rows.Next() {
		var b Box
		var pos, item, chance sql.NullInt64
		if err := rows.Scan(&b.Index, &b.Description, &b.BoxType, &b.ItemType, &pos, &item, &chance); err != nil {
			return nil, err
		}
		if len(boxes) == 0 || boxes[len(boxes)-1].Index != b.Index {
			boxes = append(boxes, b)
		}
		if pos.Valid {
			box := &boxes[len(boxes)-1]
			if pos.Int64 != int64(len(box.Rewards)+1) || !item.Valid || !chance.Valid || item.Int64 <= 0 || item.Int64 > 2147483647 || chance.Int64 < 0 {
				return nil, fmt.Errorf("invalid reward at box %d position %d", b.Index, pos.Int64)
			}
			box.Rewards = append(box.Rewards, Reward{int32(item.Int64), uint64(chance.Int64)})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return boxes, Validate(boxes)
}
