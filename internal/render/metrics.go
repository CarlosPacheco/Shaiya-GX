package render

import "github.com/Spelunkern/Shaiya-GX/internal/render/gl"

// Counters belong to the GL thread. VRAM estimates owned image/buffer storage;
// driver overhead and the window-system framebuffer are deliberately excluded.
var gpuBytes uint64
var gpuCapacity uint64
var drawCalls int

func addGPUBytes(n uint64)    { gpuBytes += n }
func removeGPUBytes(n uint64) { gpuBytes -= min(gpuBytes, n) }
func GPUBytes() uint64        { return gpuBytes }
func GPUCapacity() uint64     { return gpuCapacity }
func ResetDrawCalls()         { drawCalls = 0 }
func DrawCalls() int          { return drawCalls }

// FrameMeter uses timestamp pairs and only reads queries reported available.
// A busy ring skips timing instead of waiting for the GPU.
type FrameMeter struct {
	queries [4][2]uint32
	pending [4]bool
	active  int
	totalMS float64
	samples int
}

func NewFrameMeter() *FrameMeter {
	m := &FrameMeter{active: -1}
	for i := range m.queries {
		gl.CreateQueries(gl.TIMESTAMP, 2, &m.queries[i][0])
	}
	return m
}
func (m *FrameMeter) Begin() {
	m.active = -1
	for i := range m.queries {
		if !m.pending[i] {
			continue
		}
		var available int32
		gl.GetQueryObjectiv(m.queries[i][1], gl.QUERY_RESULT_AVAILABLE, &available)
		if available == 0 {
			continue
		}
		var start, end uint64
		gl.GetQueryObjectui64v(m.queries[i][0], gl.QUERY_RESULT, &start)
		gl.GetQueryObjectui64v(m.queries[i][1], gl.QUERY_RESULT, &end)
		if end >= start {
			m.totalMS += float64(end-start) / 1e6
			m.samples++
		}
		m.pending[i] = false
	}
	for i := range m.queries {
		if !m.pending[i] {
			m.active = i
			gl.QueryCounter(m.queries[i][0], gl.TIMESTAMP)
			return
		}
	}
}
func (m *FrameMeter) End() {
	if m.active < 0 {
		return
	}
	i := m.active
	gl.QueryCounter(m.queries[i][1], gl.TIMESTAMP)
	m.pending[i] = true
	m.active = -1
}

// TakeTimingSamples returns and clears the completed GPU timings since the last call.
func (m *FrameMeter) TakeTimingSamples() (totalMS float64, samples int) {
	if m == nil {
		return 0, 0
	}
	totalMS, samples = m.totalMS, m.samples
	m.totalMS, m.samples = 0, 0
	return totalMS, samples
}
func (m *FrameMeter) Delete() {
	if m == nil {
		return
	}
	for i := range m.queries {
		gl.DeleteQueries(2, &m.queries[i][0])
	}
}
