package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"sync/atomic"
)

// The source frames can be sampled by workers. Only the GL thread writes the
// shared draw scratch; motion-path births read one vertex from the source.
type effectGeometry struct {
	id       uint64
	source   *effect.Mesh3DE
	vertices []render.Vertex
	indices  []uint32
}

var effectGeometryIDs atomic.Uint64

func newEffectGeometry(m *effect.Mesh3DE) *effectGeometry {
	m.Trailing = nil // The native loader ignores bytes after the first object.
	g := &effectGeometry{id: effectGeometryIDs.Add(1), source: m, vertices: make([]render.Vertex, len(m.Vertices)), indices: make([]uint32, len(m.Indices))}
	for i, v := range m.Vertices {
		g.vertices[i] = render.Vertex{Pos: render.V3(v.Position), UV: v.UV}
	}
	for i, v := range m.Indices {
		g.indices[i] = uint32(v)
	}
	return g
}

// 0x557470 uses the default rate code 5 set at 0x55cf20: 30 keys/second.
// Unknown8 is read as the loop flag. Crossing the maximum resets to zero,
// discarding overshoot; a nonlooping clock holds the declared maximum.
func advanceMeshFrame(frame float32, maximum int32, loop bool, dt float32) float32 {
	frame += dt * 30
	if frame > float32(maximum) {
		if loop {
			return 0
		}
		return float32(maximum)
	}
	return frame
}

// 0x5575e0 interpolates complete position/UV snapshots. Its key search uses
// a two-key lookbehind window, including extrapolation for densely spaced keys.
func (g *effectGeometry) verticesAt(frame float32) []render.Vertex {
	if len(g.source.Frames) == 0 {
		return g.vertices
	}
	first, second, t := g.framesAt(frame)
	a, b := &g.source.Frames[first], &g.source.Frames[second]
	for i := range g.vertices {
		av, bv := a.Vertices[i], b.Vertices[i]
		g.vertices[i].Pos = interpolate3DEPosition(av.Position, bv.Position, t)
		for j := range av.UV {
			g.vertices[i].UV[j] = av.UV[j] + (bv.UV[j]-av.UV[j])*t
		}
	}
	return g.vertices
}

func (g *effectGeometry) framesAt(frame float32) (first, second int, t float32) {
	fs := g.source.Frames
	index := 0
	for index < len(fs)-1 && frame-2 >= float32(fs[index].Key) {
		index++
	}
	if frame < float32(fs[index].Key) {
		index = max(0, index-1)
	}
	first, second = index, index
	a := &fs[index]
	if frame < float32(fs[len(fs)-1].Key) && index+1 < len(fs) {
		second = index + 1
		b := &fs[second]
		if b.Key > a.Key {
			t = (frame - float32(a.Key)) / float32(b.Key-a.Key)
		}
	}
	return first, second, t
}

func interpolate3DEPosition(a, b [3]float32, t float32) render.V3 {
	return render.V3{
		a[0] + (b[0]-a[0])*t,
		a[1] + (b[1]-a[1])*t,
		a[2] + (b[2]-a[2])*t,
	}
}

func (g *effectGeometry) keyVertexData() []render.EffectKeyVertex {
	keys := make([]render.EffectKeyVertex, 0, len(g.source.Frames)*len(g.vertices))
	for _, frame := range g.source.Frames {
		for _, v := range frame.Vertices {
			keys = append(keys, render.EffectKeyVertex{Position: [4]float32{v.Position[0], v.Position[1], v.Position[2]}, UV: [4]float32{v.UV[0], v.UV[1]}})
		}
	}
	return keys
}
