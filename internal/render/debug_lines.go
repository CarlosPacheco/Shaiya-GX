package render

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"unsafe"
)

type DebugLineVertex struct {
	Position V3
	Color    [4]float32
}
type DebugLines struct {
	bytes                uint64
	program, vao, buffer uint32
	matrix               int32
}

func NewDebugLines() (*DebugLines, error) {
	program, err := Program("debug lines", `#version 450 core
layout(location=0) in vec3 position;
layout(location=1) in vec4 color;
uniform mat4 viewProjection;
out vec4 tint;
void main(){gl_Position=viewProjection*vec4(position,1);tint=color;}`, `#version 450 core
in vec4 tint;
out vec4 color;
void main(){color=tint;}`)
	if err != nil {
		return nil, err
	}
	d := &DebugLines{program: program, matrix: gl.GetUniformLocation(program, gl.Str("viewProjection\x00"))}
	gl.CreateVertexArrays(1, &d.vao)
	gl.CreateBuffers(1, &d.buffer)
	stride := int32(unsafe.Sizeof(DebugLineVertex{}))
	gl.VertexArrayVertexBuffer(d.vao, 0, d.buffer, 0, stride)
	gl.EnableVertexArrayAttrib(d.vao, 0)
	gl.VertexArrayAttribFormat(d.vao, 0, 3, gl.FLOAT, false, 0)
	gl.VertexArrayAttribBinding(d.vao, 0, 0)
	gl.EnableVertexArrayAttrib(d.vao, 1)
	gl.VertexArrayAttribFormat(d.vao, 1, 4, gl.FLOAT, false, 12)
	gl.VertexArrayAttribBinding(d.vao, 1, 0)
	return d, nil
}

// Draw is an x-ray diagnostic pass. It restores the raster/binding state and
// does not write depth, shadow, reflection or picking buffers.
func (d *DebugLines) Draw(scene *Scene3D, vertices []DebugLineVertex) {
	if d == nil || len(vertices) == 0 {
		return
	}
	depth, blend, cull, clip := gl.IsEnabled(gl.DEPTH_TEST), gl.IsEnabled(gl.BLEND), gl.IsEnabled(gl.CULL_FACE), gl.IsEnabled(gl.CLIP_DISTANCE0)
	var mask bool
	gl.GetBooleanv(gl.DEPTH_WRITEMASK, &mask)
	var previousProgram, previousVAO, srcRGB, dstRGB, srcAlpha, dstAlpha int32
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &previousProgram)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &previousVAO)
	gl.GetIntegerv(gl.BLEND_SRC_RGB, &srcRGB)
	gl.GetIntegerv(gl.BLEND_DST_RGB, &dstRGB)
	gl.GetIntegerv(gl.BLEND_SRC_ALPHA, &srcAlpha)
	gl.GetIntegerv(gl.BLEND_DST_ALPHA, &dstAlpha)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.CLIP_DISTANCE0)
	gl.DepthMask(false)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.UseProgram(d.program)
	matrix := scene.modelView(Identity()).Mul(scene.depthProjection(scene.Proj))
	gl.UniformMatrix4fv(d.matrix, 1, false, &matrix[0])
	size := uint64(len(vertices)) * uint64(unsafe.Sizeof(DebugLineVertex{}))
	if size != d.bytes {
		removeGPUBytes(d.bytes)
		addGPUBytes(size)
		d.bytes = size
	}
	gl.NamedBufferData(d.buffer, int(size), unsafe.Pointer(&vertices[0]), gl.STREAM_DRAW)
	gl.BindVertexArray(d.vao)
	gl.DrawArrays(gl.LINES, 0, int32(len(vertices)))
	drawCalls++
	gl.BindVertexArray(uint32(previousVAO))
	gl.UseProgram(uint32(previousProgram))
	gl.DepthMask(mask)
	gl.BlendFuncSeparate(uint32(srcRGB), uint32(dstRGB), uint32(srcAlpha), uint32(dstAlpha))
	restore := func(cap uint32, on bool) {
		if on {
			gl.Enable(cap)
		} else {
			gl.Disable(cap)
		}
	}
	restore(gl.DEPTH_TEST, depth)
	restore(gl.BLEND, blend)
	restore(gl.CULL_FACE, cull)
	restore(gl.CLIP_DISTANCE0, clip)
}
func (d *DebugLines) Delete() {
	if d == nil || d.program == 0 {
		return
	}
	removeGPUBytes(d.bytes)
	gl.DeleteBuffers(1, &d.buffer)
	gl.DeleteVertexArrays(1, &d.vao)
	gl.DeleteProgram(d.program)
	d.program, d.vao, d.buffer, d.bytes = 0, 0, 0, 0
}
