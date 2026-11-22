package world

// TerrainLayerBlocks returns the layer bits selected by each 32-unit FLD
// block, ordered X then Z. Game.exe 0x5b4aa0 scans alternate two-unit samples
// in each block and up to four samples into adjacent blocks. Both the local
// and transition flags allow an overlay to read its authored alpha mask.
func (w *WLD) TerrainLayerBlocks() (side int, layers []byte) {
	const blockSize = 32
	const samplesPerBlock = blockSize / 2
	const margin = 4
	samples := int(w.MapSize) / 2
	stride := samples + 1
	if w.MapSize == 0 || w.MapSize%blockSize != 0 || len(w.TextureMap) != stride*stride {
		return 0, nil
	}
	side = int(w.MapSize) / blockSize
	layers = make([]byte, side*side)
	for z := range side {
		for x := range side {
			x0, x1 := max(0, x*samplesPerBlock-margin), min(samples, (x+1)*samplesPerBlock+margin)
			z0, z1 := max(0, z*samplesPerBlock-margin), min(samples, (z+1)*samplesPerBlock+margin)
			var enabled byte
			for sz := z0; sz < z1; sz += 2 {
				for sx := x0; sx < x1; sx += 2 {
					enabled |= byte(1) << w.TextureMap[sz*stride+sx] // IDs outside 0–7 select no layer.
				}
			}
			layers[z*side+x] = enabled
		}
	}
	return side, layers
}
