//go:build integration

package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

func drawTerrainCell(s *render.Scene3D, b *terrainBatch, cell render.TerrainRange) {
	if b == nil {
		return
	}
	first := true
	for slot, layer := range b.material.Layers {
		if layer.Mask == nil || layer.Tile == nil {
			continue
		}
		s.DrawTerrainRange(b.mesh, render.TerrainSplat{Mask: layer.Mask, Tile: layer.Tile, TileSize: layer.TileSize, LayerBit: uint32(byte(1) << slot)}, b.material.Lightmap, b.material.LayerBlocks, first, cell)
		first = false
	}
}
