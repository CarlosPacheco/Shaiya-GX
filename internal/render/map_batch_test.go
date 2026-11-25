package render

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestMapBatchDefaultLimits(t *testing.T) {
	var b MapBatch
	if got := b.vertexLimit(); got != 128<<20 {
		t.Fatalf("vertexLimit() = %d, want %d", got, 128<<20)
	}
	if got := b.indexLimit(); got != 64<<20 {
		t.Fatalf("indexLimit() = %d, want %d", got, 64<<20)
	}
}

func TestMapBatchPartialFallbackOnOverflow(t *testing.T) {
	const indexBytes = 4
	const count = 3
	meshBytes := uint64(count*vertexSize + count*indexBytes)

	packed := make([]*Mesh, 5)
	for i := range packed {
		packed[i] = &Mesh{
			vbo:        uint32(i + 1),
			ebo:        uint32(i + 101),
			Count:      count,
			indexType:  gl.UNSIGNED_INT,
			indexBytes: indexBytes,
			bytes:      meshBytes,
			immutable:  true,
		}
	}
	overflow := &Mesh{
		vbo:        99,
		ebo:        199,
		Count:      count,
		indexType:  gl.UNSIGNED_INT,
		indexBytes: indexBytes,
		bytes:      uint64(100*vertexSize + count*indexBytes),
		immutable:  true,
	}
	mutable := &Mesh{
		vbo:        100,
		ebo:        200,
		Count:      count,
		indexType:  gl.UNSIGNED_INT,
		indexBytes: indexBytes,
		bytes:      meshBytes,
		immutable:  false,
	}
	tex := &Texture{
		ID: 1,
		samplingViews: [2]textureSamplingView{
			{id: 11, handle: 1001},
			{id: 12, handle: 1002},
		},
	}
	noHandleTex := &Texture{ID: 0}

	b := MapBatch{
		maxVertices: 10 * vertexSize,
		maxIndices:  1024,
		meshes:      make(map[*Mesh]skinLocation, len(packed)),
	}
	for i, m := range packed {
		b.meshes[m] = skinLocation{vertex: i * count, index: i * count * indexBytes}
	}
	b.vertices.used = len(packed) * count * vertexSize

	first := b.Placements([]uint32{0})
	b.Add(packed[0], tex, nil, -1, first, 1)
	b.Add(packed[1], tex, nil, -1, first, 1)
	b.Add(overflow, tex, nil, -1, first, 1)
	b.Add(packed[2], tex, nil, -1, first, 1)
	b.Add(mutable, tex, nil, -1, first, 1)
	b.Add(packed[3], noHandleTex, nil, -1, first, 1)
	b.Add(packed[4], tex, nil, -1, first, 1)

	if !b.prepare(true, false) {
		t.Fatal("prepare() returned false; expected packed draws to stay indirect")
	}
	if len(b.drawCommands) != 4 {
		t.Fatalf("drawCommands = %d, want 4", len(b.drawCommands))
	}
	if len(b.runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(b.runs))
	}
	if len(b.serialDraws) != 3 {
		t.Fatalf("serialDraws = %d, want 3", len(b.serialDraws))
	}
	if b.serialDraws[0].mesh != overflow || b.serialDraws[1].mesh != mutable || b.serialDraws[2].mesh != packed[3] {
		t.Fatalf("unexpected serial fallback draws: %+v", b.serialDraws)
	}
}

func TestEffectIndirectPartialFallbackPlan(t *testing.T) {
	const indexBytes = 4
	const count = 6
	meshBytes := uint64(4*vertexSize + count*indexBytes)

	cached := &Mesh{
		vbo:        1,
		ebo:        11,
		Count:      count,
		indexType:  gl.UNSIGNED_INT,
		indexBytes: indexBytes,
		bytes:      meshBytes,
		immutable:  true,
	}
	overflow := &Mesh{
		vbo:        2,
		ebo:        12,
		Count:      count,
		indexType:  gl.UNSIGNED_INT,
		indexBytes: indexBytes,
		bytes:      uint64(100*vertexSize + count*indexBytes),
		immutable:  true,
	}
	tex := &Texture{
		ID: 1,
		samplingViews: [2]textureSamplingView{
			{id: 21, handle: 2001},
			{id: 22, handle: 2002},
		},
	}

	p := effectIndirect{
		program:   1,
		maxMeshes: 1,
		geometry: map[*Mesh]effectIndirectLocation{
			cached: {vertex: 0, index: 0},
		},
	}
	draws := []queuedEffect{
		{mesh: cached, texture: tex, src: gl.SRC_ALPHA, dst: gl.ONE_MINUS_SRC_ALPHA, first: 0, end: 1},
		{mesh: cached, texture: tex, src: gl.SRC_ALPHA, dst: gl.ONE_MINUS_SRC_ALPHA, first: 1, end: 2},
		{mesh: overflow, texture: tex, src: gl.SRC_ALPHA, dst: gl.ONE_MINUS_SRC_ALPHA, first: 2, end: 3},
		{mesh: cached, texture: tex, src: gl.SRC_ALPHA, dst: gl.ONE_MINUS_SRC_ALPHA, first: 3, end: 4},
		{mesh: cached, texture: tex, src: gl.SRC_ALPHA, dst: gl.ONE_MINUS_SRC_ALPHA, first: 4, end: 5},
	}
	if !p.plan(draws) {
		t.Fatal("plan() returned false; expected ordered partial indirect runs")
	}
	if len(p.commands) != 4 {
		t.Fatalf("commands = %d, want 4", len(p.commands))
	}
	if len(p.runs) != 3 {
		t.Fatalf("runs = %d, want 3 (indirect, serial, indirect)", len(p.runs))
	}
	if p.runs[0].serial != -1 || p.runs[0].first != 0 || p.runs[0].count != 2 {
		t.Fatalf("run[0] = %+v, want indirect first=0 count=2", p.runs[0])
	}
	if p.runs[1].serial != 2 {
		t.Fatalf("run[1] = %+v, want serial=2", p.runs[1])
	}
	if p.runs[2].serial != -1 || p.runs[2].first != 2 || p.runs[2].count != 2 {
		t.Fatalf("run[2] = %+v, want indirect first=2 count=2", p.runs[2])
	}
}
