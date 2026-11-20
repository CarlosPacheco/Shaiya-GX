package render

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// EffectKeyVertex stores the original position and UV of one 3DE key vertex.
// Two vec4s keep the same std430 stride in Go and GLSL.
type EffectKeyVertex struct {
	Position [4]float32
	UV       [4]float32
}

// EffectPose selects two exact mesh key ranges and the authored interpolation.
type EffectPose [4]float32

type EffectGeometry struct {
	mesh  *Mesh
	keys  uint32
	bytes uint64
}

// EffectGeometryCache owns GPU topology and key vertices for one asset owner.
// Once full, other geometries retain the ordinary sampled-stream path.
type EffectGeometryCache struct {
	entries map[uint64]*EffectGeometry
	bytes   uint64
}

func (c *EffectGeometryCache) Geometry(id uint64, vertices []Vertex, indices []uint32, keyVertices int, sample func() []EffectKeyVertex) *EffectGeometry {
	const byteLimit = 32 << 20
	const maxGeometries = 256
	if c == nil || len(vertices) == 0 || len(indices) == 0 {
		return nil
	}
	if g, ok := c.entries[id]; ok {
		return g
	}
	if len(c.entries) >= maxGeometries {
		return nil
	}
	keyBytes := keyVertices * int(unsafe.Sizeof(EffectKeyVertex{}))
	if c.bytes+uint64(keyBytes+len(vertices)*vertexSize+len(indices)*4) > byteLimit {
		return nil
	}
	g := &EffectGeometry{mesh: NewMesh(vertices, indices), bytes: uint64(keyBytes)}
	if keyVertices != 0 {
		keys := sample()
		gl.CreateBuffers(1, &g.keys)
		gl.NamedBufferStorage(g.keys, keyBytes, unsafe.Pointer(&keys[0]), 0)
		addGPUBytes(g.bytes)
		// The authored topology is shared by poses with different extents.
		// Let the GPU depth test their exact interpolated positions.
		g.mesh.boundsValid = false
	}
	if c.entries == nil {
		c.entries = make(map[uint64]*EffectGeometry)
	}
	c.entries[id] = g
	c.bytes += g.mesh.bytes + g.bytes
	return g
}

func (c *EffectGeometryCache) Delete() {
	for _, g := range c.entries {
		g.mesh.Delete()
		if g.keys != 0 {
			gl.DeleteBuffers(1, &g.keys)
			removeGPUBytes(g.bytes)
		}
	}
	*c = EffectGeometryCache{}
}

func (s *Scene3D) DrawEffectGeometry(g *EffectGeometry, t *Texture, worlds []M4, colors [][4]float32, poses []EffectPose, source, destination int32) {
	if g.keys == 0 {
		if len(worlds) == 1 {
			s.DrawEffect(g.mesh, t, worlds[0], colors[0], source, destination)
		} else {
			s.DrawEffectBatch(g.mesh, t, worlds, colors, source, destination)
		}
		return
	}
	src, dst, ok := EFTBlend(source, destination)
	if !ok || t == nil || len(worlds) == 0 || len(poses) != len(worlds) || len(colors) != len(worlds) {
		return
	}
	if s.queueEffect(g.mesh, t, worlds, colors, poses, g, src, dst, true) {
		return
	}
	s.flushEffectQueue()
	poseOffset := s.uploadInstanceData(worlds, colors, poses)
	gl.Uniform1i(s.uEffectPoseOffset, poseOffset)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 5, g.keys)
	gl.Uniform1i(s.uInstanced, 1)
	gl.Uniform4f(s.uTint, 1, 1, 1, 1)
	s.effectMaterial(t, src, dst, true)
	s.drawInstances(g.mesh)
	if !s.effectPass.active {
		t.SetWrap(false)
	}
	gl.Uniform1i(s.uEffectPoseOffset, 0)
	gl.Uniform1i(s.uInstanceColorOffset, 0)
}
