package render

import (
	"log"
	"math"
	"slices"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

const clothScatterCS = `#version 450 core
layout(local_size_x=64) in;
layout(std430, binding=6) writeonly buffer ClothPositions { vec4 positions[]; };
layout(std430, binding=7) readonly buffer ClothUpload { uvec4 uploadData[]; };
void main() {
	uvec4 range = uploadData[gl_WorkGroupID.x];
	for (uint i = gl_LocalInvocationID.x; i < range.z; i += gl_WorkGroupSize.x) {
		positions[range.x+i] = uintBitsToFloat(uploadData[range.y+i]);
	}
}`

const maxClothPositions = 1 << 20

type clothPlacement struct {
	World, View, Projection M4
	Offsets                 [4]int32
}
type clothMaterial struct {
	mesh    *Mesh
	texture *Texture
}
type clothGroup struct {
	material   clothMaterial
	placements []clothPlacement
}
type clothSlot struct {
	start   int
	version uint64
}
type clothSpan struct{ start, end int }

// ClothCrowd owns GPU storage for opaque cloth instances. Pose vertices remain
// resident until the actor's mesh version changes. Each camera supplies its own
// placements, with the same mesh IDs and UVs as the serial path.
type ClothCrowd struct {
	slots              map[uint64]clothSlot
	positions          [][4]float32
	dirty              []clothSpan
	groups             []clothGroup
	materials          map[clothMaterial]int
	packed             []clothPlacement
	scatter            [][4]uint32
	scatterProgram     uint32
	scatterUnavailable bool
	resetSlots         bool
	buffers            [2]uint32
	bytes              [2]int
	Uploaded           int
	UploadOps          int
}

func (b *ClothCrowd) Begin(actors int) {
	b.Uploaded, b.UploadOps = 0, 0
	if len(b.slots)+actors > 2048 || b.resetSlots {
		clear(b.slots)
		b.positions = b.positions[:0]
		b.resetSlots = false
	}
}

func (b *ClothCrowd) Queue(s *Scene3D, mesh *Mesh, texture *Texture, id, version uint64, vertices []Vertex, world M4) bool {
	if mesh.Count == 0 || len(vertices) == 0 {
		return true
	}
	if b.slots == nil {
		b.slots = make(map[uint64]clothSlot)
		b.materials = make(map[clothMaterial]int)
	}
	slot, cached := b.slots[id]
	if !cached {
		if len(b.positions)+len(vertices) > maxClothPositions {
			b.resetSlots = true
			return false
		}
		slot.start = len(b.positions)
		b.positions = append(b.positions, make([][4]float32, len(vertices))...)
	}
	if !cached || slot.version != version {
		slot.version = version
		b.slots[id] = slot
		for i, v := range vertices {
			b.positions[slot.start+i] = [4]float32{v.Pos[0], v.Pos[1], v.Pos[2], 1}
		}
		b.dirty = append(b.dirty, clothSpan{slot.start, slot.start + len(vertices)})
	}
	material := clothMaterial{mesh, texture}
	i, existed := b.materials[material]
	if !existed {
		i = len(b.materials)
		b.materials[material] = i
		if i == len(b.groups) {
			b.groups = append(b.groups, clothGroup{})
		}
		b.groups[i].material = material
	}
	view := s.modelView(world)
	group := &b.groups[i]
	group.placements = append(group.placements, clothPlacement{
		World: world, View: view, Projection: view.Mul(s.depthProjection(s.Proj)), Offsets: [4]int32{int32(slot.start)},
	})
	return true
}

func (b *ClothCrowd) bufferData(i, size int, data unsafe.Pointer) {
	if b.buffers[i] == 0 {
		gl.CreateBuffers(1, &b.buffers[i])
	}
	gl.NamedBufferData(b.buffers[i], size, data, gl.STREAM_DRAW)
	removeGPUBytes(uint64(b.bytes[i]))
	addGPUBytes(uint64(size))
	b.bytes[i] = size
}

func (b *ClothCrowd) uploadPositions(s *Scene3D) {
	size := len(b.positions) * 16
	if len(b.dirty) == 0 && size <= b.bytes[0] {
		return
	}
	changed := 0
	for _, span := range b.dirty {
		changed += span.end - span.start + 1 // Include the range header.
	}
	if size <= b.bytes[0] && changed*16 < size*3/4 && b.prepareScatter() {
		b.scatter = slices.Grow(b.scatter[:0], changed)[:len(b.dirty)]
		for i, span := range b.dirty {
			b.scatter[i] = [4]uint32{uint32(span.start), uint32(len(b.scatter)), uint32(span.end - span.start)}
			for _, p := range b.positions[span.start:span.end] {
				b.scatter = append(b.scatter, [4]uint32{math.Float32bits(p[0]), math.Float32bits(p[1]), math.Float32bits(p[2]), math.Float32bits(p[3])})
			}
		}
		bytes := len(b.scatter) * 16
		b.bufferData(1, bytes, unsafe.Pointer(&b.scatter[0]))
		gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 6, b.buffers[0])
		gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 7, b.buffers[1])
		// Wait for earlier shader reads before writing resident slots, then
		// publish the modified slots to the following vertex shader.
		gl.MemoryBarrier(gl.SHADER_STORAGE_BARRIER_BIT)
		gl.UseProgram(b.scatterProgram)
		gl.DispatchCompute(uint32(len(b.dirty)), 1, 1)
		gl.MemoryBarrier(gl.SHADER_STORAGE_BARRIER_BIT)
		gl.UseProgram(s.prog)
		b.Uploaded += bytes
	} else {
		b.bufferData(0, size, unsafe.Pointer(&b.positions[0]))
		b.Uploaded += size
	}
	b.UploadOps++
	b.dirty = b.dirty[:0]
}

