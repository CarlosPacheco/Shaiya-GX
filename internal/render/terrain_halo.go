package render

import (
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

type terrainHalo struct {
	texture   *Texture
	center    V3
	projector M4
	program   uint32
	tried     bool
}

// SetTerrainHalo projects the original light texture from the fixed EP5.4
// offset (0x549130). The input position is at the character's feet.
func (s *Scene3D) SetTerrainHalo(texture *Texture, feet V3) {
	h := &s.terrainHalo
	h.texture = texture
	if texture == nil {
		return
	}
	if !h.tried {
		h.tried = true
		var err error
		h.program, err = Program("personal terrain light", terrainHaloVS, terrainHaloFS)
		if err != nil {
			log.Printf("personal terrain light: %v", err)
		}
	}
	h.center = feet
	view := LookAtLH(feet.Add(V3{20, 50, 20}), feet, V3{0, 1, 0})
	h.projector = view.Mul(PerspectiveFovLH(math.Pi/4, 1.25, .1, 400))
}

func (s *Scene3D) HasTerrainHalo() bool {
	return s.terrainHalo.texture != nil && s.terrainHalo.program != 0
}

const terrainHaloVS = `#version 450 core
layout(location=0) in vec3 pos;
layout(location=0) uniform mat4 viewProj;
layout(location=4) uniform mat4 projector;
layout(location=8) uniform mat4 view;
layout(location=12) uniform vec4 clipPlane;
out vec4 projected;
out vec2 groundXZ;
out float depth;
void main() {
 vec4 p = vec4(pos,1);
 gl_Position = viewProj*p;
 gl_ClipDistance[0] = dot(p,clipPlane);
 projected = projector*p;
 groundXZ = pos.xz;
 depth = (view*p).z;
}`

const terrainHaloFS = `#version 450 core
layout(binding=0) uniform sampler2D image;
layout(location=13) uniform vec4 area;
layout(location=14) uniform vec2 fogRange;
in vec4 projected;
in vec2 groundXZ;
in float depth;
out vec4 frag;
void main() {
 if (any(greaterThan(abs(groundXZ-area.xy),area.zw)) || projected.w <= 0.0) discard;
 // Native texture transform scales projected XY by (0.98/0.05) before
 // the D3D scale and bias from clip space to top-down texture coordinates.
 vec2 uv = projected.xy/projected.w * vec2(9.8,-9.8) + .5;
 if (any(lessThan(uv,vec2(0))) || any(greaterThan(uv,vec2(1)))) discard;
 frag = texture(image,uv);
 if (fogRange.y > fogRange.x)
  frag.a *= clamp((fogRange.y-depth)/(fogRange.y-fogRange.x),0.0,1.0);
}`

// Redraw resident terrain cells with the same depth as the base terrain pass.
// The small receiver bounds keep distant slopes out of the projection.
func (s *Scene3D) drawTerrainHalo(m *Mesh, ranges int) {
	h := &s.terrainHalo
	if !s.HasTerrainHalo() || s.Shadow {
		return
	}
	const reach = 16
	x, z := h.center[0], h.center[2]
	if m.boundsValid && (m.maximum[0] < x-reach || m.minimum[0] > x+reach || m.maximum[2] < z-reach || m.minimum[2] > z+reach) {
		return
	}
	gl.UseProgram(h.program)
	view := s.modelView(Identity())
	vp := view.Mul(s.projection)
	gl.UniformMatrix4fv(0, 1, false, &vp[0])
	gl.UniformMatrix4fv(4, 1, false, &h.projector[0])
	gl.UniformMatrix4fv(8, 1, false, &view[0])
	gl.Uniform4fv(12, 1, &s.ClipPlane[0])
	gl.Uniform4f(13, x, z, reach, reach)
	gl.Uniform2f(14, s.FogStart, s.FogEnd)
	gl.BindTextureUnit(0, h.texture.ID)
	gl.DepthMask(false)
	gl.DepthFunc(gl.EQUAL)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
	if ranges == 0 {
		m.Draw()
	} else {
		gl.BindVertexArray(m.vao)
		gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, s.terrainIndirect)
		gl.MultiDrawElementsIndirect(gl.TRIANGLES, m.indexType, nil, int32(ranges), 0)
		gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, 0)
		drawCalls++
	}
	gl.Disable(gl.BLEND)
	gl.DepthMask(true)
	s.applyDepthMode()
	gl.UseProgram(s.prog)
}
