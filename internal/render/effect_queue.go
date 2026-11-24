package render

import "github.com/Spelunkern/Shaiya-GX/internal/render/gl"

type queuedEffect struct {
	mesh       *Mesh
	texture    *Texture
	first, end int
	geometry   *EffectGeometry
	src, dst   uint32
	repeat     bool
}

type effectQueue struct {
	draws  []queuedEffect
	worlds []M4
	colors [][4]float32
	poses  []EffectPose
}

// Queue an ordered run. The shared upload is bounded even in a long
// transparent pass. Individual draw ranges retain their original material.
func (s *Scene3D) queueEffect(m *Mesh, texture *Texture, worlds []M4, colors [][4]float32, poses []EffectPose, geometry *EffectGeometry, src, dst uint32, repeat bool) bool {
	const maxInstances = 16384
	if !sceneStorageSupported || !s.effectPass.active || m == nil || m == s.effectStream || len(worlds) > maxInstances {
		return false
	}
	q := &s.effectQueue
	if len(q.worlds)+len(worlds) > maxInstances {
		s.flushEffectQueue()
	}
	first := len(q.worlds)
	q.worlds = append(q.worlds, worlds...)
	q.colors = append(q.colors, colors...)
	if len(poses) != 0 {
		q.poses = append(q.poses, poses...)
	} else {
		for range worlds {
			q.poses = append(q.poses, EffectPose{})
		}
	}
	if len(q.draws) > 0 {
		previous := &q.draws[len(q.draws)-1]
		if previous.mesh == m && previous.texture == texture && previous.geometry == geometry && previous.src == src && previous.dst == dst && previous.repeat == repeat {
			previous.end = len(q.worlds)
			return true
		}
	}
	q.draws = append(q.draws, queuedEffect{m, texture, first, len(q.worlds), geometry, src, dst, repeat})
	return true
}

func (s *Scene3D) flushEffectQueue() {
	q := &s.effectQueue
	if len(q.draws) == 0 {
		return
	}
	if !s.drawCompactEffects(q) {
		s.drawOriginalEffectQueue(q)
	}
	clear(q.draws)
	q.draws, q.worlds, q.colors, q.poses = q.draws[:0], q.worlds[:0], q.colors[:0], q.poses[:0]
}

func (s *Scene3D) drawOriginalEffectQueue(q *effectQueue) {
	poseOffset := s.uploadInstanceData(q.worlds, q.colors, q.poses)
	colorOffset := poseOffset - int32(len(q.colors))
	gl.Uniform1i(s.uInstanced, 1)
	gl.Uniform4f(s.uTint, 1, 1, 1, 1)
	var boundKeys uint32
	for _, draw := range q.draws {
		gl.Uniform1i(s.uInstanceColorOffset, colorOffset+int32(draw.first))
		var keys uint32
		if draw.geometry != nil {
			keys = draw.geometry.keys
		}
		if keys != 0 {
			if keys != boundKeys {
				gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 5, keys)
			}
			gl.Uniform1i(s.uEffectPoseOffset, poseOffset+int32(draw.first))
		} else if boundKeys != 0 {
			gl.Uniform1i(s.uEffectPoseOffset, 0)
		}
		boundKeys = keys
		s.effectMaterial(draw.texture, draw.src, draw.dst, draw.repeat)
		s.bindInstances(draw.mesh)
		gl.DrawElementsInstancedBaseInstance(gl.TRIANGLES, draw.mesh.Count, draw.mesh.indexType, nil, int32(draw.end-draw.first), uint32(draw.first))
		drawCalls++
	}
	if boundKeys != 0 {
		gl.Uniform1i(s.uEffectPoseOffset, 0)
	}
	gl.Uniform1i(s.uInstanceColorOffset, 0)
}
