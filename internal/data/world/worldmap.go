package world

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/ini"
)

// WorldMapRegion is one authored WorldMap.INI region. Coordinates are relative
// to the native 512x512 wm_map image; Country 0/1/2 means both/Light/Fury.
type WorldMapRegion struct {
	Country, Map             int
	Left, Top, Right, Bottom int
	ImageX, ImageY           int
}

// ParseWorldMap reads WorldMap.INI: one section per region, in display order.
//
//	[Williars]
//	Country=Light        Both, Light or Fury
//	Map=19               connected map, -1 for none
//	Click=180,160,234,200   left, top, right, bottom of the hover/click box
//	Image=172,149           top-left of the overlay image
func ParseWorldMap(raw []byte) ([]WorldMapRegion, error) {
	var out []WorldMapRegion
	for _, section := range ini.Parse(raw) {
		if section.Name == "" {
			continue
		}
		country, ok := worldMapCountries[strings.ToLower(first(section, "country"))]
		mapNum, err := strconv.Atoi(first(section, "map"))
		click, clickOK := numbers(first(section, "click"), 4)
		image, imageOK := numbers(first(section, "image"), 2)
		if !ok || err != nil || !clickOK || !imageOK {
			return nil, fmt.Errorf("WorldMap.INI region %q is incomplete or invalid", section.Name)
		}
		r := WorldMapRegion{country, mapNum, click[0], click[1], click[2], click[3], image[0], image[1]}
		if r.Map < -1 || r.Right < r.Left || r.Bottom < r.Top {
			return nil, fmt.Errorf("WorldMap.INI region %q is invalid", section.Name)
		}
		out = append(out, r)
	}
	if len(out) == 0 || len(out) > 4096 {
		return nil, fmt.Errorf("WorldMap.INI has %d regions", len(out))
	}
	return out, nil
}

var worldMapCountries = map[string]int{"both": 0, "light": 1, "fury": 2}

func first(s ini.Section, key string) string {
	v, _ := s.Get(key)
	return v
}

// numbers reads exactly n comma-separated integers.
func numbers(text string, n int) ([]int, bool) {
	fields := strings.Split(text, ",")
	if len(fields) != n {
		return nil, false
	}
	out := make([]int, n)
	for i, f := range fields {
		v, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}
func (r WorldMapRegion) Contains(x, y float32) bool {
	return x >= float32(r.Left) && x <= float32(r.Right) && y >= float32(r.Top) && y <= float32(r.Bottom)
}
func (r WorldMapRegion) Available(fury bool) bool {
	return r.Country == 0 || r.Country == 1 && !fury || r.Country == 2 && fury
}
