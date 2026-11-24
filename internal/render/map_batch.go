package render

import (
	"log"
	"math"
	"unsafe"

	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

// MapBatch retains immutable geometry and placements for one map. Each camera
// uploads only its ordered draw commands, material handles and placement IDs.
type MapBatch struct {
	Transforms []M4

	vao, boundVertex, boundIndex    uint32
	vertices, indices               packedMeshBuffer
	meshes                          map[*Mesh]skinLocation
	placements, matrices            effectUpload
	references, commands, materials effectUpload
	uploaded                        bool

	draws         []mapDraw
	visible       []uint32
	drawCommands  []indexedDrawCommand
	drawMaterials [][5]uint32
	runs          []mapDrawRun
	fallback      []M4
}

type mapDraw struct {
	mesh          *Mesh
	tex, lightmap *Texture
	first, count  int
	alpha         float32
}

type mapDrawRun struct {
	first int
	index uint32
	cull  bool
}

type mapPipeline struct {
	draw, compute uint32
	initialized   bool
}

func (b *MapBatch) Placements(ids []uint32) int {
	first := len(b.visible)
	b.visible = append(b.visible, ids...)
	return first
}

func (b *MapBatch) Add(mesh *Mesh, tex, lightmap *Texture, alpha float32, first, count int) {
	if mesh == nil || mesh.Count == 0 || count <= 0 {
		return
	}
	if n := len(b.draws); n > 0 {
		last := &b.draws[n-1]
		if last.mesh == mesh && last.tex == tex && last.lightmap == lightmap &&
			last.alpha == alpha && last.first+last.count == first {
			last.count += count
			return
		}
	}
	b.draws = append(b.draws, mapDraw{mesh, tex, lightmap, first, count, alpha})
}

func (b *MapBatch) location(m *Mesh) (skinLocation, bool) {
	if loc, ok := b.meshes[m]; ok {
		return loc, m.vbo != 0
	}
	const maxVertices, maxIndices = 40 << 20, 20 << 20
	indexBytes := int(m.Count) * int(m.indexBytes)
	vertexBytes := int(m.bytes) - indexBytes
	at := (b.indices.used + 3) &^ 3
	if !m.immutable || m.vbo == 0 || indexBytes <= 0 || vertexBytes <= 0 ||
		b.vertices.used+vertexBytes > maxVertices || at+indexBytes > maxIndices {
		return skinLocation{}, false
	}
	if b.meshes == nil {
		b.meshes = make(map[*Mesh]skinLocation)
	}
	loc := skinLocation{b.vertices.used / vertexSize, at}
	b.vertices.copy(m.vbo, b.vertices.used, vertexBytes, maxVertices)
	b.indices.copy(m.ebo, at, indexBytes, maxIndices)
	b.meshes[m] = loc
	return loc, true
}

func (b *MapBatch) prepare() bool {
	b.drawCommands, b.drawMaterials, b.runs = b.drawCommands[:0], b.drawMaterials[:0], b.runs[:0]
	for _, draw := range b.draws {
		loc, ok := b.location(draw.mesh)
		if !ok {
			return false
		}
		t := diffuseTexture(draw.tex)
		diffuse := t.bindlessHandle(t.repeat)
		var light uint64
		if draw.lightmap != nil {
			light = draw.lightmap.bindlessHandle(draw.lightmap.repeat)
		}
		if diffuse == 0 || draw.lightmap != nil && light == 0 {
			return false
		}
		m := draw.mesh
		if n := len(b.runs); n == 0 || b.runs[n-1].index != m.indexType || b.runs[n-1].cull != m.CullBack {
			b.runs = append(b.runs, mapDrawRun{len(b.drawCommands), m.indexType, m.CullBack})
		}
		b.drawCommands = append(b.drawCommands, indexedDrawCommand{uint32(m.Count), uint32(draw.count), uint32(loc.index / int(m.indexBytes)), uint32(loc.vertex), uint32(draw.first)})
		b.drawMaterials = append(b.drawMaterials, [5]uint32{uint32(diffuse), uint32(diffuse >> 32), uint32(light), uint32(light >> 32), math.Float32bits(draw.alpha)})
	}
	return len(b.drawCommands) > len(b.runs)
}

func (s *Scene3D) mapPrograms() *mapPipeline {
	p := &s.mapPipeline
	if p.initialized {
		return p
	}
	p.initialized = true
	if !skinIndirectSupported {
		return p
	}
	var err error
	p.draw, err = Program("map bindless indirect", mapVS, mapFS)
	if err == nil {
		p.compute, err = computeProgram("map transforms", mapCS)
	}
	if err != nil {
		log.Printf("map batch shaders: %v", err)
		gl.DeleteProgram(p.draw)
		p.draw = 0
	}
	return p
}

func (b *MapBatch) Draw(s *Scene3D) {
	if len(b.draws) == 0 {
		b.visible = b.visible[:0]
		return
	}
	p := s.mapPrograms()
	if s.Shadow || p.draw == 0 || len(b.draws) < 4 || !b.prepare() {
		for _, d := range b.draws {
			if d.lightmap != nil {
				s.drawLightmapped(d.mesh, d.tex, d.lightmap, true)
			} else if d.count == 1 {
				s.DrawVisibleWorldModel(d.mesh, d.tex, b.Transforms[b.visible[d.first]], d.alpha)
			} else {
				b.fallback = b.fallback[:0]
				for _, id := range b.visible[d.first : d.first+d.count] {
					b.fallback = append(b.fallback, b.Transforms[id])
				}
				s.SetInstances(b.fallback)
				s.DrawWorldInstanced(d.mesh, d.tex, d.alpha)
			}
		}
	} else {
		b.drawIndirect(s, p)
	}
	clear(b.draws)
	b.draws, b.visible = b.draws[:0], b.visible[:0]
}

func (b *MapBatch) drawIndirect(s *Scene3D, p *mapPipeline) {
	if b.vao == 0 {
		gl.CreateVertexArrays(1, &b.vao)
	}
	if b.boundVertex != b.vertices.id || b.boundIndex != b.indices.id {
		bindMeshLayout(b.vao, b.vertices.id, b.indices.id)
		b.boundVertex, b.boundIndex = b.vertices.id, b.indices.id
	}
	if !b.uploaded {
		b.placements.upload(len(b.Transforms)*64, unsafe.Pointer(&b.Transforms[0]))
		b.uploaded = true
	}
	b.references.upload(len(b.visible)*4, unsafe.Pointer(&b.visible[0]))
	b.commands.upload(len(b.drawCommands)*indexedDrawStride, unsafe.Pointer(&b.drawCommands[0]))
	b.materials.upload(len(b.drawMaterials)*20, unsafe.Pointer(&b.drawMaterials[0]))
	if size := len(b.visible) * 192; uint64(size) > b.matrices.bytes {
		b.matrices.upload(max(size, int(b.matrices.bytes)*2), nil)
	}

	gl.UseProgram(p.compute)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 0, b.placements.id)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 1, b.references.id)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 2, b.matrices.id)
	gl.UniformMatrix4fv(0, 1, false, &s.View[0])
	gl.UniformMatrix4fv(4, 1, false, &s.projection[0])
	gl.Uniform3fv(8, 1, &s.cameraEye[0])
	relative := int32(0)
	if s.cameraValid && s.cameraView == s.View {
		relative = 1
	}
	gl.Uniform1i(9, relative)
	gl.Uniform1ui(10, uint32(len(b.visible)))
	gl.DispatchCompute(uint32((len(b.visible)+63)/64), 1, 1)
	gl.MemoryBarrier(gl.SHADER_STORAGE_BARRIER_BIT)

	gl.UseProgram(p.draw)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 3, b.materials.id)
	gl.Uniform4fv(0, 1, &s.ClipPlane[0])
	gl.Uniform3fv(1, 1, &s.LightDir[0])
	gl.Uniform1f(2, s.Ambient)
	gl.Uniform3fv(3, 1, &s.FogColor[0])
	gl.Uniform2f(4, s.FogStart, s.FogEnd)
	gl.Disable(gl.BLEND)
	gl.BindVertexArray(b.vao)
	gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, b.commands.id)
	gl.FrontFace(gl.CW)
	gl.CullFace(gl.BACK)
	for i, run := range b.runs {
		if run.cull {
			gl.Enable(gl.CULL_FACE)
		} else {
			gl.Disable(gl.CULL_FACE)
		}
		end := len(b.drawCommands)
		if i+1 < len(b.runs) {
			end = b.runs[i+1].first
		}
		gl.Uniform1i(6, int32(run.first))
		gl.MultiDrawElementsIndirectWithOffset(gl.TRIANGLES, run.index, uintptr(run.first*indexedDrawStride), int32(end-run.first), 0)
		drawCalls++
	}
	gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, 0)
	gl.Disable(gl.CULL_FACE)
	gl.FrontFace(gl.CCW)
	gl.UseProgram(s.prog)
	s.worldUniformsValid = false
}

