package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// GeometryReduction zero is the unchanged default. Lower optional presets trim only
// decorative vegetation, never buildings, terrain, actors or collision data.
func (e *fieldDrawEntry) detailVisible(s *render.Scene3D, center render.V3, radius float32) bool {
	if s.GeometryReduction < 1 || s.Shadow || e.object.detailClass == 0 {
		return true
	}
	v := s.View
	p := render.V3{center[0]*v[0] + center[1]*v[4] + center[2]*v[8] + v[12], center[0]*v[1] + center[1]*v[5] + center[2]*v[9] + v[13], center[0]*v[2] + center[1]*v[6] + center[2]*v[10] + v[14]}
	distance := max(float32(0), p.Len()-radius)
	if e.object.detailClass == 2 {
		if s.GeometryReduction == 2 {
			return radius > 32 || distance <= 55
		}
		return radius > 32 || distance <= 135
	}
	if s.GeometryReduction == 2 {
		return false
	}
	// Keep nearby grass and a stable quarter of the remaining placements.
	return distance <= 12 || distance <= 50 && e.object.detailRank == 0
}
