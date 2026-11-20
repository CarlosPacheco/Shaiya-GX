package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

const terrainBatchCells = 4

type terrainBatchCPU struct {
	vertices []render.Vertex
	indices  []uint32
	cells    [terrainBatchCells * terrainBatchCells]render.TerrainRange
}

// A batch covers the same fixed section and spatial block throughout its
// lifetime. Its inactive cells are never submitted, so adjoining residency
// changes only the indirect range list.
func buildTerrainBatch(w *World, key terrainBatchKey) *terrainBatchCPU {
	out := &terrainBatchCPU{}
	sections := w.sectionsPerSide()
	section := (w.side - 1) / sections
	sx, sz := key.section%sections, key.section/sections
	for dz := range terrainBatchCells {
		for dx := range terrainBatchCells {
			x := (key.x*terrainBatchCells + dx) * terrainCellSamples
			z := (key.z*terrainBatchCells + dz) * terrainCellSamples
			ax, az := max(x, sx*section), max(z, sz*section)
			bx, bz := min(x+terrainCellSamples, (sx+1)*section), min(z+terrainCellSamples, (sz+1)*section)
			if ax >= bx || az >= bz {
				continue
			}
			first := uint32(len(out.indices))
			out.vertices, out.indices = appendTerrain(out.vertices, out.indices, w, ax, az, bx-ax, bz-az, sx, sz)
			out.cells[dz*terrainBatchCells+dx] = render.TerrainRange{First: first, Count: uint32(len(out.indices)) - first}
		}
	}
	return out
}

func (s *fieldStream) rebuildTerrainBatches(dirty map[terrainBatchKey]bool) {
	for group := range dirty {
		batch := s.terrainBatches[group]
		resident := false
		for dz := range terrainBatchCells {
			for dx := range terrainBatchCells {
				cell := s.cells[cellKey{group.x*terrainBatchCells + dx, group.z*terrainBatchCells + dz}]
				if cell == nil {
					continue
				}
				for i := range cell.parts {
					p := &cell.parts[i]
					if p.batch != group {
						continue
					}
					resident = true
					if batch == nil {
						var cpu *terrainBatchCPU
						if s.loader != nil {
							state := s.loader.states[terrainAssetKey(group)]
							if state != nil && state.result != nil {
								cpu, state.result = state.result.terrain, nil
							}
						}
						if cpu == nil {
							cpu = buildTerrainBatch(s.world, group)
						}
						batch = &terrainBatch{mesh: render.NewMesh(cpu.vertices, cpu.indices), material: p.terrain, cells: cpu.cells}
						s.terrainBatches[group] = batch
					}
					p.rangeInBatch = batch.cells[dz*terrainBatchCells+dx]
				}
			}
		}
		if batch != nil && !resident {
			batch.mesh.Delete()
			delete(s.terrainBatches, group)
			if s.loader != nil {
				delete(s.loader.states, terrainAssetKey(group))
			}
		}
	}
}

func drawTerrainBatch(s *render.Scene3D, b *terrainBatch) {
	if b == nil || len(b.ranges) == 0 {
		return
	}
	s.SetTerrainRanges(b.ranges)
	drawTerrainSplats(s, b.mesh, b.material, len(b.ranges))
}
