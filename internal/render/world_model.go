package render

import "github.com/go-gl/gl/v4.5-core/gl"

// Only proven coincident front/back sheets are culled. Ordinary unpaired
// sheets stay two-sided, including cutout vegetation and thin decorations.
func beginWorldFaces(m *Mesh) bool {
	if m == nil || !m.CullBack {
		gl.Disable(gl.CULL_FACE)
		gl.FrontFace(gl.CCW)
		return false
	}
	gl.Enable(gl.CULL_FACE)
	gl.FrontFace(gl.CW)
	gl.CullFace(gl.BACK)
	return true
}
func endWorldFaces(culled bool) {
	if culled {
		gl.Disable(gl.CULL_FACE)
		gl.FrontFace(gl.CCW)
	}
}

func (s *Scene3D) DrawWorldModel(m *Mesh, t *Texture, world M4, alphaRef float32) {
	culled := beginWorldFaces(m)
	s.DrawModel(m, t, world, alphaRef, false)
	endWorldFaces(culled)
}

// DrawVisibleWorldModel accepts geometry already tested against its mesh bounds.
func (s *Scene3D) DrawVisibleWorldModel(m *Mesh, t *Texture, world M4, alphaRef float32) {
	culled := beginWorldFaces(m)
	s.drawVisibleModel(m, t, world, alphaRef, false, true)
	endWorldFaces(culled)
}

func (s *Scene3D) WorldMeshHidden(m *Mesh, world M4) bool { return s.meshOccluded(m, world) }

func (s *Scene3D) DrawWorldInstanced(m *Mesh, t *Texture, alphaRef float32) {
	culled := beginWorldFaces(m)
	s.DrawInstanced(m, t, alphaRef)
	endWorldFaces(culled)
}
