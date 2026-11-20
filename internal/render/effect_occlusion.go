package render

// CullEffectInstances compacts one emitter's instance arrays in order. It
// only reads the camera mask so that workers can use it during preparation.
func (o *TerrainOcclusion) CullEffectInstances(m *Mesh, worlds []M4, colors [][4]float32) int {
	if o == nil || m == nil || !m.boundsValid || len(worlds) == 0 {
		return len(worlds)
	}
	if len(worlds) >= 8 && o.effectEmitterHidden(m, worlds) {
		return 0
	}
	n := 0
	localCenter := m.minimum.Add(m.maximum).Scale(.5)
	for i, world := range worlds {
		if o.PointMayBeHidden(effectBoundsCenter(world, localCenter)) {
			lo, hi := effectWorldBounds(m, world)
			if o.BoxHidden(lo, hi) {
				continue
			}
		}
		if n != i {
			worlds[n], colors[n] = world, colors[i]
		}
		n++
	}
	return n
}

func (o *TerrainOcclusion) effectEmitterHidden(m *Mesh, worlds []M4) bool {
	localCenter := m.minimum.Add(m.maximum).Scale(.5)
	// A single point that the mask cannot hide also rules out hiding its
	// enclosing box. Open terrain skips the expensive bound projection.
	for _, world := range worlds {
		if !o.PointMayBeHidden(effectBoundsCenter(world, localCenter)) {
			return false
		}
	}
	lo, hi := effectWorldBounds(m, worlds[0])
	for _, world := range worlds[1:] {
		p, q := effectWorldBounds(m, world)
		for axis := range 3 {
			lo[axis], hi[axis] = min(lo[axis], p[axis]), max(hi[axis], q[axis])
		}
	}
	return o.BoxHidden(lo, hi)
}

func effectBoundsCenter(world M4, center V3) V3 {
	if center == (V3{}) {
		return V3(world[12:15])
	}
	var p V3
	for i := range p {
		p[i] = center[0]*world[i] + center[1]*world[4+i] + center[2]*world[8+i] + world[12+i]
	}
	return p
}

// Select each output axis's extremal corners without transforming all eight.
func effectWorldBounds(m *Mesh, world M4) (lo, hi V3) {
	for i := range lo {
		a, b := m.minimum[0]*world[i], m.maximum[0]*world[i]
		c, d := m.minimum[1]*world[4+i], m.maximum[1]*world[4+i]
		e, f := m.minimum[2]*world[8+i], m.maximum[2]*world[8+i]
		lo[i] = min(a, b) + min(c, d) + min(e, f) + world[12+i]
		hi[i] = max(a, b) + max(c, d) + max(e, f) + world[12+i]
	}
	return lo, hi
}
