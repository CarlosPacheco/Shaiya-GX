package world

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/data/ini"
)

// DungeonMinimap contains native pixels per world unit.
type DungeonMinimap struct{ X, Z float32 }

var dungeonSection = regexp.MustCompile(`(?i)^map\s+(\d+)$`)

// ParseDungeonMinimap reads DungeonMiniMaps.INI, one section per numbered image:
//
//	[Map 64]
//	ScaleX=1.036
//	ScaleZ=1.052
func ParseDungeonMinimap(raw []byte) (map[uint16]DungeonMinimap, error) {
	result := map[uint16]DungeonMinimap{}
	for _, section := range ini.Parse(raw) {
		if section.Name == "" {
			continue
		}
		match := dungeonSection.FindStringSubmatch(section.Name)
		if match == nil {
			return nil, fmt.Errorf("DungeonMiniMaps.INI section %q is invalid", section.Name)
		}
		id, idErr := strconv.ParseUint(match[1], 10, 16)
		x, xErr := strconv.ParseFloat(first(section, "scalex"), 32)
		z, zErr := strconv.ParseFloat(first(section, "scalez"), 32)
		if idErr != nil || xErr != nil || zErr != nil || x <= 0 || z <= 0 {
			return nil, fmt.Errorf("DungeonMiniMaps.INI section %q is invalid", section.Name)
		}
		if _, exists := result[uint16(id)]; exists {
			return nil, fmt.Errorf("DungeonMiniMaps.INI duplicate map %d", id)
		}
		result[uint16(id)] = DungeonMinimap{float32(x), float32(z)}
	}
	if len(result) == 0 || len(result) > 4096 {
		return nil, fmt.Errorf("DungeonMiniMaps.INI has %d maps", len(result))
	}
	return result, nil
}