// Reclaim dead meshes on residency changes. Retain packed copies of surviving
// resources until the discarded space is worth rebuilding.
func (b *MapBatch) Trim() {
	dead := 0
	for m := range b.meshes {
		if m.vbo == 0 {
			dead++
		}
	}
	if dead > 0 && (dead*4 >= len(b.meshes) || b.vertices.used > 36<<20 || b.indices.used > 18<<20) {
		b.releaseGeometry()
	}
}

func (b *MapBatch) releaseGeometry() {
	if b.vao != 0 {
		gl.DeleteVertexArrays(1, &b.vao)
		b.vao, b.boundVertex, b.boundIndex = 0, 0, 0
	}
	b.vertices.delete()
	b.indices.delete()
	b.meshes = nil
}

func (b *MapBatch) Delete() {
	b.releaseGeometry()
	for _, buf := range []*effectUpload{&b.placements, &b.references, &b.matrices, &b.commands, &b.materials} {
		buf.delete()
	}
	*b = MapBatch{}
}

const mapCS = `#version 450 core
layout(local_size_x=64) in;
struct Pose { mat4 world; mat4 mv; mat4 mvp; };
layout(std430,binding=0) readonly buffer Worlds { mat4 worlds[]; };
layout(std430,binding=1) readonly buffer Visible { uint ids[]; };
layout(std430,binding=2) writeonly buffer Poses { Pose poses[]; };
layout(location=0) uniform mat4 view;
layout(location=4) uniform mat4 projection;
layout(location=8) uniform vec3 eye;
layout(location=9) uniform int relative;
layout(location=10) uniform uint count;
void main() {
 uint i=gl_GlobalInvocationID.x;
 if (i>=count) return;
 mat4 world=worlds[ids[i]];
 precise mat4 mv=view*world;
 if (relative != 0) {
  dvec3 delta=dvec3(world[3].xyz)-dvec3(eye);
  for (int j=0; j<3; j++)
   mv[3][j]=float(delta.x*double(view[0][j])+delta.y*double(view[1][j])+delta.z*double(view[2][j]));
 }
 precise mat4 mvp=projection*mv;
 poses[i]=Pose(world,mv,mvp);
}`

