package catalogfiles

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/gameini"
	"github.com/Spelunkern/Shaiya-GX/internal/itemboxes"
	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
)

func ParseWorldDrops(raw []byte) ([]worlddrops.Rule, error) {
	sections, err := entries(raw, "Drop_", 63)
	if err != nil {
		return nil, err
	}
	rows := make([]worlddrops.Rule, 0, len(sections))
	for _, section := range sections {
		r := gameini.Record{Fields: section.Fields}
		row := worlddrops.Rule{Index: int64(section.Index), Description: r.Text("Description"),
			Grade: uint16(r.Uint("Grade", 16)), MinLevel: uint16(r.Uint("MinLevel", 16)), MaxLevel: uint16(r.Uint("MaxLevel", 16)),
			DropChance: r.Percent("DropChance", worlddrops.ParseChance)}
		if r.Err != nil {
			return nil, fmt.Errorf("drop %d: %w", section.Index, r.Err)
		}
		rows = append(rows, row)
	}
	return rows, worlddrops.Validate(rows)
}

func EncodeWorldDrops(rows []worlddrops.Rule) []byte {
	var b strings.Builder
	b.WriteString("; Each eligible rule rolls independently. DropChance is a literal percentage.\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "\n[Drop_%d]\nDescription=%q\nGrade=%d\nMinLevel=%d\nMaxLevel=%d\nDropChance=%s\n",
			r.Index, r.Description, r.Grade, r.MinLevel, r.MaxLevel, worlddrops.FormatChance(r.DropChance))
	}
	return []byte(b.String())
}

func ParseItemBoxes(raw []byte) ([]itemboxes.Box, error) {
	sections, err := entries(raw, "Box_", 15)
	if err != nil {
		return nil, err
	}
	rows := make([]itemboxes.Box, 0, len(sections))
	for _, section := range sections {
		r := gameini.Record{Fields: section.Fields}
		box := itemboxes.Box{Index: int16(section.Index), Description: r.Text("Description"),
			BoxType: uint8(r.Uint("BoxType", 8)), ItemType: uint8(r.Uint("ItemType", 8))}
		if r.Err != nil {
			return nil, fmt.Errorf("box %d: %w", section.Index, r.Err)
		}
		for i := 1; ; i++ {
			key := fmt.Sprintf("REWARD%d", i)
			text, found := section.Fields[key]
			if !found {
				break
			}
			parts := gameini.Fields(text)
			if len(parts) != 2 {
				return nil, fmt.Errorf("box %d %s requires item and chance", box.Index, key)
			}
			item, itemErr := strconv.ParseInt(parts[0], 10, 32)
			parse := worlddrops.ParseChance
			if box.BoxType == itemboxes.Normal {
				parse = func(s string) (uint64, error) { return strconv.ParseUint(s, 10, 64) }
			}
			chance, chanceErr := parse(parts[1])
			if itemErr != nil || chanceErr != nil {
				return nil, fmt.Errorf("box %d invalid %s", box.Index, key)
			}
			box.Rewards = append(box.Rewards, itemboxes.Reward{Item: int32(item), Chance: chance})
		}
		for key := range section.Fields {
			if suffix, ok := strings.CutPrefix(key, "REWARD"); ok {
				n, err := strconv.Atoi(suffix)
				if err != nil || n < 1 || n > len(box.Rewards) || suffix != strconv.Itoa(n) {
					return nil, fmt.Errorf("box %d reward keys must be consecutive from Reward1", box.Index)
				}
			}
		}
		rows = append(rows, box)
	}
	return rows, itemboxes.Validate(rows)
}

func EncodeItemBoxes(rows []itemboxes.Box) []byte {
	var b strings.Builder
	b.WriteString("; Box index matches ReqVg. BoxType 1 picks one reward by weight (0-1000).\n; BoxType 2 rolls each reward by literal percentage. ItemType 1 is grade, 2 is ItemID.\n")
	for _, box := range rows {
		fmt.Fprintf(&b, "\n[Box_%d]\nDescription=%q\nBoxType=%d\nItemType=%d\n", box.Index, box.Description, box.BoxType, box.ItemType)
		for i, r := range box.Rewards {
			chance := strconv.FormatUint(r.Chance, 10)
			if box.BoxType == itemboxes.Independent {
				chance = worlddrops.FormatChance(r.Chance)
			}
			fmt.Fprintf(&b, "Reward%d=%d, %s\n", i+1, r.Item, chance)
		}
	}
	return []byte(b.String())
}
