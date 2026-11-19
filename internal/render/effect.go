package render

import "github.com/go-gl/gl/v4.5-core/gl"

type effectPassState struct {
	active, repeat bool
	src, dst       uint32
	texture        *Texture
}

// BeginEffects establishes state common to consecutive transparent EFT draws.
// Material changes keep their original order; no depth is written.
func (s *Scene3D) BeginEffects() {
	s.Begin()
	s.effectPass = effectPassState{active: true, src: ^uint32(0), dst: ^uint32(0)}
	gl.Uniform1i(s.uMode, 3)
	gl.Uniform1i(s.uLit, 0)
	gl.Uniform2f(s.uUVOff, 0, 0)
	gl.Enable(gl.BLEND)
	gl.DepthMask(false)
}

func (s *Scene3D) EndEffects() {
	s.flushEffectQueue()
	if s.effectPass.texture != nil && s.effectPass.repeat {
		s.effectPass.texture.SetWrap(false)
	}
	s.effectPass = effectPassState{}
	s.End()
}

func (s *Scene3D) effectMaterial(t *Texture, src, dst uint32, repeat bool) {
	p := &s.effectPass
	if !p.active {
		gl.Uniform1i(s.uMode, 3)
		gl.Uniform1i(s.uLit, 0)
		gl.Uniform2f(s.uUVOff, 0, 0)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(src, dst)
		gl.DepthMask(false)
		t.SetWrap(repeat)
		gl.BindTextureUnit(0, t.ID)
		return
	}
	if p.src != src || p.dst != dst {
		gl.BlendFunc(src, dst)
		p.src, p.dst = src, dst
	}
	if p.texture == t && p.repeat == repeat {
		return
	}
	if p.texture != t {
		if p.texture != nil && p.repeat {
			p.texture.SetWrap(false)
		}
		gl.BindTextureUnit(0, t.ID)
	}
	t.SetWrap(repeat)
	p.texture, p.repeat = t, repeat
}

// DrawEffectBatch preserves particle order within one emitter while sharing
// geometry, material and draw call; each instance has its own tint.
func (s *Scene3D) DrawEffectBatch(m *Mesh, t *Texture, worlds []M4, colors [][4]float32, source, destination int32, clampTexture ...bool) {
	src, dst, ok := EFTBlend(source, destination)
	if !ok || t == nil || m == nil || m.Count == 0 || len(worlds) == 0 || len(worlds) != len(colors) {
		return
	}
	worlds, colors = s.visibleEffectInstances(m, worlds, colors)
	if len(worlds) == 0 {
		return
	}
	repeat := len(clampTexture) == 0 || !clampTexture[0]
	if s.queueEffect(m, t, worlds, colors, nil, 0, src, dst, repeat) {
		return
	}
	s.flushEffectQueue()
	s.uploadInstances(worlds, colors)
	gl.Uniform1i(s.uInstanced, 1)
	gl.Uniform4f(s.uTint, 1, 1, 1, 1)
	s.effectMaterial(t, src, dst, repeat)
	s.drawInstances(m)
	if !s.effectPass.active {
		t.SetWrap(false)
	}
	gl.Uniform1i(s.uInstanceColorOffset, 0)
}

// DrawGroundItems uses the native unlit texture-factor colour while retaining
// scene fog and depth. Independent factors share a single instanced draw.
func (s *Scene3D) DrawGroundItems(m *Mesh, t *Texture, worlds []M4, colors [][4]float32) {
	if m == nil || m.Count == 0 || t == nil || len(worlds) == 0 || len(worlds) != len(colors) {
		return
	}
	worlds, colors = s.visibleEffectInstances(m, worlds, colors)
	if len(worlds) == 0 {
		return
	}
	s.uploadInstances(worlds, colors)
	gl.Uniform1i(s.uInstanced, 1)
	gl.Uniform1i(s.uMode, 8)
	gl.Uniform1i(s.uLit, 0)
	gl.Uniform1f(s.uRef, .5)
	s.blend(false)
	gl.DepthMask(true)
	bindDiffuse(t)
	s.drawInstances(m)
	gl.Uniform1i(s.uInstanceColorOffset, 0)
}

// DrawEffectMesh shares one GPU stream across every 3DE particle and cast.
func (s *Scene3D) DrawEffectMesh(vertices []Vertex, indices []uint32, t *Texture, world M4, color [4]float32, source, destination int32) {
	s.flushEffectQueue()
	if s.effectStream == nil {
		s.effectStream = NewStreamMesh()
	}
	s.effectStream.Stream(vertices, indices)
	s.DrawEffect(s.effectStream, t, world, color, source, destination)
}

// DrawEffectMeshBatch shares one sampled 3DE pose among consecutive
// particles, preserving their order and individual transforms and colors.
func (s *Scene3D) DrawEffectMeshBatch(vertices []Vertex, indices []uint32, t *Texture, worlds []M4, colors [][4]float32, source, destination int32) {
	if len(worlds) == 0 || len(worlds) != len(colors) {
		return
	}
	s.flushEffectQueue()
	if s.effectStream == nil {
		s.effectStream = NewStreamMesh()
	}
	s.effectStream.Stream(vertices, indices)
	if len(worlds) == 1 {
		s.DrawEffect(s.effectStream, t, worlds[0], colors[0], source, destination)
	} else {
		s.DrawEffectBatch(s.effectStream, t, worlds, colors, source, destination)
	}
}

// EFTBlend follows the authored catalog at Game.exe 0x6ccc34, not D3D enums.
func EFTBlend(source, destination int32) (uint32, uint32, bool) {
	f := [...]uint32{gl.ZERO, gl.ONE, gl.SRC_COLOR, gl.ONE_MINUS_SRC_COLOR, gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA, gl.DST_ALPHA, gl.ONE_MINUS_DST_ALPHA, gl.DST_COLOR, gl.ONE_MINUS_DST_COLOR, gl.SRC_ALPHA_SATURATE, gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA}
	if source < 0 || source >= int32(len(f)) || destination < 0 || destination >= int32(len(f)) {
		return 0, 0, false
	}
	s, d := f[source], f[destination]
	if source == 11 {
		d = gl.ONE_MINUS_SRC_ALPHA
	} else if source == 12 {
		d = gl.SRC_ALPHA
	}
	return s, d, true
}

// DrawEffect renders texture * vertex-equivalent tint, depth tested without
// depth writes (0x55cb10, native_effect_ports.cpp and native_controller_state.cpp).
func (s *Scene3D) DrawEffect(m *Mesh, t *Texture, world M4, color [4]float32, source, destination int32, clampTexture ...bool) {
	src, dst, ok := EFTBlend(source, destination)
	if !ok || t == nil || s.meshOccluded(m, world) {
		return
	}
	repeat := len(clampTexture) == 0 || !clampTexture[0]
	if s.effectPass.active {
		if m == nil || m.Count == 0 || s.queueEffect(m, t, []M4{world}, [][4]float32{color}, nil, 0, src, dst, repeat) {
			return
		}
	}
	s.flushEffectQueue()
	s.setWorld(world)
	gl.Uniform4fv(s.uTint, 1, &color[0])
	s.effectMaterial(t, src, dst, repeat)
	m.Draw()
	if !s.effectPass.active {
		t.SetWrap(false)
	}
}
