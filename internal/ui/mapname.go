package ui

import (
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
)

const (
	firstMapNameMessage = 11001
	mapNameCount        = 200
)

// MapName reads the loose SysMsg row for a map, including map zero.
func MapName(messages text.SysMsg, id uint16) string {
	if id < mapNameCount {
		if name := messages.Get(firstMapNameMessage + int(id)); name != "" {
			return name
		}
	}
	return "Map " + strconv.Itoa(int(id))
}
