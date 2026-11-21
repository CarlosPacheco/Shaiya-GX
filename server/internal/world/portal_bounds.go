package world

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type portalVolume struct {
	Target   uint16     `json:"target"`
	Min      [3]float32 `json:"min"`
	Max      [3]float32 `json:"max"`
	ArrivalY *float32   `json:"arrival_y,omitempty"`
}

type portalVolumeCatalog map[uint16]map[uint8]portalVolume

var portalVolumes portalVolumeCatalog

func readPortalVolumes(path string) (portalVolumeCatalog, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var catalog portalVolumeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, err
	}
	for mapID, byIndex := range catalog {
		for index, volume := range byIndex {
			for axis := range volume.Min {
				if volume.Min[axis] >= volume.Max[axis] {
					return nil, fmt.Errorf("invalid portal bounds for map %d portal %d", mapID, index)
				}
			}
		}
	}
	return catalog, nil
}

func loadPortalVolumes() error {
	catalog, err := readPortalVolumes(filepath.Join(gameDataDir(), "PortalBounds.json"))
	if err != nil {
		return err
	}
	portalVolumes = catalog
	return nil
}

func (catalog portalVolumeCatalog) apply(mapID uint16, portals []MapPortal) {
	for i := range portals {
		p := &portals[i]
		v, ok := catalog[mapID][p.Index]
		if !ok || v.Target != p.TargetMapID || !v.contains(p.PosX, p.PosY, p.PosZ, .01) {
			continue
		}
		p.volume = &v
		if v.ArrivalY != nil {
			p.TargetY = *v.ArrivalY
		}
	}
}

func (v *portalVolume) contains(x, y, z, tolerance float32) bool {
	for axis, value := range [3]float32{x, y, z} {
		if !(value >= v.Min[axis]-tolerance && value <= v.Max[axis]+tolerance) {
			return false
		}
	}
	return true
}
