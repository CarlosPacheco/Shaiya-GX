package render

import (
	"cmp"
	"log"
	"math"
	"unsafe"

	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

const (
	defaultMapMaxVertices = 128 << 20
	defaultMapMaxIndices  = 64 << 20
)

// MapBatch retains immutable geometry and placements for one map. Each camera
// uploads only its ordered draw commands, materials and placement IDs.
type MapBatch struct {
	Transforms []M4

	vao                                     uint32
	boundVertexCapacity, boundIndexCapacity int
	maxVertices, maxIndices                 int
	vertices, indices                       packedMeshBuffer
	meshes                                  map[*Mesh]skinLocation
	placements, matrices                    effectUpload
	references, commands, materials         effectUpload
	uploaded                                bool

	draws         []mapDraw
	serialDraws   []mapDraw
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
	unlit         bool
}

type mapDrawRun struct {
	first    int
	count    int
	serial   int
	index    uint32
	cull     bool
	textures drawTextures
}

type mapPipeline struct {
	draw, bound, compute uint32
	initialized          bool
	materialAttribute    bool
}

func (b *MapBatch) Placements(ids []uint32) int {
	first := len(b.visible)
	b.visible = append(b.visible, ids...)
	return first
}

// ResetTransforms marks b.Transforms dirty so the next indirect draw re-uploads
// the placement buffer. Used for per-frame dynamic placement lists such as
// depth-sorted translucent VANI foliage.
func (b *MapBatch) ResetTransforms() {
	b.uploaded = false
}

func (b *MapBatch) Add(mesh *Mesh, tex, lightmap *Texture, alpha float32, first, count int) {
	b.add(mesh, tex, lightmap, alpha, false, first, count)
}

// AddTranslucent queues a blended world-model draw (alpha test disabled,
// two-sided) for DrawTranslucent while preserving exact submission order.
func (b *MapBatch) AddTranslucent(mesh *Mesh, tex *Texture, lit bool, first, count int) {
	b.add(mesh, tex, nil, -1, !lit, first, count)
}

func (b *MapBatch) add(mesh *Mesh, tex, lightmap *Texture, alpha float32, unlit bool, first, count int) {
	if mesh == nil || mesh.Count == 0 || count <= 0 {
		return
	}
	if n := len(b.draws); n > 0 {
		last := &b.draws[n-1]
		if last.mesh == mesh && last.tex == tex && last.lightmap == lightmap &&
			last.alpha == alpha && last.unlit == unlit && last.first+last.count == first {
			last.count += count
			return
		}
	}
	b.draws = append(b.draws, mapDraw{mesh, tex, lightmap, first, count, alpha, unlit})
}

func (b *MapBatch) vertexLimit() int {
	return cmp.Or(b.maxVertices, defaultMapMaxVertices)
}

func (b *MapBatch) indexLimit() int {
	return cmp.Or(b.maxIndices, defaultMapMaxIndices)
}

func (b *MapBatch) location(m *Mesh) (skinLocation, bool) {
	if loc, ok := b.meshes[m]; ok {
		return loc, m.vbo != 0
	}
	maxVertices, maxIndices := b.vertexLimit(), b.indexLimit()
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

func (b *MapBatch) prepare(bindless, materialAttribute bool) bool {
	return b.prepareMode(bindless, materialAttribute, false)
}

func (b *MapBatch) prepareMode(bindless, materialAttribute, translucent bool) bool {
	b.drawCommands, b.drawMaterials, b.runs, b.serialDraws = b.drawCommands[:0], b.drawMaterials[:0], b.runs[:0], b.serialDraws[:0]
	for i, draw := range b.draws {
		loc, ok := b.location(draw.mesh)
		if !ok {
			if translucent {
				b.newSerialRun(i)
			} else {
				b.serialDraws = append(b.serialDraws, draw)
			}
			continue
		}
		t := diffuseTexture(draw.tex)
		m := draw.mesh
		var diffuse, light uint64
		if bindless {
			diffuse = t.bindlessHandle(t.repeat)
			if draw.lightmap != nil {
				light = draw.lightmap.bindlessHandle(draw.lightmap.repeat)
			}
			if diffuse == 0 || draw.lightmap != nil && light == 0 {
				if translucent {
					b.newSerialRun(i)
				} else {
					b.serialDraws = append(b.serialDraws, draw)
				}
				continue
			}
		}
		cull := m.CullBack && !translucent
		if n := len(b.runs); n == 0 || b.runs[n-1].serial >= 0 || b.runs[n-1].index != m.indexType || b.runs[n-1].cull != cull {
			b.newRun(m.indexType, cull)
		}
		if !bindless {
			run := &b.runs[len(b.runs)-1]
			diffuse, light, ok = run.textureSlots(t, draw.lightmap)
			if !ok {
				b.newRun(m.indexType, cull)
				diffuse, light, _ = b.runs[len(b.runs)-1].textureSlots(t, draw.lightmap)
			}
		}
		if draw.unlit && draw.lightmap == nil {
			light = ^uint64(0)
		}
		command := indexedDrawCommand{uint32(m.Count), uint32(draw.count), uint32(loc.index / int(m.indexBytes)), uint32(loc.vertex), uint32(draw.first)}
		alpha := math.Float32bits(draw.alpha)
		material := [5]uint32{uint32(diffuse), uint32(diffuse >> 32), uint32(light), uint32(light >> 32), alpha}
		if materialAttribute {
			material = [5]uint32{uint32(diffuse), uint32(light), alpha, command.baseInstance}
			command.baseInstance = uint32(len(b.drawMaterials))
		}
		b.drawCommands = append(b.drawCommands, command)
		b.drawMaterials = append(b.drawMaterials, material)
		b.runs[len(b.runs)-1].count++
	}
	if translucent {
		return len(b.drawCommands) > 0 && len(b.runs) < len(b.draws)
	}
	return len(b.drawCommands) > len(b.runs)
}

func (b *MapBatch) newRun(index uint32, cull bool) {
	if len(b.runs) < cap(b.runs) {
		b.runs = b.runs[:len(b.runs)+1]
		run := &b.runs[len(b.runs)-1]
		run.first, run.count, run.serial, run.index, run.cull = len(b.drawCommands), 0, -1, index, cull
		run.textures = run.textures[:0]
		return
	}
	b.runs = append(b.runs, mapDrawRun{first: len(b.drawCommands), serial: -1, index: index, cull: cull})
}

func (b *MapBatch) newSerialRun(drawIndex int) {
	if len(b.runs) < cap(b.runs) {
		b.runs = b.runs[:len(b.runs)+1]
		run := &b.runs[len(b.runs)-1]
		run.first, run.count, run.serial, run.index, run.cull = 0, 0, drawIndex, 0, false
		run.textures = run.textures[:0]
		return
	}
	b.runs = append(b.runs, mapDrawRun{serial: drawIndex})
}

func (r *mapDrawRun) textureSlots(diffuse, light *Texture) (uint64, uint64, bool) {
	textures := r.textures
	d, ok := textures.slot(diffuse.ID, drawTextureLimit)
	if !ok {
		return 0, 0, false
	}
	if light == nil {
		r.textures = textures
		return uint64(d), 0, true
	}
	l, ok := textures.slot(light.ID, drawTextureLimit)
	if !ok {
		return 0, 0, false
	}
	r.textures = textures
	// Zero reserves the existing material field for the no-lightmap case.
	return uint64(d), uint64(l + 1), true
}

func (s *Scene3D) mapPrograms() *mapPipeline {
	p := &s.mapPipeline
	if p.initialized {
		return p
	}
	p.initialized = true
	if !mapIndirectSupported {
		return p
	}
	var err error
	p.compute, err = computeProgram("map transforms", mapCS)
	if err != nil {
		log.Printf("map batch shaders: %v", err)
		return p
	}
	if skinIndirectSupported {
		p.draw, err = Program("map bindless indirect", mapVS, mapFS)
		if err != nil {
			log.Printf("map bindless shader: %v", err)
		}
	}
	if p.draw == 0 && drawParametersSupported {
		p.bound, err = Program("map bound indirect", mapVS, mapBoundFS)
		if err != nil {
			log.Printf("map bound shader: %v", err)
		} else {
			log.Printf("OpenGL map draw path: indirect with bound textures")
		}
	}
	if p.draw == 0 && p.bound == 0 {
		const header = "#version 450 core\n#define DRAW_MATERIAL_ATTRIBUTE\n"
		p.bound, err = Program("map OpenGL 4.5 indirect", header+mapVertex, header+drawMaterialFragment+mapFragment)
		if err != nil {
			log.Printf("map OpenGL 4.5 shader: %v", err)
		} else {
			p.materialAttribute = true
			log.Printf("OpenGL map draw path: indirect with instanced materials")
		}
	}
	return p
}

func (b *MapBatch) Draw(s *Scene3D) {
	if len(b.draws) == 0 {
		b.visible = b.visible[:0]
		return
	}
	p := s.mapPrograms()
	if s.Shadow || (p.draw == 0 && p.bound == 0) || len(b.draws) < 4 || !b.prepare(p.draw != 0, p.materialAttribute) {
		b.drawSerial(s, b.draws)
	} else {
		b.drawIndirect(s, p, false)
		b.drawSerial(s, b.serialDraws)
	}
	clear(b.draws)
	clear(b.serialDraws)
	b.draws, b.serialDraws, b.visible = b.draws[:0], b.serialDraws[:0], b.visible[:0]
}

// DrawTranslucent renders queued translucent world models in exact submission
// order with source-alpha blending, read-only depth testing and two-sided faces,
// matching BeginTranslucentWorldModels / DrawTranslucentWorldModel.
func (b *MapBatch) DrawTranslucent(s *Scene3D) {
	if len(b.draws) == 0 || s.Shadow {
		clear(b.draws)
		b.draws, b.visible = b.draws[:0], b.visible[:0]
		return
	}
	s.BeginTranslucentWorldModels()
	p := s.mapPrograms()
	if (p.draw == 0 && p.bound == 0) || len(b.draws) < 2 || !b.prepareMode(p.draw != 0, p.materialAttribute, true) {
		for _, d := range b.draws {
			b.drawSerialTranslucentOne(s, d)
		}
	} else {
		b.drawIndirect(s, p, true)
	}
	s.EndTranslucentWorldModels()
	clear(b.draws)
	clear(b.serialDraws)
	b.draws, b.serialDraws, b.visible = b.draws[:0], b.serialDraws[:0], b.visible[:0]
}

func (b *MapBatch) drawSerialTranslucentOne(s *Scene3D, d mapDraw) {
	b.fallback = b.fallback[:0]
	for _, id := range b.visible[d.first : d.first+d.count] {
		b.fallback = append(b.fallback, b.Transforms[id])
	}
	s.DrawTranslucentWorldModel(d.mesh, d.tex, b.fallback, !d.unlit)
}

func (b *MapBatch) drawSerial(s *Scene3D, draws []mapDraw) {
	for _, d := range draws {
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
}

func (b *MapBatch) drawIndirect(s *Scene3D, p *mapPipeline, translucent bool) {
	if b.vao == 0 {
		gl.CreateVertexArrays(1, &b.vao)
	}
	if b.boundVertexCapacity != b.vertices.capacity || b.boundIndexCapacity != b.indices.capacity {
		bindMeshLayout(b.vao, b.vertices.id, b.indices.id)
		b.boundVertexCapacity, b.boundIndexCapacity = b.vertices.capacity, b.indices.capacity
	}
	if !b.uploaded {
		b.placements.upload(len(b.Transforms)*64, unsafe.Pointer(&b.Transforms[0]))
		b.uploaded = true
	}
	b.references.upload(len(b.visible)*4, unsafe.Pointer(&b.visible[0]))
	b.commands.upload(len(b.drawCommands)*indexedDrawStride, unsafe.Pointer(&b.drawCommands[0]))
	const materialStride = int(unsafe.Sizeof([5]uint32{}))
	b.materials.upload(len(b.drawMaterials)*materialStride, unsafe.Pointer(&b.drawMaterials[0]))
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

	program := p.draw
	if program == 0 {
		program = p.bound
	}
	bindIndirect := func() {
		gl.UseProgram(program)
		if p.materialAttribute {
			bindDrawMaterial(b.vao, b.materials.id, int32(materialStride))
		} else {
			gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 3, b.materials.id)
		}
		gl.Uniform4fv(0, 1, &s.ClipPlane[0])
		gl.Uniform3fv(1, 1, &s.LightDir[0])
		gl.Uniform1f(2, s.Ambient)
		gl.Uniform3fv(3, 1, &s.FogColor[0])
		gl.Uniform2f(4, s.FogStart, s.FogEnd)
		if translucent {
			s.blend(true)
			gl.DepthMask(false)
			gl.Disable(gl.CULL_FACE)
		} else {
			gl.Disable(gl.BLEND)
			gl.FrontFace(gl.CW)
			gl.CullFace(gl.BACK)
		}
		gl.BindVertexArray(b.vao)
		gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, b.commands.id)
	}
	indirectBound := false
	for _, run := range b.runs {
		if run.serial >= 0 {
			if indirectBound {
				gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, 0)
				gl.UseProgram(s.prog)
				s.worldUniformsValid = false
				indirectBound = false
			} else {
				gl.UseProgram(s.prog)
				s.worldUniformsValid = false
			}
			b.drawSerialTranslucentOne(s, b.draws[run.serial])
			continue
		}
		if !indirectBound {
			bindIndirect()
			indirectBound = true
		}
		if !translucent {
			if run.cull {
				gl.Enable(gl.CULL_FACE)
			} else {
				gl.Disable(gl.CULL_FACE)
			}
		}
		if !p.materialAttribute {
			gl.Uniform1i(6, int32(run.first))
		}
		run.textures.bind()
		gl.MultiDrawElementsIndirectWithOffset(gl.TRIANGLES, run.index, uintptr(run.first*indexedDrawStride), int32(run.count), 0)
		drawCalls++
	}
	if indirectBound {
		gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, 0)
		if !translucent {
			gl.Disable(gl.CULL_FACE)
			gl.FrontFace(gl.CCW)
		}
		gl.UseProgram(s.prog)
		s.worldUniformsValid = false
	}
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
	if dead > 0 && (dead*4 >= len(b.meshes) || b.vertices.used > b.vertexLimit()*9/10 || b.indices.used > b.indexLimit()*9/10) {
		b.releaseGeometry()
	}
}