const mapVS = `#version 450 core
#extension GL_ARB_shader_draw_parameters : require
layout(location=0) in vec3 pos;
layout(location=1) in vec3 nrm;
layout(location=2) in vec2 uv;
layout(location=3) in vec2 uv2;
layout(location=0) uniform vec4 clipPlane;
layout(location=1) uniform vec3 lightDir;
layout(location=2) uniform float ambient;
layout(location=6) uniform int firstDraw;
struct Pose { mat4 world; mat4 mv; mat4 mvp; };
layout(std430,binding=2) readonly buffer Poses { Pose poses[]; };
out vec2 vUV, vUV2;
out float vDepth, vDiffuse;
flat out int material;
void main() {
 Pose p=poses[gl_BaseInstanceARB+gl_InstanceID];
 vec4 local=vec4(pos,1);
 gl_Position=p.mvp*local;
 gl_ClipDistance[0]=dot(p.world*local,clipPlane);
 vDepth=(p.mv*local).z;
 vDiffuse=clamp(ambient+max(0.0,dot(normalize((p.world*vec4(nrm,0)).xyz),-normalize(lightDir))),0.0,1.0);
 vUV=uv; vUV2=uv2;
 material=firstDraw+gl_DrawIDARB;
}`

const mapFS = `#version 450 core
#extension GL_ARB_bindless_texture : require
layout(location=3) uniform vec3 fogColor;
layout(location=4) uniform vec2 fogRange;
layout(std430,binding=3) readonly buffer Materials { uint data[]; };
in vec2 vUV, vUV2;
in float vDepth, vDiffuse;
flat in int material;
out vec4 frag;
void main() {
 int at=material*5;
 vec4 c=texture(sampler2D(uvec2(data[at],data[at+1])),vUV);
 if (c.a <= uintBitsToFloat(data[at+4])) discard;
 uvec2 light=uvec2(data[at+2],data[at+3]);
 if (any(notEqual(light,uvec2(0)))) c.rgb*=texture(sampler2D(light),vUV2).rgb;
 else c.rgb*=vDiffuse;
 if (fogRange.y > fogRange.x) {
  float f=clamp((fogRange.y-vDepth)/(fogRange.y-fogRange.x),0.0,1.0);
  c.rgb=mix(fogColor,c.rgb,f);
 }
 frag=c;
}`