func (b *ClothCrowd) prepareScatter() bool {
	if b.scatterProgram != 0 {
		return true
	}
	if b.scatterUnavailable {
		return false
	}
	var err error
	b.scatterProgram, err = computeProgram("cloth positions", clothScatterCS)
	if err != nil {
		log.Printf("Cloth compute upload unavailable: %v", err)
	}
	b.scatterUnavailable = err != nil
	return !b.scatterUnavailable
}

func (b *ClothCrowd) Draw(s *Scene3D) {
	count := len(b.materials)
	if count == 0 {
		return
	}
	b.uploadPositions(s)
	b.packed = b.packed[:0]
	for _, group := range b.groups[:count] {
		b.packed = append(b.packed, group.placements...)
	}
	b.bufferData(1, len(b.packed)*int(unsafe.Sizeof(b.packed[0])), unsafe.Pointer(&b.packed[0]))
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 6, b.buffers[0])
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 7, b.buffers[1])
	gl.Uniform1i(s.uMode, 0)
	gl.Uniform1i(s.uLit, 0)
	gl.Uniform1f(s.uRef, .5)
	gl.Uniform1i(s.uInstanced, 0)
	s.blend(false)
	first := 1
	for i := range count {
		group := &b.groups[i]
		mesh := group.material.mesh
		gl.Uniform1i(s.uClothBase, int32(first))
		bindDiffuse(group.material.texture)
		gl.BindVertexArray(mesh.vao)
		gl.DrawElementsInstanced(gl.TRIANGLES, mesh.Count, mesh.indexType, nil, int32(len(group.placements)))
		drawCalls++
		first += len(group.placements)
		group.placements = group.placements[:0]
		group.material = clothMaterial{}
	}
	gl.Uniform1i(s.uClothBase, 0)
	clear(b.materials)
}

func (b *ClothCrowd) Delete() {
	if b.scatterProgram != 0 {
		gl.DeleteProgram(b.scatterProgram)
	}
	for i, buffer := range b.buffers {
		if buffer != 0 {
			gl.DeleteBuffers(1, &b.buffers[i])
			removeGPUBytes(uint64(b.bytes[i]))
		}
	}
	*b = ClothCrowd{}
}