func (b *MapBatch) releaseGeometry() {
	if b.vao != 0 {
		gl.DeleteVertexArrays(1, &b.vao)
		b.vao, b.boundVertexCapacity, b.boundIndexCapacity = 0, 0, 0
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
` + mapVertex

const mapVertex = `layout(location=0) in vec3 pos;
layout(location=1) in vec3 nrm;
layout(location=2) in vec2 uv;
layout(location=3) in vec2 uv2;
layout(location=0) uniform vec4 clipPlane;
layout(location=1) uniform vec3 lightDir;
layout(location=2) uniform float ambient;
#ifdef DRAW_MATERIAL_ATTRIBUTE
layout(location=4) in uvec4 drawMaterial;
flat out uvec3 material;
#else
layout(location=6) uniform int firstDraw;
flat out int material;
#endif
struct Pose { mat4 world; mat4 mv; mat4 mvp; };
layout(std430,binding=2) readonly buffer Poses { Pose poses[]; };
out vec2 vUV, vUV2;
out float vDepth, vDiffuse;
void main() {
#ifdef DRAW_MATERIAL_ATTRIBUTE
 Pose p=poses[int(drawMaterial.w)+gl_InstanceID];
 material=drawMaterial.xyz;
#else
 Pose p=poses[gl_BaseInstanceARB+gl_InstanceID];
 material=firstDraw+gl_DrawIDARB;
#endif
 vec4 local=vec4(pos,1);
 gl_Position=p.mvp*local;
 gl_ClipDistance[0]=dot(p.world*local,clipPlane);
 vDepth=(p.mv*local).z;
 vDiffuse=clamp(ambient+max(0.0,dot(normalize((p.world*vec4(nrm,0)).xyz),-normalize(lightDir))),0.0,1.0);
 vUV=uv; vUV2=uv2;
}`

const mapFS = `#version 450 core
#extension GL_ARB_bindless_texture : require
` + mapFragment

const mapBoundFS = `#version 450 core
#define MAP_BOUND
layout(binding=16) uniform sampler2D mapTextures[16];
` + mapFragment

const mapFragment = `
layout(location=3) uniform vec3 fogColor;
layout(location=4) uniform vec2 fogRange;
#ifdef DRAW_MATERIAL_ATTRIBUTE
flat in uvec3 material;
#else
layout(std430,binding=3) readonly buffer Materials { uint data[]; };
flat in int material;
#endif
in vec2 vUV, vUV2;
in float vDepth, vDiffuse;
out vec4 frag;
void main() {
#ifdef DRAW_MATERIAL_ATTRIBUTE
 vec4 c=sampleDrawTexture(material.x,vUV);
 vec3 lighting=vec3(vDiffuse);
 if (material.y==0xffffffffu) lighting=vec3(1.0);
 else if (material.y!=0u) lighting=sampleDrawTexture(material.y-1u,vUV2).rgb;
 if (c.a <= uintBitsToFloat(material.z)) discard;
 c.rgb*=lighting;
#else
 int at=material*5;
#ifdef MAP_BOUND
 vec4 c=texture(mapTextures[data[at]],vUV);
 if (c.a <= uintBitsToFloat(data[at+4])) discard;
 uint light=data[at+2];
 if (light==0xffffffffu) {}
 else if (light!=0u) c.rgb*=texture(mapTextures[light-1u],vUV2).rgb;
 else c.rgb*=vDiffuse;
#else
 vec4 c=texture(sampler2D(uvec2(data[at],data[at+1])),vUV);
 if (c.a <= uintBitsToFloat(data[at+4])) discard;
 uvec2 light=uvec2(data[at+2],data[at+3]);
 if (all(equal(light,uvec2(0xffffffffu)))) {}
 else if (any(notEqual(light,uvec2(0)))) c.rgb*=texture(sampler2D(light),vUV2).rgb;
 else c.rgb*=vDiffuse;
#endif
#endif
 if (fogRange.y > fogRange.x) {
  float f=clamp((fogRange.y-vDepth)/(fogRange.y-fogRange.x),0.0,1.0);
  c.rgb=mix(fogColor,c.rgb,f);
 }
 frag=c;
}`
