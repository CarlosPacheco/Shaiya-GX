package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// relicNoticeMessage follows Game.exe 0x562240 for 0x2103 (owners 0
// neutral, 1 light, 2 fury; faction 0 light, 1 fury): 463 "Destoyed <p>."
// when the reader's country took the relic, 462 "<p> is destroyed!" when
// it lost it; a neutral relic taken by the other country says nothing.
func relicNoticeMessage(oldOwner, newOwner, faction uint8) (int, bool) {
	switch {
	case oldOwner == 0 && newOwner == 1 && faction == 0,
		oldOwner == 0 && newOwner == 2 && faction == 1,
		oldOwner == 1 && newOwner == 2 && faction == 1,
		oldOwner == 2 && newOwner == 1 && faction == 0:
		return 463, true
	case oldOwner == 1 && newOwner == 2 && faction == 0,
		oldOwner == 2 && newOwner == 1 && faction == 1:
		return 462, true
	}
	return 0, false
}

// Native callback 0x562240 uses central type 8 on maps 0/18/30/70,
// and history type 7 elsewhere.
func relicNoticeStyle(mapID uint16) uint8 {
	switch mapID {
	case 0, 18, 30, 70:
		return 8
	}
	return 7
}

// relicMapMarkers adds the current map's relics, under the other markers.
func relicMapMarkers(s *game.State, mapSize float32, markers []ui.MapMarker) []ui.MapMarker {
	if s.MapID == 68 || s.MapID == 69 {
		return markers
	}
	for _, r := range s.Relics {
		kind, ok := ui.RelicMarkerKind(r.Owner)
		if r.Owner <= 2 && (s.MapID == 105 || s.MapID == 108) {
			continue
		}
		if r.Owner >= 100 {
			ok = r.Owner >= 101 && r.Owner <= 103
			if s.MapID >= 100 && s.MapID <= 120 {
				ok = false
			}
			kind = ui.MapRelicPortal
		}
		if !ok {
			continue
		}
		if u, v, ok := ui.MapPosition(r.X, r.Z, mapSize); ok {
			markers = append(markers, ui.MapMarker{U: u, V: v, Kind: kind})
		}
	}
	return markers
}
