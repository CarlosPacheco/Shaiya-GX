package world

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// DungeonMinimap contains native pixels per world unit. ImgPos_x/y are fixed
// point scales (1000 = one pixel/unit), not image offsets or world extents.
type DungeonMinimap struct{ X, Z float32 }

func ParseDungeonMinimap(raw []byte) (map[uint16]DungeonMinimap, error) {
	sections := map[string]map[string]int{}
	section := ""
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\xef\xbb\xbf"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			if sections[section] != nil {
				return nil, fmt.Errorf("Dungeon_MiniMap.cfg duplicate section %s", section)
			}
			sections[section] = map[string]int{}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || section == "" {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("Dungeon_MiniMap.cfg %s/%s: %w", section, key, err)
		}
		if _, exists := sections[section][key]; exists {
			return nil, fmt.Errorf("Dungeon_MiniMap.cfg duplicate key %s/%s", section, key)
		}
		sections[section][key] = n
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	count := sections["worldmap"]["count"]
	if count < 1 || count > 4096 {
		return nil, fmt.Errorf("Dungeon_MiniMap.cfg invalid count %d", count)
	}
	result := make(map[uint16]DungeonMinimap, count)
	for n := range count {
		row := sections[strconv.Itoa(n)]
		id, exists := row["mapnum"]
		if !exists || id < 0 || id > 65535 || row["imgpos_x"] <= 0 || row["imgpos_y"] <= 0 {
			return nil, fmt.Errorf("Dungeon_MiniMap.cfg invalid row %d", n)
		}
		if _, exists := result[uint16(id)]; exists {
			return nil, fmt.Errorf("Dungeon_MiniMap.cfg duplicate map %d", id)
		}
		result[uint16(id)] = DungeonMinimap{float32(row["imgpos_x"]) / 1000, float32(row["imgpos_y"]) / 1000}
	}
	return result, nil
}
