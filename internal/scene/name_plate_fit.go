package scene

import (
	"sort"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Name plates keep the original anchor (MON Height plus its constant), but the
// MON Height is authored loosely: for some models it ends inside the body, for
// others it floats far above it. FitPlate measures the body once (the 99.5th
// percentile of skinned vertex height, so wings, horns and weapons do not set
// the height) and keeps the plate within [top+low, top+high] of it.
const (
	plateBodyPercentile = .995
	plateHighest        = .6 // never farther above the body than this
)

// bodyTop is the cached body height above the actor's origin.
func (m *CreatureActor) bodyTop() (float32, bool) {
	if m == nil || m.Character == nil {
		return 0, false
	}
	if m.plateTopReady {
		return m.plateTop, m.plateTopValid
	}
	if _, _, ok := m.WorldBounds(); !ok { // prepares the pose of every part
		return 0, false
	}
	var heights []float32
	for _, p := range m.parts {
		if p == nil || len(p.vertices) == 0 || len(p.pose) == 0 {
			continue
		}
		for _, v := range p.vertices {
			bone := int(v.Bone0)
			if v.Weight < .5 {
				bone = int(v.Bone1)
			}
			if bone < 0 || bone >= len(p.pose) {
				continue
			}
			heights = append(heights, transformPoint(v.Pos, p.pose[bone].MulAffine(m.World))[1]-m.World[13])
		}
	}
	m.plateTopReady = true
	if len(heights) < 8 {
		return 0, false
	}
	sort.Slice(heights, func(i, j int) bool { return heights[i] < heights[j] })
	m.plateTop, m.plateTopValid = heights[int(float64(len(heights)-1)*plateBodyPercentile)], true
	return m.plateTop, true
}

// FitPlate returns base moved, if needed, into the band above the body. low is
// the smallest gap between the body and the plate.
func (m *CreatureActor) FitPlate(base render.V3, low float32) render.V3 {
	top, ok := m.bodyTop()
	if !ok {
		return base
	}
	relative := base[1] - m.World[13]
	base[1] = m.World[13] + min(max(relative, top+low), top+plateHighest)
	return base
}
