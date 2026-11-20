package scene

import (
	"strconv"
	"strings"
)

func terrainAssetKey(k terrainBatchKey) fieldAssetKey {
	return fieldAssetKey{fieldTerrain, strconv.Itoa(k.x) + ":" + strconv.Itoa(k.z) + ":" + strconv.Itoa(k.section)}
}

// Replanning preserves wanted decode jobs and their dependency textures. GPU
// residency remains confined to the GL thread and to the new camera region.
func (s *fieldStream) replanAssets() {
	l := s.loader
	l.generation++
	oldPins := l.pinned
	l.pinned = make(map[fieldAssetKey]bool)
	keep := func(id int) {
		o := s.objects[id]
		kind := fieldSMOD
		if o.vani {
			kind = fieldVANI
		}
		key := fieldAssetKey{kind, strings.ToLower(o.file)}
		if l.states[key] != nil || oldPins[key] {
			l.pinned[key] = true
		}
	}
	for _, id := range s.global {
		keep(id)
	}
	terrain := make(map[fieldAssetKey]bool)
	n := s.world.sectionsPerSide()
	step := (s.world.side - 1) / n
	for _, cell := range s.order {
		for _, id := range s.index[cell] {
			keep(id)
		}
		x, z := cell.x*terrainCellSamples, cell.z*terrainCellSamples
		if x < 0 || z < 0 || x >= s.world.side-1 || z >= s.world.side-1 {
			continue
		}
		xe, ze := min(x+terrainCellSamples, s.world.side-1)-1, min(z+terrainCellSamples, s.world.side-1)-1
		for sz := z / step; sz <= ze/step; sz++ {
			for sx := x / step; sx <= xe/step; sx++ {
				k := terrainBatchKey{cell.x / terrainBatchCells, cell.z / terrainBatchCells, sz*n + sx}
				terrain[terrainAssetKey(k)] = true
			}
		}
	}
	for key := range l.pinned {
		state := l.states[key]
		if state == nil || state.result == nil {
			continue
		}
		result := state.result
		if result.smod != nil {
			for _, mesh := range result.smod.Meshes {
				if mesh.Texture != "" {
					l.pinned[fieldAssetKey{fieldTexture, strings.ToLower(modelTextureFile(mesh.Texture))}] = true
				}
			}
		}
		if result.vani != nil && s.assets.byStem != nil {
			for _, mesh := range result.vani.Meshes {
				if path, ok := s.assets.findTextureFile(mesh.Texture); ok {
					l.pinned[fieldAssetKey{fieldTexture, strings.ToLower(path)}] = true
				}
			}
		}
	}
	for key, state := range l.states {
		switch key.kind {
		case fieldSMOD, fieldVANI:
			if l.pinned[key] {
				continue
			}
			if state.result != nil {
				l.decodedModels--
			}
		case fieldTerrain:
			if terrain[key] {
				continue
			}
		case fieldTexture:
			if l.pinned[key] {
				continue
			}
			if state.result == nil && !s.assetCached(key) {
				// A bounded in-flight texture can still be a wanted section dependency.
				l.pinned[key] = true
				continue
			}
		case fieldTextureIndex:
			continue
		}
		delete(l.states, key)
	}
}
