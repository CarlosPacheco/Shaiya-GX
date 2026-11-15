package app

import (
	"encoding/json"
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type dungeonMinimapData struct {
	loaded bool
	scales map[uint16]world.DungeonMinimap
	sizes  map[uint16]float32
}

func (a *App) loadDungeonMinimaps() {
	if a.dungeonMinimaps.loaded || a.arc == nil {
		return
	}
	a.dungeonMinimaps.loaded = true
	if _, ok := a.arc.Lookup("Dungeon_MiniMap.cfg"); ok {
		raw, err := a.arc.ReadFile("Dungeon_MiniMap.cfg")
		if err == nil {
			a.dungeonMinimaps.scales, err = world.ParseDungeonMinimap(raw)
		}
		if err != nil {
			log.Printf("dungeon minimap calibration: %v", err)
		}
	}
	if _, ok := a.arc.Lookup("interface/MiniMap/map-sizes.json"); ok {
		raw, err := a.arc.ReadFile("interface/MiniMap/map-sizes.json")
		if err == nil {
			err = json.Unmarshal(raw, &a.dungeonMinimaps.sizes)
		}
		if err != nil {
			log.Printf("minimap coordinate sizes: %v", err)
		}
	}
}
func (a *App) mapProjection(v *mapView) ui.MapProjection {
	if a.wv == nil || a.wv.world == nil || a.wv.world.WLD == nil {
		return ui.MapProjection{}
	}
	layout := a.wv.world.WLD
	if !layout.Dungeon {
		return ui.MapProjection{Width: float32(layout.MapSize), Height: float32(layout.MapSize)}
	}
	a.loadDungeonMinimaps()
	scale, ok := a.dungeonMinimaps.scales[a.wv.mapID]
	if !ok || v.image == nil {
		return ui.MapProjection{}
	}
	return ui.MapProjection{Width: float32(v.image.W) / scale.X, Height: float32(v.image.H) / scale.Z}
}
func (a *App) minimapWorldSize(mapID uint16) float32 {
	if a.wv != nil && a.wv.mapID == mapID && a.wv.world != nil && a.wv.world.WLD != nil && a.wv.world.WLD.MapSize > 0 {
		return float32(a.wv.world.WLD.MapSize)
	}
	a.loadDungeonMinimaps()
	return a.dungeonMinimaps.sizes[mapID]
}

// Marker producers retain their shared square-coordinate API and cache. Use a
// containing square, then calibrate the final copied list on both image axes.
func calibrateMapMarkers(markers []ui.MapMarker, p ui.MapProjection, square float32) []ui.MapMarker {
	if p.Width == square && p.Height == square {
		return markers
	}
	result := markers[:0]
	for _, m := range markers {
		u, v, ok := p.Position(m.U*square, (1-m.V)*square)
		if ok {
			m.U, m.V = u, v
			result = append(result, m)
		}
	}
	return result
}
