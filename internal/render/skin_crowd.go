package render

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// SkinInstance matches CrowdInstance's std430 layout.
type SkinInstance struct {
	World, View, Projection M4
	Palette                 [4]int32
}

type SkinGroup struct {
	Mesh       *SkinMesh
	Texture    *Texture
	Reflection *Texture
	AlphaRef   float32
	Unlit      bool
	TwoSided   bool
	First      int
	Count      int
}

func (s *Scene3D) CrowdSkinInstance(world M4) SkinInstance {
	view := s.modelView(world)
	return SkinInstance{World: world, View: view, Projection: view.Mul(s.depthProjection(s.Proj))}
}

func (k *Skinner) uploadCrowdBuffer(i int, size int, data unsafe.Pointer) {
	if k.crowdBuffers[i] == 0 {
		gl.CreateBuffers(1, &k.crowdBuffers[i])
	}
	gl.NamedBufferData(k.crowdBuffers[i], size, data, gl.STREAM_DRAW)
	removeGPUBytes(k.crowdBytes[i])
	k.crowdBytes[i] = uint64(size)
	addGPUBytes(k.crowdBytes[i])
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, uint32(4+i), k.crowdBuffers[i])
}

func (k *Skinner) uploadCrowdInstances(instances []SkinInstance, palettes []M4) {
	k.uploadCrowdBuffer(0, len(instances)*int(unsafe.Sizeof(instances[0])), unsafe.Pointer(&instances[0]))
	if len(palettes) != 0 {
		k.uploadCrowdBuffer(1, len(palettes)*64, unsafe.Pointer(&palettes[0]))
	}
}

// DrawCrowd preserves independent skeletal palettes while sharing material and
// draw state. Callers queue only opaque or alpha-tested world characters.
func (k *Skinner) DrawCrowd(s *Scene3D, groups []SkinGroup, instances []SkinInstance, palettes []M4, ambient float32) {
	if len(instances) == 0 {
		return
	}
	k.Begin(s, ambient)
	k.uploadCrowdInstances(instances, palettes)
	gl.FrontFace(gl.CW)
	gl.CullFace(gl.BACK)
	for _, group := range groups {
		gl.Uniform1i(k.uCrowdBase, int32(group.First+1))
		gl.Uniform1f(k.uRef, group.AlphaRef)
		k.bindSkinReflection(group.Reflection)
		light := ambient
		if group.Unlit {
			light = 1
		}
		k.SetAmbient(light)
		if group.TwoSided {
			gl.Disable(gl.CULL_FACE)
		} else {
			gl.Enable(gl.CULL_FACE)
		}
		bindDiffuse(group.Texture)
		gl.BindVertexArray(group.Mesh.vao)
		gl.DrawElementsInstanced(gl.TRIANGLES, group.Mesh.Count, group.Mesh.indexType, nil, int32(group.Count))
		drawCalls++
	}
	gl.Uniform1i(k.uCrowdBase, 0)
	gl.Disable(gl.CULL_FACE)
	gl.FrontFace(gl.CCW)
}

// RigidInstanceVisible applies the ordinary model's per-mesh occlusion test
// before a rigid attachment is added to an instanced draw.
func (s *Scene3D) RigidInstanceVisible(mesh *Mesh, world M4) bool {
	return !s.meshOccluded(mesh, world)
}
