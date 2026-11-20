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

// BeginTranslucentWorldModels uses the WLD blended pass state, with depth
// testing against the opaque scene and no alpha test or depth writes.
func (s *Scene3D) BeginTranslucentWorldModels() {
	s.Begin()
	gl.Uniform1i(s.uMode, 0)
	gl.Uniform1f(s.uRef, -1)
	s.blend(true)
	gl.DepthMask(false)
}

func (s *Scene3D) DrawTranslucentWorldModel(m *Mesh, t *Texture, worlds []M4, lit bool) {
	if m == nil || m.Count == 0 || len(worlds) == 0 {
		return
	}
	lighting := int32(0)
	if lit {
		lighting = 1
	}
	gl.Uniform1i(s.uLit, lighting)
	bindDiffuse(t)
	if len(worlds) == 1 {
		s.setWorld(worlds[0])
		m.Draw()
		return
	}
	s.SetInstances(worlds)
	gl.Uniform1i(s.uInstanced, 1)
	s.drawInstances(m)
}

func (s *Scene3D) EndTranslucentWorldModels() {
	gl.DepthMask(true)
	s.blend(false)
}
